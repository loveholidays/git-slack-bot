package handler_test

import (
	"errors"
	"git-slack-bot/internal/config"
	mock_github "git-slack-bot/internal/github/mocks"
	"git-slack-bot/internal/handler"
	mock_handler "git-slack-bot/internal/handler/mocks"
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
		githubMock   *mock_github.MockInteractor
		slackMock    *mock_slack.MockInteractor
		builder      *mock_handler.MockStackMessageBuilder
		users        *mock_user.MockService
		notifier     handler.PullRequestNotifier
		pr           *gh.PullRequest
		review       *gh.PullRequestReview
		stack        *gh.PullRequestStackDetails
		message      *slack.Message
		emoji        config.EmojiConfiguration
		notification bot_slack.Notification
	)
	BeforeEach(func() {
		controller := gomock.NewController(GinkgoT())
		githubMock = mock_github.NewMockInteractor(controller)
		slackMock = mock_slack.NewMockInteractor(controller)
		builder = mock_handler.NewMockStackMessageBuilder(controller)
		users = mock_user.NewMockService(controller)
		emoji = config.EmojiConfiguration{Approve: "approved-custom", Merge: "merged-custom", Close: "closed-custom"}
		notifier = handler.NewStackedPRNotifier(githubMock, builder, users, emoji)
		pr = &gh.PullRequest{Number: new(42), HTMLURL: new("https://github.com/loveholidays/flux/pull/42"), Stack: &gh.PullRequestStack{ID: new(int64(123)), Number: new(7)}, Base: &gh.PullRequestBranch{Repo: &gh.Repository{Name: new("flux"), Owner: &gh.User{Login: new("loveholidays")}}}}
		review = &gh.PullRequestReview{State: new("approved"), User: &gh.User{Login: new("reviewer")}}
		stack = &gh.PullRequestStackDetails{ID: 123, Number: 7}
		message = &slack.Message{Msg: slack.Msg{Text: "existing stack", Timestamp: "123.456"}}
		notification = bot_slack.Notification{Key: "PR Stack 123:", Text: "updated stack"}
	})

	Context("NotifyPullRequestOpened", func() {
		It("should rebuild the stack using its repository, stack number and previous message", func() {
			githubMock.EXPECT().GetStack("loveholidays", "flux", 7).Return(stack, nil)
			slackMock.EXPECT().GetMessage(notification.Key).Return(message, nil)
			builder.EXPECT().BuildPRStackMessage(*stack, message.Text, emoji).Return(notification.Text)
			slackMock.EXPECT().SendOrUpdateMessage(notification)
			notifier.NotifyPullRequestOpened(slackMock, pr)
		})
		It("should create the first message when the stack is not posted yet", func() {
			githubMock.EXPECT().GetStack("loveholidays", "flux", 7).Return(stack, nil)
			slackMock.EXPECT().GetMessage(notification.Key).Return(nil, bot_slack.ErrMessageNotFound)
			builder.EXPECT().BuildPRStackMessage(*stack, "", emoji).Return(notification.Text)
			slackMock.EXPECT().SendOrUpdateMessage(notification)
			notifier.NotifyPullRequestOpened(slackMock, pr)
		})
		It("should stop when GitHub stack lookup fails", func() {
			lookupError := errors.New("GitHub unavailable")
			githubMock.EXPECT().GetStack("loveholidays", "flux", 7).Return(nil, lookupError)
			notifier.NotifyPullRequestOpened(slackMock, pr)
		})
		It("should stop when GitHub returns no usable stack", func() {
			githubMock.EXPECT().GetStack("loveholidays", "flux", 7).Return(nil, nil)
			notifier.NotifyPullRequestOpened(slackMock, pr)
		})
		It("should not overwrite the message when Slack lookup fails", func() {
			lookupError := errors.New("Slack unavailable")
			githubMock.EXPECT().GetStack("loveholidays", "flux", 7).Return(stack, nil)
			slackMock.EXPECT().GetMessage(notification.Key).Return(nil, lookupError)
			notifier.NotifyPullRequestOpened(slackMock, pr)
		})
	})

	Context("with an existing stack message", func() {
		BeforeEach(func() {
			slackMock.EXPECT().GetMessage(notification.Key).Return(message, nil)
		})
		It("should update the closed PR status", func() {
			pr.State = new("closed")
			builder.EXPECT().BuildPRStackStatusMessage(message, *pr, false, emoji).Return(notification.Text)
			slackMock.EXPECT().SendOrUpdateMessage(notification)
			notifier.NotifyPullRequestClosed(slackMock, pr)
		})
		It("should update the reopened PR status", func() {
			pr.State = new("open")
			builder.EXPECT().BuildPRStackStatusMessage(message, *pr, false, emoji).Return(notification.Text)
			slackMock.EXPECT().SendOrUpdateMessage(notification)
			notifier.NotifyPullRequestReopened(slackMock, pr)
		})
		It("should update approval status and post the formatted review", func() {
			review.Body = new("Looks good")
			builder.EXPECT().BuildPRStackStatusMessage(message, *pr, true, emoji).Return(notification.Text)
			slackMock.EXPECT().SendOrUpdateMessage(notification)
			users.EXPECT().GetUserDescriptor("reviewer").Return("<@reviewer>")
			builder.EXPECT().BuildPRStackReviewMessage("<@reviewer>", *pr, *review).Return("formatted review")
			slackMock.EXPECT().SendReply(message, "formatted review")
			notifier.NotifyReviewSubmitted(slackMock, pr, review)
		})
		It("should update approval status without replying when the review has no comment", func() {
			builder.EXPECT().BuildPRStackStatusMessage(message, *pr, true, emoji).Return(notification.Text)
			slackMock.EXPECT().SendOrUpdateMessage(notification)
			notifier.NotifyReviewSubmitted(slackMock, pr, review)
		})
		It("should post a non-approval review without changing approval status", func() {
			review.State = new("changes_requested")
			review.Body = new("Please update this")
			users.EXPECT().GetUserDescriptor("reviewer").Return("<@reviewer>")
			builder.EXPECT().BuildPRStackReviewMessage("<@reviewer>", *pr, *review).Return("formatted review")
			slackMock.EXPECT().SendReply(message, "formatted review")
			notifier.NotifyReviewSubmitted(slackMock, pr, review)
		})
		It("should clear approval status without replying when a review is dismissed", func() {
			builder.EXPECT().BuildPRStackStatusMessage(message, *pr, false, emoji).Return(notification.Text)
			slackMock.EXPECT().SendOrUpdateMessage(notification)
			notifier.NotifyReviewDismissed(slackMock, pr)
		})
		It("should identify the PR and preserve the supplied comment", func() {
			expected := "<https://github.com/loveholidays/flux/pull/42|PR #42>:\ncomment body"
			slackMock.EXPECT().SendReply(message, expected)
			notifier.NotifyComment(slackMock, pr, "comment body")
		})
	})

	It("should ignore a draft closure", func() {
		pr.Draft = new(true)
		notifier.NotifyPullRequestClosed(slackMock, pr)
	})

	Context("when the stack message cannot be found", func() {
		BeforeEach(func() {
			lookupError := errors.New("thread unavailable")
			slackMock.EXPECT().GetMessage(notification.Key).Return(nil, lookupError)
		})
		It("should stop on closure", func() {
			notifier.NotifyPullRequestClosed(slackMock, pr)
		})
		It("should stop on reopening", func() {
			notifier.NotifyPullRequestReopened(slackMock, pr)
		})
		It("should stop on a review submission", func() {
			notifier.NotifyReviewSubmitted(slackMock, pr, review)
		})
		It("should stop on a review dismissal", func() {
			notifier.NotifyReviewDismissed(slackMock, pr)
		})
		It("should stop on a comment", func() {
			notifier.NotifyComment(slackMock, pr, "comment body")
		})
	})
})
