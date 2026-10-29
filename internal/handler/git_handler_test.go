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
	bot_slack "git-slack-bot/internal/slack"
	mock_slack "git-slack-bot/internal/slack/mocks"
	mock_user "git-slack-bot/internal/user/mocks"

	gh "github.com/google/go-github/v56/github"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/slack-go/slack"
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
		userMock            *mock_user.MockService
		githubConfiguration config.GitHubConfiguration
	)

	BeforeEach(func() {
		mockCtrl = gomock.NewController(GinkgoT())
		slackMock = mock_slack.NewMockInteractor(mockCtrl)
		userMock = mock_user.NewMockService(mockCtrl)
		githubConfiguration = config.GitHubConfiguration{}
	})

	Context("HandlePullRequestEvents", func() {
		It("should no-op if coming from a ignored repo", func() {
			webHookHandler := handler.NewGitHandler(slackMock, userMock, validEmojis(), config.GitHubConfiguration{IgnoredRepos: []string{"hotels-and-ancillaries"}})

			userMock.EXPECT().IsTeamMember(gomock.Any()).Times(0)
			slackMock.EXPECT().SendMessageOnce(gomock.Any()).Times(0)
			webHookHandler.HandlePullRequestEvent(prOpenedJSONData)
		})

		It("should post slack message when pull request opened", func() {
			webHookHandler := handler.NewGitHandler(slackMock, userMock, validEmojis(), githubConfiguration)

			userMock.EXPECT().IsTeamMember(gomock.Any()).Return(true)
			userMock.EXPECT().GetUserDescriptor(gomock.Any()).Return("<@123>")

			expected := `<@123> [GS] Test slack id change:
https://github.com/loveholidays/hotels-and-ancillaries/pull/808`

			slackMock.EXPECT().SendMessageOnce(gomock.Cond[bot_slack.Notification](func(notification bot_slack.Notification) bool { return notification.Text == expected }))
			webHookHandler.HandlePullRequestEvent(prOpenedJSONData)
		})

		It("should post slack message when pull request ready for review", func() {
			webHookHandler := handler.NewGitHandler(slackMock, userMock, validEmojis(), githubConfiguration)

			userMock.EXPECT().IsTeamMember(gomock.Any()).Return(true)
			userMock.EXPECT().GetUserDescriptor(gomock.Any()).Return("<@123>")

			expected := `<@123> Moving duplicating configmaps to base:
https://github.com/loveholidays/flux/pull/92504`

			slackMock.EXPECT().SendMessageOnce(gomock.Cond[bot_slack.Notification](func(notification bot_slack.Notification) bool { return notification.Text == expected }))
			webHookHandler.HandlePullRequestEvent(prReadyForReviewJSONData)
		})

		It("should add merged emoji to message when pull request merged", func() {
			webHookHandler := handler.NewGitHandler(slackMock, userMock, validEmojis(), githubConfiguration)

			userMock.EXPECT().IsTeamMember(gomock.Any()).Return(true)
			messageKey := &slack.Message{}
			slackMock.EXPECT().GetMessage(gomock.Any()).Return(messageKey, nil)

			slackMock.EXPECT().AddReactionToMessage("merged", messageKey)
			webHookHandler.HandlePullRequestEvent(prMergedJSONData)
		})

		It("should add closed emoji when pull request closed", func() {
			webHookHandler := handler.NewGitHandler(slackMock, userMock, validEmojis(), githubConfiguration)

			userMock.EXPECT().IsTeamMember(gomock.Any()).Return(true)
			messageKey := &slack.Message{}
			slackMock.EXPECT().GetMessage(gomock.Any()).Return(messageKey, nil)

			slackMock.EXPECT().AddReactionToMessage("x", messageKey)
			webHookHandler.HandlePullRequestEvent(prClosedJSONData)
		})

		It("should remove closed emoji when pull request reopened", func() {
			webHookHandler := handler.NewGitHandler(slackMock, userMock, validEmojis(), githubConfiguration)

			userMock.EXPECT().IsTeamMember(gomock.Any()).Return(true)
			messageKey := &slack.Message{}
			slackMock.EXPECT().GetMessage(gomock.Any()).Return(messageKey, nil)

			slackMock.EXPECT().RemoveReactionFromMessage("x", messageKey)
			webHookHandler.HandlePullRequestEvent(prReopenedJSONData)
		})
	})

	Context("HandlePullRequestReviewEvent", func() {
		DescribeTable("bot reviews do not change the approved reaction", func(login, action string) {
			var event gh.PullRequestReviewEvent
			Expect(json.Unmarshal(prApprovedJSONData, &event)).To(Succeed())
			event.Action = gh.String(action)
			event.Review.User = &gh.User{Login: gh.String(login), Type: gh.String("Bot")}
			event.Review.Body = gh.String("Automated review")
			body, err := json.Marshal(event)
			Expect(err).NotTo(HaveOccurred())

			webHookHandler := handler.NewGitHandler(slackMock, userMock, validEmojis(), githubConfiguration)
			userMock.EXPECT().IsTeamMember(gomock.Any()).Return(true)
			userMock.EXPECT().IsIgnoredReviewUser(login).Return(false)
			if action == "submitted" {
				parent := &slack.Message{}
				slackMock.EXPECT().GetMessage(gomock.Any()).Return(parent, nil)
				userMock.EXPECT().GetUserDescriptor(login).Return(login)
				slackMock.EXPECT().SendReply(parent, login+" left a <"+event.Review.GetHTMLURL()+"|review>:\nAutomated review")
			}
			webHookHandler.HandlePullRequestReviewEvent(body)
		},
			Entry("CodeScene approval", "codescene-access[bot]", "submitted"),
			Entry("Codex approval", "chatgpt-codex-connector[bot]", "submitted"),
			Entry("Copilot approval", "copilot-pull-request-reviewer[bot]", "submitted"),
			Entry("dismissed bot approval", "codescene-access[bot]", "dismissed"),
		)

		It("should no-op if coming from a ignored repo", func() {
			webHookHandler := handler.NewGitHandler(slackMock, userMock, validEmojis(), config.GitHubConfiguration{IgnoredRepos: []string{"frontier"}})

			userMock.EXPECT().IsTeamMember(gomock.Any()).Times(0)
			slackMock.EXPECT().SendMessageOnce(gomock.Any()).Times(0)
			webHookHandler.HandlePullRequestReviewEvent(prApprovedJSONData)
		})

		It("should add tick emoji when pull request approved", func() {
			webHookHandler := handler.NewGitHandler(slackMock, userMock, validEmojis(), githubConfiguration)

			userMock.EXPECT().IsTeamMember(gomock.Any()).Return(true)
			userMock.EXPECT().IsIgnoredReviewUser(gomock.Any()).Return(false)
			messageKey := &slack.Message{}
			slackMock.EXPECT().GetMessage(gomock.Any()).Return(messageKey, nil)

			slackMock.EXPECT().AddReactionToMessage("+1", messageKey)
			webHookHandler.HandlePullRequestReviewEvent(prApprovedJSONData)
		})

		It("should post review body to slack as a reply when pull request review has body", func() {
			webHookHandler := handler.NewGitHandler(slackMock, userMock, validEmojis(), githubConfiguration)
			reviewBodyJSON := []byte(`{
				"action": "submitted",
				"review": {
					"state": "changes_requested",
					"body": "Please update this before merging.",
					"html_url": "https://github.test/example-org/example-repo/pull/42#pullrequestreview-1001",
					"user": {
						"login": "reviewer-login"
					}
				},
				"pull_request": {
					"html_url": "https://github.test/example-org/example-repo/pull/42",
					"user": {
						"login": "author-login"
					}
				},
				"repository": {
					"name": "example-repo"
				}
			}`)

			userMock.EXPECT().IsTeamMember("author-login").Return(true)
			userMock.EXPECT().IsIgnoredReviewUser("reviewer-login").Return(false)
			userMock.EXPECT().GetUserDescriptor("reviewer-login").Return("<@123>")
			messageKey := &slack.Message{}
			slackMock.EXPECT().GetMessage("<https://github.test/example-org/example-repo/pull/42>").Return(messageKey, nil)

			expected := `<@123> left a <https://github.test/example-org/example-repo/pull/42#pullrequestreview-1001|review>:
Please update this before merging.`

			slackMock.EXPECT().SendReply(messageKey, expected)
			slackMock.EXPECT().AddReactionToMessage(gomock.Any(), gomock.Any()).Times(0)
			webHookHandler.HandlePullRequestReviewEvent(reviewBodyJSON)
		})

		It("should post approval review body and add tick emoji when approved review has body", func() {
			webHookHandler := handler.NewGitHandler(slackMock, userMock, validEmojis(), githubConfiguration)
			approvedReviewBodyJSON := []byte(`{
				"action": "submitted",
				"review": {
					"state": "approved",
					"body": "Looks good overall.",
					"html_url": "https://github.test/example-org/example-repo/pull/42#pullrequestreview-1002",
					"user": {
						"login": "reviewer-login"
					}
				},
				"pull_request": {
					"html_url": "https://github.test/example-org/example-repo/pull/42",
					"user": {
						"login": "author-login"
					}
				},
				"repository": {
					"name": "example-repo"
				}
			}`)

			userMock.EXPECT().IsTeamMember("author-login").Return(true)
			userMock.EXPECT().IsIgnoredReviewUser("reviewer-login").Return(false)
			userMock.EXPECT().GetUserDescriptor("reviewer-login").Return("<@123>")
			messageKey := &slack.Message{}
			slackMock.EXPECT().GetMessage("<https://github.test/example-org/example-repo/pull/42>").Return(messageKey, nil)

			expected := `<@123> left a <https://github.test/example-org/example-repo/pull/42#pullrequestreview-1002|review>:
Looks good overall.`

			slackMock.EXPECT().AddReactionToMessage("+1", messageKey)
			slackMock.EXPECT().SendReply(messageKey, expected)
			webHookHandler.HandlePullRequestReviewEvent(approvedReviewBodyJSON)
		})

		It("should not add tick emoji when pull request reviewer is ignored", func() {
			webHookHandler := handler.NewGitHandler(slackMock, userMock, validEmojis(), githubConfiguration)

			userMock.EXPECT().IsTeamMember(gomock.Any()).Return(true)
			userMock.EXPECT().IsIgnoredReviewUser(gomock.Any()).Return(true)
			slackMock.EXPECT().GetMessage(gomock.Any()).Times(0)
			slackMock.EXPECT().AddReactionToMessage(gomock.Any(), gomock.Any()).Times(0)

			webHookHandler.HandlePullRequestReviewEvent(prApprovedJSONData)
		})

		It("should remove approve emoji when pull request review is dismissed", func() {
			webHookHandler := handler.NewGitHandler(slackMock, userMock, validEmojis(), githubConfiguration)

			userMock.EXPECT().IsTeamMember(gomock.Any()).Return(true)
			userMock.EXPECT().IsIgnoredReviewUser(gomock.Any()).Return(false)
			messageKey := &slack.Message{}
			slackMock.EXPECT().GetMessage(gomock.Any()).Return(messageKey, nil)

			slackMock.EXPECT().RemoveReactionFromMessage("+1", messageKey)
			webHookHandler.HandlePullRequestReviewEvent(prReviewDismissedJSONData)
		})

		It("should not remove approve emoji when dismissed review is from ignored reviewer", func() {
			webHookHandler := handler.NewGitHandler(slackMock, userMock, validEmojis(), githubConfiguration)

			userMock.EXPECT().IsTeamMember(gomock.Any()).Return(true)
			userMock.EXPECT().IsIgnoredReviewUser(gomock.Any()).Return(true)
			slackMock.EXPECT().GetMessage(gomock.Any()).Times(0)
			slackMock.EXPECT().RemoveReactionFromMessage(gomock.Any(), gomock.Any()).Times(0)

			webHookHandler.HandlePullRequestReviewEvent(prReviewDismissedJSONData)
		})
	})

	DescribeTable("handles comment actions without posting duplicate replies", func(body []byte, action string, isIssueComment bool) {
		var event map[string]json.RawMessage
		Expect(json.Unmarshal(body, &event)).To(Succeed())
		actionJSON, err := json.Marshal(action)
		Expect(err).NotTo(HaveOccurred())
		event["action"] = actionJSON
		body, err = json.Marshal(event)
		Expect(err).NotTo(HaveOccurred())

		webHookHandler := handler.NewGitHandler(slackMock, userMock, validEmojis(), githubConfiguration)
		if action == "edited" || action == "deleted" {
			userMock.EXPECT().IsTeamMember(gomock.Any()).Return(true)
			userMock.EXPECT().IsIgnoredCommentUser(gomock.Any()).Return(false)
			parent := &slack.Message{}
			slackMock.EXPECT().GetMessage(gomock.Any()).Return(parent, nil)
			commentURL := "https://github.com/loveholidays/yielding-ui/pull/61#discussion_r1425573584"
			if isIssueComment {
				commentURL = "https://github.com/loveholidays/hotels-and-ancillaries/pull/1015#issuecomment-1924011855"
			}
			if action == "edited" {
				userMock.EXPECT().GetUserDescriptor(gomock.Any()).Return("<@123>")
				slackMock.EXPECT().UpdateReply(parent, commentURL, gomock.Any())
			} else {
				slackMock.EXPECT().DeleteReply(parent, commentURL)
			}
		}
		if isIssueComment {
			webHookHandler.HandleIssueCommentEvent(body)
		} else {
			webHookHandler.HandlePullRequestReviewCommentEvent(body)
		}
	},
		Entry("edited inline comment", prCommentJSONData, "edited", false),
		Entry("deleted inline comment", prCommentJSONData, "deleted", false),
		Entry("inline comment without action", prCommentJSONData, "", false),
		Entry("unknown inline comment action", prCommentJSONData, "unknown", false),
		Entry("edited top-level comment", prIssueCommentJSONData, "edited", true),
		Entry("deleted top-level comment", prIssueCommentJSONData, "deleted", true),
		Entry("top-level comment without action", prIssueCommentJSONData, "", true),
		Entry("unknown top-level comment action", prIssueCommentJSONData, "unknown", true),
	)

	Context("HandlePullRequestReviewCommentEvent", func() {
		It("should no-op if coming from a ignored repo", func() {
			webHookHandler := handler.NewGitHandler(slackMock, userMock, validEmojis(), config.GitHubConfiguration{IgnoredRepos: []string{"yielding-ui"}})

			userMock.EXPECT().IsTeamMember(gomock.Any()).Times(0)
			slackMock.EXPECT().SendMessageOnce(gomock.Any()).Times(0)
			webHookHandler.HandlePullRequestReviewCommentEvent(prCommentJSONData)
		})

		It("should post comment to slack as a reply when pull request commented on", func() {
			webHookHandler := handler.NewGitHandler(slackMock, userMock, validEmojis(), githubConfiguration)

			userMock.EXPECT().IsTeamMember(gomock.Any()).Return(true)
			userMock.EXPECT().GetUserDescriptor(gomock.Any()).Return("<@123>")
			userMock.EXPECT().IsIgnoredCommentUser("szmglh").Return(false)
			messageKey := &slack.Message{}
			slackMock.EXPECT().GetMessage(gomock.Any()).Return(messageKey, nil)

			expected := `<@123> left a <https://github.com/loveholidays/yielding-ui/pull/61#discussion_r1425573584|comment>:
> @L1 Dockerfile
Sorry, that's not allowed since https://adrs.lvh.systems/adr/20231205-container-image-tagging/`

			slackMock.EXPECT().SendReply(messageKey, expected)
			webHookHandler.HandlePullRequestReviewCommentEvent(prCommentJSONData)
		})

		It("should ignore pull request commented on from ignored comment user", func() {
			webHookHandler := handler.NewGitHandler(slackMock, userMock, validEmojis(), githubConfiguration)

			userMock.EXPECT().IsTeamMember(gomock.Any()).Return(true)
			userMock.EXPECT().IsIgnoredCommentUser(gomock.Any()).Return(true)

			slackMock.EXPECT().GetMessage(gomock.Any()).MaxTimes(0)
			slackMock.EXPECT().SendReply(gomock.Any(), gomock.Any()).MaxTimes(0)

			webHookHandler.HandlePullRequestReviewCommentEvent(prCommentJSONData)
		})
	})

	Context("HandleIssueCommentEvent", func() {
		It("should no-op if coming from a ignored repo", func() {
			webHookHandler := handler.NewGitHandler(slackMock, userMock, validEmojis(), config.GitHubConfiguration{IgnoredRepos: []string{"hotels-and-ancillaries"}})

			userMock.EXPECT().IsTeamMember(gomock.Any()).Times(0)
			slackMock.EXPECT().SendMessageOnce(gomock.Any()).Times(0)
			webHookHandler.HandleIssueCommentEvent(prIssueCommentJSONData)
		})

		It("should post comment to slack as a reply when top level pull request comment is added to PR", func() {
			webHookHandler := handler.NewGitHandler(slackMock, userMock, validEmojis(), githubConfiguration)

			userMock.EXPECT().IsTeamMember(gomock.Any()).Return(true)
			userMock.EXPECT().GetUserDescriptor(gomock.Any()).Return("<@123>")
			userMock.EXPECT().IsIgnoredCommentUser("georgesmith96").Return(false)
			messageKey := &slack.Message{}
			slackMock.EXPECT().GetMessage(gomock.Any()).Return(messageKey, nil)

			expected := `<@123> left a <https://github.com/loveholidays/hotels-and-ancillaries/pull/1015#issuecomment-1924011855|comment>:
Just leaving a top level comment here`

			slackMock.EXPECT().SendReply(messageKey, expected)
			webHookHandler.HandleIssueCommentEvent(prIssueCommentJSONData)
		})

		It("should ignore top level pull request comment added to PR from ignored comment user", func() {
			webHookHandler := handler.NewGitHandler(slackMock, userMock, validEmojis(), githubConfiguration)

			userMock.EXPECT().IsTeamMember(gomock.Any()).Return(true)
			userMock.EXPECT().IsIgnoredCommentUser("georgesmith96").Return(true)
			slackMock.EXPECT().GetMessage(gomock.Any()).MaxTimes(0)

			slackMock.EXPECT().SendReply(gomock.Any(), gomock.Any()).MaxTimes(0)
			webHookHandler.HandleIssueCommentEvent(prIssueCommentJSONData)
		})
	})
})

func validEmojis() config.EmojiConfiguration {
	return config.EmojiConfiguration{
		Approve: "+1",
		Merge:   "merged",
		Close:   "x",
	}
}
