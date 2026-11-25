/*
git-slack-bot
Copyright (C) 2025 loveholidays

This program is free software; you can redistribute it and/or
modify it under the terms of the GNU Lesser General Public
License as published by the Free Software Foundation; either
version 3 of the License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the GNU
Lesser General Public License for more details.

You should have received a copy of the GNU Lesser General Public License
along with this program; if not, write to the Free Software Foundation,
Inc., 51 Franklin Street, Fifth Floor, Boston, MA  02110-1301, USA.
*/

package handler_test

import (
	_ "embed"
	"encoding/json"
	"git-slack-bot/internal/config"
	"git-slack-bot/internal/handler"
	mock_handler "git-slack-bot/internal/handler/mocks"
	mock_slack "git-slack-bot/internal/slack/mocks"
	mock_user "git-slack-bot/internal/user/mocks"

	gh "github.com/google/go-github/v56/github"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
)

var (
	//go:embed example-requests/pr-opened.json
	prOpenedJSONData []byte
	//go:embed example-requests/pr-ready-for-review.json
	prReadyForReviewJSONData []byte
	//go:embed example-requests/pr-merged.json
	prMergedJSONData []byte
	//go:embed example-requests/pr-closed.json
	prClosedJSONData []byte
	//go:embed example-requests/pr-reopened.json
	prReopenedJSONData []byte
	//go:embed example-requests/pr-approved.json
	prApprovedJSONData []byte
	//go:embed example-requests/pr-review-dismissed.json
	prReviewDismissedJSONData []byte
	//go:embed example-requests/pr-comment.json
	prCommentJSONData []byte
	//go:embed example-requests/pr-top-level-comment.json
	prIssueCommentJSONData []byte
)

