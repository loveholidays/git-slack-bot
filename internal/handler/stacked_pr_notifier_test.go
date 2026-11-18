package handler_test

import (
	"errors"
	"git-slack-bot/internal/config"
	mock_github "git-slack-bot/internal/github/mocks"
	"git-slack-bot/internal/handler"
	bot_slack "git-slack-bot/internal/slack"
	mock_slack "git-slack-bot/internal/slack/mocks"
	mock_user "git-slack-bot/internal/user/mocks"

	gh "github.com/google/go-github/v93/github"
	. "github.com/onsi/ginkgo/v2"
	"github.com/slack-go/slack"
	"go.uber.org/mock/gomock"
)

var _ = Describe("Stacked PR notifier", func() {
	var (
		githubMock  *mock_github.MockInteractor
		slackMock   *mock_slack.MockInteractor
		userMock    *mock_user.MockService
		notifier    handler.PullRequestNotifier
		pullRequest *gh.PullRequest
		review      *gh.PullRequestReview
		message     *slack.Message
		messageKey  string
	)

	BeforeEach(func() {
		controller := gomock.NewController(GinkgoT())
		githubMock = mock_github.NewMockInteractor(controller)
		slackMock = mock_slack.NewMockInteractor(controller)
		userMock = mock_user.NewMockService(controller)
		emojis := config.EmojiConfiguration{Approve: "approved-custom", Merge: "merged-custom", Close: "closed-custom"}
		notifier = handler.NewStackedPRNotifier(githubMock, userMock, emojis)
		pullRequest = &gh.PullRequest{
			Number:  new(42),
			HTMLURL: new("https://github.com/loveholidays/flux/pull/42"),
			State:   new("open"),
			Stack:   &gh.PullRequestStack{ID: new(int64(123)), Number: new(7), Position: new(2)},
			Base:    &gh.PullRequestBranch{Repo: &gh.Repository{Name: new("flux"), Owner: &gh.User{Login: new("loveholidays")}}},
		}
		review = &gh.PullRequestReview{
			State: new("approved"), Body: new("Looks good"),
			HTMLURL: new("https://github.com/loveholidays/flux/pull/42#pullrequestreview-1"),
			User:    &gh.User{Login: new("reviewer")},
		}
		messageKey = "PR Stack 123:"
		message = &slack.Message{Msg: slack.Msg{
			Timestamp: "123.456",
			Text:      "PR Stack 123:\n- <https://github.com/loveholidays/flux/pull/50|First PR>\n- <https://github.com/loveholidays/flux/pull/42|Second PR>",
		}}
	})

	Context("NotifyPullRequestOpened", func() {
		It("should rebuild in API order and preserve approvals with terminal statuses taking precedence", func() {
			message.Text = "PR Stack 123:\n- <https://github.com/loveholidays/flux/pull/40|Merged PR> :approved-custom:\n- <https://github.com/loveholidays/flux/pull/42|Second PR> :approved-custom:\n- <https://github.com/loveholidays/flux/pull/50|First PR> :approved-custom:"
			stack := &gh.PullRequestStackDetails{
				ID: 123, Number: 7,
				PullRequests: []*gh.PullRequestStackEntry{
					{Number: 50, Title: "First & <important>\nPR", HTMLURL: "https://github.com/loveholidays/flux/pull/50", State: "open"},
					{Number: 51, Title: "Draft PR", Draft: true},
					{Number: 42, Title: "Second PR", HTMLURL: pullRequest.GetHTMLURL(), State: "closed"},
					{Number: 40, Title: "Merged PR", HTMLURL: "https://github.com/loveholidays/flux/pull/40", MergedAt: &gh.Timestamp{}},
				},
			}
			expected := bot_slack.Notification{
				Key:  messageKey,
				Text: "PR Stack 123:\n- <https://github.com/loveholidays/flux/pull/50|First &amp; &lt;important&gt; PR> :approved-custom:\n- <https://github.com/loveholidays/flux/pull/42|Second PR> :closed-custom:\n- <https://github.com/loveholidays/flux/pull/40|Merged PR> :merged-custom:",
			}
			githubMock.EXPECT().GetStack("loveholidays", "flux", 7).Return(stack, nil)
			slackMock.EXPECT().GetMessage(messageKey).Return(message, nil)
			slackMock.EXPECT().SendOrUpdateMessage(expected)

			notifier.NotifyPullRequestOpened(slackMock, pullRequest)
		})

		It("should create the first stack message without an emoji for an open PR", func() {
			stack := &gh.PullRequestStackDetails{
				ID: 123, Number: 7,
				PullRequests: []*gh.PullRequestStackEntry{{Number: 42, Title: "Second PR", HTMLURL: pullRequest.GetHTMLURL(), State: "open"}},
			}
			expected := bot_slack.Notification{Key: messageKey, Text: "PR Stack 123:\n- <https://github.com/loveholidays/flux/pull/42|Second PR>"}
			githubMock.EXPECT().GetStack("loveholidays", "flux", 7).Return(stack, nil)
			slackMock.EXPECT().GetMessage(messageKey).Return(nil, bot_slack.ErrMessageNotFound)
			slackMock.EXPECT().SendOrUpdateMessage(expected)

			notifier.NotifyPullRequestOpened(slackMock, pullRequest)
		})

		It("should not overwrite approval state when Slack history is unavailable", func() {
			stack := &gh.PullRequestStackDetails{ID: 123, Number: 7}
			lookupError := errors.New("Slack unavailable")
			githubMock.EXPECT().GetStack("loveholidays", "flux", 7).Return(stack, nil)
			slackMock.EXPECT().GetMessage(messageKey).Return(nil, lookupError)

			notifier.NotifyPullRequestOpened(slackMock, pullRequest)
		})

		It("should stop when stack retrieval fails", func() {
			lookupError := errors.New("GitHub unavailable")
			githubMock.EXPECT().GetStack("loveholidays", "flux", 7).Return(nil, lookupError)

			notifier.NotifyPullRequestOpened(slackMock, pullRequest)
		})

		It("should stop when the stack query returns no usable stack", func() {
			githubMock.EXPECT().GetStack("loveholidays", "flux", 7).Return(nil, nil)

			notifier.NotifyPullRequestOpened(slackMock, pullRequest)
		})
	})

	Context("NotifyPullRequestClosed", func() {
		It("should update only the closed PR status", func() {
			pullRequest.State = new("closed")
			message.Text += " :approved-custom:"
			expected := bot_slack.Notification{Key: messageKey, Text: "PR Stack 123:\n- <https://github.com/loveholidays/flux/pull/50|First PR>\n- <https://github.com/loveholidays/flux/pull/42|Second PR> :closed-custom:"}
			slackMock.EXPECT().GetMessage(messageKey).Return(message, nil)
			slackMock.EXPECT().SendOrUpdateMessage(expected)

			notifier.NotifyPullRequestClosed(slackMock, pullRequest)
		})

		It("should mark a merged PR without adding a reaction to the whole stack", func() {
			pullRequest.State = new("closed")
			pullRequest.MergedAt = &gh.Timestamp{}
			message.Text += " :approved-custom:"
			expected := bot_slack.Notification{Key: messageKey, Text: "PR Stack 123:\n- <https://github.com/loveholidays/flux/pull/50|First PR>\n- <https://github.com/loveholidays/flux/pull/42|Second PR> :merged-custom:"}
			slackMock.EXPECT().GetMessage(messageKey).Return(message, nil)
			slackMock.EXPECT().SendOrUpdateMessage(expected)

			notifier.NotifyPullRequestClosed(slackMock, pullRequest)
		})

		It("should ignore a draft PR", func() {
			pullRequest.Draft = new(true)

			notifier.NotifyPullRequestClosed(slackMock, pullRequest)
		})

		It("should stop when the stack message cannot be found", func() {
			lookupError := errors.New("thread unavailable")
			slackMock.EXPECT().GetMessage(messageKey).Return(nil, lookupError)

			notifier.NotifyPullRequestClosed(slackMock, pullRequest)
		})
	})

	Context("NotifyPullRequestReopened", func() {
		It("should restore the PR status to open", func() {
			message.Text = "PR Stack 123:\n- <https://github.com/loveholidays/flux/pull/50|First PR> :merged-custom:\n- <https://github.com/loveholidays/flux/pull/42|Second PR> :closed-custom:"
			expected := bot_slack.Notification{Key: messageKey, Text: "PR Stack 123:\n- <https://github.com/loveholidays/flux/pull/50|First PR> :merged-custom:\n- <https://github.com/loveholidays/flux/pull/42|Second PR>"}
			slackMock.EXPECT().GetMessage(messageKey).Return(message, nil)
			slackMock.EXPECT().SendOrUpdateMessage(expected)

			notifier.NotifyPullRequestReopened(slackMock, pullRequest)
		})

		It("should stop when the stack message cannot be found", func() {
			lookupError := errors.New("thread unavailable")
			slackMock.EXPECT().GetMessage(messageKey).Return(nil, lookupError)

			notifier.NotifyPullRequestReopened(slackMock, pullRequest)
		})
	})

	Context("NotifyReviewSubmitted", func() {
		It("should identify the PR and approval in a reply with the review body", func() {
			updated := bot_slack.Notification{Key: messageKey, Text: message.Text + " :approved-custom:"}
			expected := "<@reviewer> left a <https://github.com/loveholidays/flux/pull/42#pullrequestreview-1|approved review> for <https://github.com/loveholidays/flux/pull/42|PR #42>.\nLooks good"
			slackMock.EXPECT().GetMessage(messageKey).Return(message, nil)
			slackMock.EXPECT().SendOrUpdateMessage(updated)
			userMock.EXPECT().GetUserDescriptor("reviewer").Return("<@reviewer>")
			slackMock.EXPECT().SendReply(message, expected)

			notifier.NotifyReviewSubmitted(slackMock, pullRequest, review)
		})

		It("should announce an approval without a body", func() {
			review.Body = new("")
			message.Text += " :approved-custom:"
			updated := bot_slack.Notification{Key: messageKey, Text: message.Text}
			expected := "<@reviewer> left a <https://github.com/loveholidays/flux/pull/42#pullrequestreview-1|approved review> for <https://github.com/loveholidays/flux/pull/42|PR #42>."
			slackMock.EXPECT().GetMessage(messageKey).Return(message, nil)
			slackMock.EXPECT().SendOrUpdateMessage(updated)
			userMock.EXPECT().GetUserDescriptor("reviewer").Return("<@reviewer>")
			slackMock.EXPECT().SendReply(message, expected)

			notifier.NotifyReviewSubmitted(slackMock, pullRequest, review)
		})

		It("should identify changes requested in the reply", func() {
			review.State = new("changes_requested")
			review.Body = new("Please update this")
			expected := "<@reviewer> left a <https://github.com/loveholidays/flux/pull/42#pullrequestreview-1|changes requested review> for <https://github.com/loveholidays/flux/pull/42|PR #42>.\nPlease update this"
			slackMock.EXPECT().GetMessage(messageKey).Return(message, nil)
			userMock.EXPECT().GetUserDescriptor("reviewer").Return("<@reviewer>")
			slackMock.EXPECT().SendReply(message, expected)

			notifier.NotifyReviewSubmitted(slackMock, pullRequest, review)
		})

		It("should keep only the close emoji when an approval arrives after closure", func() {
			pullRequest.State = new("closed")
			updated := bot_slack.Notification{Key: messageKey, Text: message.Text + " :closed-custom:"}
			expected := "<@reviewer> left a <https://github.com/loveholidays/flux/pull/42#pullrequestreview-1|approved review> for <https://github.com/loveholidays/flux/pull/42|PR #42>.\nLooks good"
			slackMock.EXPECT().GetMessage(messageKey).Return(message, nil)
			slackMock.EXPECT().SendOrUpdateMessage(updated)
			userMock.EXPECT().GetUserDescriptor("reviewer").Return("<@reviewer>")
			slackMock.EXPECT().SendReply(message, expected)

			notifier.NotifyReviewSubmitted(slackMock, pullRequest, review)
		})

		It("should keep only the merge emoji when an approval arrives after merging", func() {
			pullRequest.State = new("closed")
			pullRequest.MergedAt = &gh.Timestamp{}
			updated := bot_slack.Notification{Key: messageKey, Text: message.Text + " :merged-custom:"}
			expected := "<@reviewer> left a <https://github.com/loveholidays/flux/pull/42#pullrequestreview-1|approved review> for <https://github.com/loveholidays/flux/pull/42|PR #42>.\nLooks good"
			slackMock.EXPECT().GetMessage(messageKey).Return(message, nil)
			slackMock.EXPECT().SendOrUpdateMessage(updated)
			userMock.EXPECT().GetUserDescriptor("reviewer").Return("<@reviewer>")
			slackMock.EXPECT().SendReply(message, expected)

			notifier.NotifyReviewSubmitted(slackMock, pullRequest, review)
		})

		It("should stop when the stack message cannot be found", func() {
			lookupError := errors.New("thread unavailable")
			slackMock.EXPECT().GetMessage(messageKey).Return(nil, lookupError)

			notifier.NotifyReviewSubmitted(slackMock, pullRequest, review)
		})
	})

	Context("NotifyReviewDismissed", func() {
		It("should identify the PR whose review was dismissed", func() {
			updated := bot_slack.Notification{Key: messageKey, Text: message.Text}
			message.Text += " :approved-custom:"
			expected := "Review dismissed for <https://github.com/loveholidays/flux/pull/42|PR #42>."
			slackMock.EXPECT().GetMessage(messageKey).Return(message, nil)
			slackMock.EXPECT().SendOrUpdateMessage(updated)
			slackMock.EXPECT().SendReply(message, expected)

			notifier.NotifyReviewDismissed(slackMock, pullRequest)
		})

		It("should stop when the stack message cannot be found", func() {
			lookupError := errors.New("thread unavailable")
			slackMock.EXPECT().GetMessage(messageKey).Return(nil, lookupError)

			notifier.NotifyReviewDismissed(slackMock, pullRequest)
		})
	})

	Context("NotifyComment", func() {
		It("should identify the PR and preserve the supplied comment", func() {
			comment := "<@reviewer> left a comment:\nPlease check this"
			expected := "<https://github.com/loveholidays/flux/pull/42|PR #42>:\n" + comment
			slackMock.EXPECT().GetMessage(messageKey).Return(message, nil)
			slackMock.EXPECT().SendReply(message, expected)

			notifier.NotifyComment(slackMock, pullRequest, comment)
		})

		It("should stop when the stack message cannot be found", func() {
			lookupError := errors.New("thread unavailable")
			slackMock.EXPECT().GetMessage(messageKey).Return(nil, lookupError)

			notifier.NotifyComment(slackMock, pullRequest, "A comment")
		})
	})
})
