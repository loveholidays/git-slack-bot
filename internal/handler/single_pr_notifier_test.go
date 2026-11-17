package handler_test

import (
	"errors"
	"git-slack-bot/internal/config"
	"git-slack-bot/internal/handler"
	bot_slack "git-slack-bot/internal/slack"
	mock_slack "git-slack-bot/internal/slack/mocks"
	mock_user "git-slack-bot/internal/user/mocks"
	"time"

	gh "github.com/google/go-github/v56/github"
	. "github.com/onsi/ginkgo/v2"
	"github.com/slack-go/slack"
	"go.uber.org/mock/gomock"
)

var _ = Describe("Single PR notifier", func() {
	var (
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
		slackMock = mock_slack.NewMockInteractor(controller)
		userMock = mock_user.NewMockService(controller)
		emojis := config.EmojiConfiguration{Approve: "approved-custom", Merge: "merged-custom", Close: "closed-custom"}
		notifier = handler.NewSinglePRNotifier(userMock, emojis)
		createdAt := gh.Timestamp{Time: time.Date(2026, time.November, 17, 12, 0, 0, 0, time.UTC)}
		pullRequest = &gh.PullRequest{
			Title:     new("Update deployment"),
			HTMLURL:   new("https://github.com/loveholidays/flux/pull/42"),
			User:      &gh.User{Login: new("author")},
			State:     new("open"),
			CreatedAt: &createdAt,
		}
		review = &gh.PullRequestReview{
			State:   new("approved"),
			Body:    new("Looks good"),
			HTMLURL: new("https://github.com/loveholidays/flux/pull/42#pullrequestreview-1"),
			User:    &gh.User{Login: new("reviewer")},
		}
		message = &slack.Message{Msg: slack.Msg{Timestamp: "123.456"}}
		messageKey = "<https://github.com/loveholidays/flux/pull/42>"
	})

	Context("NotifyPullRequestOpened", func() {
		It("should send the PR message with its lookup key and creation time", func() {
			expected := bot_slack.Notification{
				Key:       messageKey,
				Text:      "<@author> Update deployment:\nhttps://github.com/loveholidays/flux/pull/42",
				CreatedAt: pullRequest.GetCreatedAt().Time,
			}
			userMock.EXPECT().GetUserDescriptor("author").Return("<@author>")
			slackMock.EXPECT().SendMessageOnce(expected)

			notifier.NotifyPullRequestOpened(slackMock, pullRequest)
		})

		It("should not notify for a draft PR", func() {
			pullRequest.Draft = new(true)

			notifier.NotifyPullRequestOpened(slackMock, pullRequest)
		})

		It("should not notify for a closed PR", func() {
			pullRequest.State = new("closed")

			notifier.NotifyPullRequestOpened(slackMock, pullRequest)
		})
	})

	Context("NotifyPullRequestClosed", func() {
		It("should add the close reaction when closed without merging", func() {
			pullRequest.State = new("closed")
			slackMock.EXPECT().GetMessage(messageKey).Return(message, nil)
			slackMock.EXPECT().AddReactionToMessage("closed-custom", message)

			notifier.NotifyPullRequestClosed(slackMock, pullRequest)
		})

		It("should add the merge reaction when merged", func() {
			pullRequest.State = new("closed")
			pullRequest.MergedAt = pullRequest.CreatedAt
			slackMock.EXPECT().GetMessage(messageKey).Return(message, nil)
			slackMock.EXPECT().AddReactionToMessage("merged-custom", message)

			notifier.NotifyPullRequestClosed(slackMock, pullRequest)
		})

		It("should ignore a draft PR", func() {
			pullRequest.Draft = new(true)

			notifier.NotifyPullRequestClosed(slackMock, pullRequest)
		})

		It("should stop when the original Slack message cannot be found", func() {
			lookupError := errors.New("thread unavailable")
			slackMock.EXPECT().GetMessage(messageKey).Return(nil, lookupError)

			notifier.NotifyPullRequestClosed(slackMock, pullRequest)
		})
	})

	Context("NotifyPullRequestReopened", func() {
		It("should remove the close reaction from the original message", func() {
			slackMock.EXPECT().GetMessage(messageKey).Return(message, nil)
			slackMock.EXPECT().RemoveReactionFromMessage("x", message)

			notifier.NotifyPullRequestReopened(slackMock, pullRequest)
		})

		It("should stop when the original Slack message cannot be found", func() {
			lookupError := errors.New("thread unavailable")
			slackMock.EXPECT().GetMessage(messageKey).Return(nil, lookupError)

			notifier.NotifyPullRequestReopened(slackMock, pullRequest)
		})
	})

	Context("NotifyReviewSubmitted", func() {
		It("should add the approval reaction without a reply for an approval without a body", func() {
			review.Body = new("")
			slackMock.EXPECT().GetMessage(messageKey).Return(message, nil)
			slackMock.EXPECT().AddReactionToMessage("approved-custom", message)

			notifier.NotifyReviewSubmitted(slackMock, pullRequest, review)
		})

		It("should add the approval reaction and reply for an approval with a body", func() {
			expected := "<@reviewer> left a <https://github.com/loveholidays/flux/pull/42#pullrequestreview-1|review>:\nLooks good"
			slackMock.EXPECT().GetMessage(messageKey).Return(message, nil)
			slackMock.EXPECT().AddReactionToMessage("approved-custom", message)
			userMock.EXPECT().GetUserDescriptor("reviewer").Return("<@reviewer>")
			slackMock.EXPECT().SendReply(message, expected)

			notifier.NotifyReviewSubmitted(slackMock, pullRequest, review)
		})

		It("should reply without an approval reaction when changes are requested", func() {
			review.State = new("changes_requested")
			review.Body = new("Please update this")
			expected := "<@reviewer> left a <https://github.com/loveholidays/flux/pull/42#pullrequestreview-1|review>:\nPlease update this"
			slackMock.EXPECT().GetMessage(messageKey).Return(message, nil)
			userMock.EXPECT().GetUserDescriptor("reviewer").Return("<@reviewer>")
			slackMock.EXPECT().SendReply(message, expected)

			notifier.NotifyReviewSubmitted(slackMock, pullRequest, review)
		})

		It("should not react or reply for a commented review without a body", func() {
			review.State = new("commented")
			review.Body = new("")
			slackMock.EXPECT().GetMessage(messageKey).Return(message, nil)

			notifier.NotifyReviewSubmitted(slackMock, pullRequest, review)
		})

		It("should stop when the original Slack message cannot be found", func() {
			lookupError := errors.New("thread unavailable")
			slackMock.EXPECT().GetMessage(messageKey).Return(nil, lookupError)

			notifier.NotifyReviewSubmitted(slackMock, pullRequest, review)
		})
	})

	Context("NotifyReviewDismissed", func() {
		It("should remove the configured approval reaction from the original message", func() {
			slackMock.EXPECT().GetMessage(messageKey).Return(message, nil)
			slackMock.EXPECT().RemoveReactionFromMessage("approved-custom", message)

			notifier.NotifyReviewDismissed(slackMock, pullRequest)
		})

		It("should stop when the original Slack message cannot be found", func() {
			lookupError := errors.New("thread unavailable")
			slackMock.EXPECT().GetMessage(messageKey).Return(nil, lookupError)

			notifier.NotifyReviewDismissed(slackMock, pullRequest)
		})
	})

	Context("NotifyComment", func() {
		It("should send the supplied comment to the original thread", func() {
			comment := "<@reviewer> left a comment:\nPlease check this"
			slackMock.EXPECT().GetMessage(messageKey).Return(message, nil)
			slackMock.EXPECT().SendReply(message, comment)

			notifier.NotifyComment(slackMock, pullRequest, comment)
		})

		It("should stop when the original Slack message cannot be found", func() {
			lookupError := errors.New("thread unavailable")
			slackMock.EXPECT().GetMessage(messageKey).Return(nil, lookupError)

			notifier.NotifyComment(slackMock, pullRequest, "A comment")
		})
	})
})
