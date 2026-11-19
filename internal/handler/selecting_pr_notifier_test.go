package handler_test

import (
	"errors"
	mock_github "git-slack-bot/internal/github/mocks"
	"git-slack-bot/internal/handler"
	mock_handler "git-slack-bot/internal/handler/mocks"
	mock_slack "git-slack-bot/internal/slack/mocks"

	gh "github.com/google/go-github/v93/github"
	. "github.com/onsi/ginkgo/v2"
	"go.uber.org/mock/gomock"
)

var _ = Describe("Selecting PR notifier", func() {
	var (
		single     *mock_handler.MockPullRequestNotifier
		stacked    *mock_handler.MockPullRequestNotifier
		githubMock *mock_github.MockInteractor
		slackMock  *mock_slack.MockInteractor
		notifier   handler.PullRequestNotifier
		pr         *gh.PullRequest
		review     *gh.PullRequestReview
	)
	BeforeEach(func() {
		controller := gomock.NewController(GinkgoT())
		single = mock_handler.NewMockPullRequestNotifier(controller)
		stacked = mock_handler.NewMockPullRequestNotifier(controller)
		githubMock = mock_github.NewMockInteractor(controller)
		slackMock = mock_slack.NewMockInteractor(controller)
		notifier = handler.NewSelectingPRNotifier(single, stacked, githubMock)
		pr = &gh.PullRequest{Number: new(42), HTMLURL: new("https://github.com/loveholidays/flux/pull/42"), Base: &gh.PullRequestBranch{}}
		review = &gh.PullRequestReview{State: new("approved"), Body: new("Looks good")}
	})

	Context("without stack metadata", func() {
		It("should delegate opening to the single notifier", func() {
			single.EXPECT().NotifyPullRequestOpened(slackMock, pr)
			notifier.NotifyPullRequestOpened(slackMock, pr)
		})
		It("should delegate closure to the single notifier", func() {
			single.EXPECT().NotifyPullRequestClosed(slackMock, pr)
			notifier.NotifyPullRequestClosed(slackMock, pr)
		})
		It("should delegate reopening to the single notifier", func() {
			single.EXPECT().NotifyPullRequestReopened(slackMock, pr)
			notifier.NotifyPullRequestReopened(slackMock, pr)
		})
		It("should delegate a submitted review to the single notifier", func() {
			single.EXPECT().NotifyReviewSubmitted(slackMock, pr, review)
			notifier.NotifyReviewSubmitted(slackMock, pr, review)
		})
		It("should delegate a dismissed review to the single notifier", func() {
			single.EXPECT().NotifyReviewDismissed(slackMock, pr)
			notifier.NotifyReviewDismissed(slackMock, pr)
		})
		It("should delegate a comment to the single notifier without fetching a complete PR again", func() {
			single.EXPECT().NotifyComment(slackMock, pr, "comment")
			notifier.NotifyComment(slackMock, pr, "comment")
		})
	})

	Context("with stack metadata", func() {
		BeforeEach(func() {
			pr.Stack = &gh.PullRequestStack{ID: new(int64(123)), Number: new(7)}
			pr.Base = nil
		})
		It("should delegate opening to the stacked notifier", func() {
			stacked.EXPECT().NotifyPullRequestOpened(slackMock, pr)
			notifier.NotifyPullRequestOpened(slackMock, pr)
		})
		It("should delegate closure to the stacked notifier", func() {
			stacked.EXPECT().NotifyPullRequestClosed(slackMock, pr)
			notifier.NotifyPullRequestClosed(slackMock, pr)
		})
		It("should delegate reopening to the stacked notifier", func() {
			stacked.EXPECT().NotifyPullRequestReopened(slackMock, pr)
			notifier.NotifyPullRequestReopened(slackMock, pr)
		})
		It("should delegate a submitted review to the stacked notifier", func() {
			stacked.EXPECT().NotifyReviewSubmitted(slackMock, pr, review)
			notifier.NotifyReviewSubmitted(slackMock, pr, review)
		})
		It("should delegate a dismissed review to the stacked notifier", func() {
			stacked.EXPECT().NotifyReviewDismissed(slackMock, pr)
			notifier.NotifyReviewDismissed(slackMock, pr)
		})
		It("should delegate a comment without another lookup when stack metadata is already present", func() {
			stacked.EXPECT().NotifyComment(slackMock, pr, "comment")
			notifier.NotifyComment(slackMock, pr, "comment")
		})
	})

	Context("with a partial PR from an issue comment", func() {
		var partial *gh.PullRequest
		BeforeEach(func() {
			partial = &gh.PullRequest{HTMLURL: pr.HTMLURL}
		})
		It("should fetch the complete PR and delegate to the single notifier", func() {
			githubMock.EXPECT().GetPullRequest(pr.GetHTMLURL()).Return(pr, nil)
			single.EXPECT().NotifyComment(slackMock, pr, "comment")
			notifier.NotifyComment(slackMock, partial, "comment")
		})
		It("should fetch the complete PR and delegate to the stacked notifier", func() {
			pr.Stack = &gh.PullRequestStack{ID: new(int64(123)), Number: new(7)}
			githubMock.EXPECT().GetPullRequest(pr.GetHTMLURL()).Return(pr, nil)
			stacked.EXPECT().NotifyComment(slackMock, pr, "comment")
			notifier.NotifyComment(slackMock, partial, "comment")
		})
		It("should not delegate when the lookup fails", func() {
			lookupError := errors.New("GitHub unavailable")
			githubMock.EXPECT().GetPullRequest(pr.GetHTMLURL()).Return(nil, lookupError)
			notifier.NotifyComment(slackMock, partial, "comment")
		})
		It("should not delegate when the lookup returns no PR", func() {
			githubMock.EXPECT().GetPullRequest(pr.GetHTMLURL()).Return(nil, nil)
			notifier.NotifyComment(slackMock, partial, "comment")
		})
	})
})