var _ = Describe("HandleGitEvents", func() {
	var (
		mockCtrl            *gomock.Controller
		slackMock           *mock_slack.MockInteractor
		notifierMock        *mock_handler.MockPullRequestNotifier
		userMock            *mock_user.MockService
		githubConfiguration config.GitHubConfiguration
	)

	BeforeEach(func() {
		mockCtrl = gomock.NewController(GinkgoT())
		slackMock = mock_slack.NewMockInteractor(mockCtrl)
		notifierMock = mock_handler.NewMockPullRequestNotifier(mockCtrl)
		userMock = mock_user.NewMockService(mockCtrl)
		githubConfiguration = config.GitHubConfiguration{}
	})

	Context("HandlePullRequestEvents", func() {
		It("should no-op if coming from a ignored repo", func() {
			webHookHandler := handler.NewGitHandler(slackMock, userMock, notifierMock, config.Configuration{GitHub: config.GitHubConfiguration{IgnoredRepos: []string{"hotels-and-ancillaries"}}})

			userMock.EXPECT().IsTeamMember(gomock.Any()).Times(0)
			webHookHandler.HandlePullRequestEvent(prOpenedJSONData)
		})

		It("should notify when pull request opened", func() {
			webHookHandler := handler.NewGitHandler(slackMock, userMock, notifierMock, config.Configuration{GitHub: githubConfiguration})
			expectedEvent := gh.PullRequestEvent{}
			err := json.Unmarshal(prOpenedJSONData, &expectedEvent)
			Expect(err).NotTo(HaveOccurred())

			userMock.EXPECT().IsTeamMember(gomock.Any()).Return(true)
			notifierMock.EXPECT().NotifyPullRequestOpened(slackMock, expectedEvent.PullRequest)

			webHookHandler.HandlePullRequestEvent(prOpenedJSONData)
		})

		It("should notify when pull request ready for review", func() {
			webHookHandler := handler.NewGitHandler(slackMock, userMock, notifierMock, config.Configuration{GitHub: githubConfiguration})
			expectedEvent := gh.PullRequestEvent{}
			err := json.Unmarshal(prReadyForReviewJSONData, &expectedEvent)
			Expect(err).NotTo(HaveOccurred())

			userMock.EXPECT().IsTeamMember(gomock.Any()).Return(true)
			notifierMock.EXPECT().NotifyPullRequestOpened(slackMock, expectedEvent.PullRequest)

			webHookHandler.HandlePullRequestEvent(prReadyForReviewJSONData)
		})

		It("should notify when pull request merged", func() {
			webHookHandler := handler.NewGitHandler(slackMock, userMock, notifierMock, config.Configuration{GitHub: githubConfiguration})
			expectedEvent := gh.PullRequestEvent{}
			err := json.Unmarshal(prMergedJSONData, &expectedEvent)
			Expect(err).NotTo(HaveOccurred())

			userMock.EXPECT().IsTeamMember(gomock.Any()).Return(true)
			notifierMock.EXPECT().NotifyPullRequestClosed(slackMock, expectedEvent.PullRequest)

			webHookHandler.HandlePullRequestEvent(prMergedJSONData)
		})

		It("should notify when pull request closed", func() {
			webHookHandler := handler.NewGitHandler(slackMock, userMock, notifierMock, config.Configuration{GitHub: githubConfiguration})
			expectedEvent := gh.PullRequestEvent{}
			err := json.Unmarshal(prClosedJSONData, &expectedEvent)
			Expect(err).NotTo(HaveOccurred())

			userMock.EXPECT().IsTeamMember(gomock.Any()).Return(true)
			notifierMock.EXPECT().NotifyPullRequestClosed(slackMock, expectedEvent.PullRequest)

			webHookHandler.HandlePullRequestEvent(prClosedJSONData)
		})

		It("should notify when pull request reopened", func() {
			webHookHandler := handler.NewGitHandler(slackMock, userMock, notifierMock, config.Configuration{GitHub: githubConfiguration})
			expectedEvent := gh.PullRequestEvent{}
			err := json.Unmarshal(prReopenedJSONData, &expectedEvent)
			Expect(err).NotTo(HaveOccurred())

			userMock.EXPECT().IsTeamMember(gomock.Any()).Return(true)
			notifierMock.EXPECT().NotifyPullRequestReopened(slackMock, expectedEvent.PullRequest)

			webHookHandler.HandlePullRequestEvent(prReopenedJSONData)
		})
	})

	Context("HandlePullRequestReviewEvent", func() {
		It("should no-op if coming from a ignored repo", func() {
			webHookHandler := handler.NewGitHandler(slackMock, userMock, notifierMock, config.Configuration{GitHub: config.GitHubConfiguration{IgnoredRepos: []string{"frontier"}}})

			userMock.EXPECT().IsTeamMember(gomock.Any()).Times(0)
			webHookHandler.HandlePullRequestReviewEvent(prApprovedJSONData)
		})

		It("should notify when a review is submitted", func() {
			webHookHandler := handler.NewGitHandler(slackMock, userMock, notifierMock, config.Configuration{GitHub: githubConfiguration})
			expectedEvent := gh.PullRequestReviewEvent{}
			err := json.Unmarshal(prApprovedJSONData, &expectedEvent)
			Expect(err).NotTo(HaveOccurred())

			userMock.EXPECT().IsTeamMember(gomock.Any()).Return(true)
			userMock.EXPECT().IsIgnoredReviewUser(gomock.Any()).Return(false)
			notifierMock.EXPECT().NotifyReviewSubmitted(slackMock, expectedEvent.PullRequest, expectedEvent.Review)

			webHookHandler.HandlePullRequestReviewEvent(prApprovedJSONData)
		})

		It("should not notify when pull request reviewer is ignored", func() {
			webHookHandler := handler.NewGitHandler(slackMock, userMock, notifierMock, config.Configuration{GitHub: githubConfiguration})

			userMock.EXPECT().IsIgnoredReviewUser(gomock.Any()).Return(true)

			webHookHandler.HandlePullRequestReviewEvent(prApprovedJSONData)
		})

		It("should notify when a review is dismissed", func() {
			webHookHandler := handler.NewGitHandler(slackMock, userMock, notifierMock, config.Configuration{GitHub: githubConfiguration})
			expectedEvent := gh.PullRequestReviewEvent{}
			err := json.Unmarshal(prReviewDismissedJSONData, &expectedEvent)
			Expect(err).NotTo(HaveOccurred())

			userMock.EXPECT().IsTeamMember(gomock.Any()).Return(true)
			userMock.EXPECT().IsIgnoredReviewUser(gomock.Any()).Return(false)
			notifierMock.EXPECT().NotifyReviewDismissed(slackMock, expectedEvent.PullRequest)

			webHookHandler.HandlePullRequestReviewEvent(prReviewDismissedJSONData)
		})

		It("should not notify when dismissed review is from ignored reviewer", func() {
			webHookHandler := handler.NewGitHandler(slackMock, userMock, notifierMock, config.Configuration{GitHub: githubConfiguration})

			userMock.EXPECT().IsIgnoredReviewUser(gomock.Any()).Return(true)

			webHookHandler.HandlePullRequestReviewEvent(prReviewDismissedJSONData)
		})
	})

	Context("HandlePullRequestReviewCommentEvent", func() {
		It("should no-op if coming from a ignored repo", func() {
			webHookHandler := handler.NewGitHandler(slackMock, userMock, notifierMock, config.Configuration{GitHub: config.GitHubConfiguration{IgnoredRepos: []string{"yielding-ui"}}})

			userMock.EXPECT().IsTeamMember(gomock.Any()).Times(0)
			webHookHandler.HandlePullRequestReviewCommentEvent(prCommentJSONData)
		})

		It("should post comment to slack as a reply when pull request commented on", func() {
			webHookHandler := handler.NewGitHandler(slackMock, userMock, notifierMock, config.Configuration{GitHub: githubConfiguration})

			userMock.EXPECT().IsTeamMember(gomock.Any()).Return(true)
			userMock.EXPECT().GetUserDescriptor(gomock.Any()).Return("<@123>")
			userMock.EXPECT().IsIgnoredCommentUser("szmglh").Return(false)

			expected := `<@123> left a <https://github.com/loveholidays/yielding-ui/pull/61#discussion_r1425573584|comment>:
> @L1 Dockerfile
Sorry, that's not allowed since https://adrs.lvh.systems/adr/20231205-container-image-tagging/`

			expectedEvent := gh.PullRequestReviewCommentEvent{}
			err := json.Unmarshal(prCommentJSONData, &expectedEvent)
			Expect(err).NotTo(HaveOccurred())
			expectedPullRequest := expectedEvent.PullRequest
			notifierMock.EXPECT().NotifyComment(slackMock, expectedPullRequest, expected)
			webHookHandler.HandlePullRequestReviewCommentEvent(prCommentJSONData)
		})

		It("should ignore pull request commented on from ignored comment user", func() {
			webHookHandler := handler.NewGitHandler(slackMock, userMock, notifierMock, config.Configuration{GitHub: githubConfiguration})

			userMock.EXPECT().IsIgnoredCommentUser(gomock.Any()).Return(true)

			webHookHandler.HandlePullRequestReviewCommentEvent(prCommentJSONData)
		})
	})

	Context("HandleIssueCommentEvent", func() {
		It("should no-op if coming from a ignored repo", func() {
			webHookHandler := handler.NewGitHandler(slackMock, userMock, notifierMock, config.Configuration{GitHub: config.GitHubConfiguration{IgnoredRepos: []string{"hotels-and-ancillaries"}}})

			userMock.EXPECT().IsTeamMember(gomock.Any()).Times(0)
			webHookHandler.HandleIssueCommentEvent(prIssueCommentJSONData)
		})

		It("should post comment to slack as a reply when top level pull request comment is added to PR", func() {
			webHookHandler := handler.NewGitHandler(slackMock, userMock, notifierMock, config.Configuration{GitHub: githubConfiguration})

			userMock.EXPECT().IsTeamMember(gomock.Any()).Return(true)
			userMock.EXPECT().GetUserDescriptor(gomock.Any()).Return("<@123>")
			userMock.EXPECT().IsIgnoredCommentUser("georgesmith96").Return(false)

			expected := `<@123> left a <https://github.com/loveholidays/hotels-and-ancillaries/pull/1015#issuecomment-1924011855|comment>:
Just leaving a top level comment here`

			expectedEvent := gh.IssueCommentEvent{}
			err := json.Unmarshal(prIssueCommentJSONData, &expectedEvent)
			Expect(err).NotTo(HaveOccurred())
			expectedPullRequest := &gh.PullRequest{User: expectedEvent.Issue.User, Assignees: expectedEvent.Issue.Assignees, HTMLURL: expectedEvent.Issue.HTMLURL}
			notifierMock.EXPECT().NotifyComment(slackMock, expectedPullRequest, expected)
			webHookHandler.HandleIssueCommentEvent(prIssueCommentJSONData)
		})

		It("should ignore top level pull request comment added to PR from ignored comment user", func() {
			webHookHandler := handler.NewGitHandler(slackMock, userMock, notifierMock, config.Configuration{GitHub: githubConfiguration})

			userMock.EXPECT().IsIgnoredCommentUser("georgesmith96").Return(true)

			webHookHandler.HandleIssueCommentEvent(prIssueCommentJSONData)
		})
	})
})
