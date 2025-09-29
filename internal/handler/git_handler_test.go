/*
git-slack-bot
Copyright (C) 2024 loveholidays

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
	"git-slack-bot/internal/config"
	"git-slack-bot/internal/handler"
	mock_slack "git-slack-bot/internal/slack/mocks"
	mock_user "git-slack-bot/internal/user/mocks"

	. "github.com/onsi/ginkgo/v2"
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
	//go:embed example-requests/pr-comment.json
	prCommentJSONData []byte
	//go:embed example-requests/pr-top-level-comment.json
	prIssueCommentJSONData []byte
)

var _ = Describe("HandleGitEvents", func() {
	var (
		mockCtrl          *gomock.Controller
		slackMock         *mock_slack.MockInteractor
		userMock          *mock_user.MockService
		ignoredReposEmpty []string
	)

	BeforeEach(func() {
		mockCtrl = gomock.NewController(GinkgoT())
		slackMock = mock_slack.NewMockInteractor(mockCtrl)
		userMock = mock_user.NewMockService(mockCtrl)
		ignoredReposEmpty = []string{}
	})

	Context("HandlePullRequestEvents", func() {
		It("should no-op if coming from a ignored repo", func() {
			webHookHandler := handler.NewGitHandler(slackMock, userMock, validEmojis(), []string{"hotels-and-ancillaries"})

			userMock.EXPECT().IsTeamMember(gomock.Any()).Times(0)
			slackMock.EXPECT().SendMessage(gomock.Any()).Times(0)
			webHookHandler.HandlePullRequestEvent(prOpenedJSONData)
		})

		It("should post slack message when pull request opened", func() {
			webHookHandler := handler.NewGitHandler(slackMock, userMock, validEmojis(), ignoredReposEmpty)

			userMock.EXPECT().IsTeamMember(gomock.Any()).Return(true)
			userMock.EXPECT().GetUserDescriptor(gomock.Any()).Return("<@123>")

			expected := `<@123> [GS] Test slack id change:
https://github.com/loveholidays/hotels-and-ancillaries/pull/808`

			slackMock.EXPECT().SendMessage(expected)
			webHookHandler.HandlePullRequestEvent(prOpenedJSONData)
		})

		It("should post slack message when pull request ready for review", func() {
			webHookHandler := handler.NewGitHandler(slackMock, userMock, validEmojis(), ignoredReposEmpty)

			userMock.EXPECT().IsTeamMember(gomock.Any()).Return(true)
			userMock.EXPECT().GetUserDescriptor(gomock.Any()).Return("<@123>")

			expected := `<@123> Moving duplicating configmaps to base:
https://github.com/loveholidays/flux/pull/92504`

			slackMock.EXPECT().SendMessage(expected)
			webHookHandler.HandlePullRequestEvent(prReadyForReviewJSONData)
		})

		It("should add merged emoji to message when pull request merged", func() {
			webHookHandler := handler.NewGitHandler(slackMock, userMock, validEmojis(), ignoredReposEmpty)

			userMock.EXPECT().IsTeamMember(gomock.Any()).Return(true)
			messageKey := &slack.Message{}
			slackMock.EXPECT().GetMessage(gomock.Any()).Return(messageKey, nil)

			slackMock.EXPECT().AddReactionToMessage("merged", messageKey)
			webHookHandler.HandlePullRequestEvent(prMergedJSONData)
		})

		It("should add closed emoji when pull request closed", func() {
			webHookHandler := handler.NewGitHandler(slackMock, userMock, validEmojis(), ignoredReposEmpty)

			userMock.EXPECT().IsTeamMember(gomock.Any()).Return(true)
			messageKey := &slack.Message{}
			slackMock.EXPECT().GetMessage(gomock.Any()).Return(messageKey, nil)

			slackMock.EXPECT().AddReactionToMessage("x", messageKey)
			webHookHandler.HandlePullRequestEvent(prClosedJSONData)
		})

		It("should remove closed emoji when pull request reopened", func() {
			webHookHandler := handler.NewGitHandler(slackMock, userMock, validEmojis(), ignoredReposEmpty)

			userMock.EXPECT().IsTeamMember(gomock.Any()).Return(true)
			messageKey := &slack.Message{}
			slackMock.EXPECT().GetMessage(gomock.Any()).Return(messageKey, nil)

			slackMock.EXPECT().RemoveReactionFromMessage("x", messageKey)
			webHookHandler.HandlePullRequestEvent(prReopenedJSONData)
		})
	})

	Context("HandlePullRequestReviewEvent", func() {
		It("should no-op if coming from a ignored repo", func() {
			webHookHandler := handler.NewGitHandler(slackMock, userMock, validEmojis(), []string{"frontier"})

			userMock.EXPECT().IsTeamMember(gomock.Any()).Times(0)
			slackMock.EXPECT().SendMessage(gomock.Any()).Times(0)
			webHookHandler.HandlePullRequestReviewEvent(prApprovedJSONData)
		})

		It("should add tick emoji when pull request approved", func() {
			webHookHandler := handler.NewGitHandler(slackMock, userMock, validEmojis(), ignoredReposEmpty)

			userMock.EXPECT().IsTeamMember(gomock.Any()).Return(true)
			userMock.EXPECT().IsIgnoredReviewUser(gomock.Any()).Return(false)
			messageKey := &slack.Message{}
			slackMock.EXPECT().GetMessage(gomock.Any()).Return(messageKey, nil)

			slackMock.EXPECT().AddReactionToMessage("+1", messageKey)
			webHookHandler.HandlePullRequestReviewEvent(prApprovedJSONData)
		})

		It("should not add tick emoji when pull request reviewer is ignored", func() {
			webHookHandler := handler.NewGitHandler(slackMock, userMock, validEmojis(), ignoredReposEmpty)

			userMock.EXPECT().IsTeamMember(gomock.Any()).Return(true)
			userMock.EXPECT().IsIgnoredReviewUser(gomock.Any()).Return(true)
			slackMock.EXPECT().GetMessage(gomock.Any()).Times(0)
			slackMock.EXPECT().AddReactionToMessage(gomock.Any(), gomock.Any()).Times(0)

			webHookHandler.HandlePullRequestReviewEvent(prApprovedJSONData)
		})
	})

	Context("HandlePullRequestReviewCommentEvent", func() {
		It("should no-op if coming from a ignored repo", func() {
			webHookHandler := handler.NewGitHandler(slackMock, userMock, validEmojis(), []string{"yielding-ui"})

			userMock.EXPECT().IsTeamMember(gomock.Any()).Times(0)
			slackMock.EXPECT().SendMessage(gomock.Any()).Times(0)
			webHookHandler.HandlePullRequestReviewCommentEvent(prCommentJSONData)
		})

		It("should post comment to slack as a reply when pull request commented on", func() {
			webHookHandler := handler.NewGitHandler(slackMock, userMock, validEmojis(), ignoredReposEmpty)

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
			webHookHandler := handler.NewGitHandler(slackMock, userMock, validEmojis(), ignoredReposEmpty)

			userMock.EXPECT().IsTeamMember(gomock.Any()).Return(true)
			userMock.EXPECT().IsIgnoredCommentUser(gomock.Any()).Return(true)

			slackMock.EXPECT().GetMessage(gomock.Any()).MaxTimes(0)
			slackMock.EXPECT().SendReply(gomock.Any(), gomock.Any()).MaxTimes(0)

			webHookHandler.HandlePullRequestReviewCommentEvent(prCommentJSONData)
		})
	})

	Context("HandleIssueCommentEvent", func() {
		It("should no-op if coming from a ignored repo", func() {
			webHookHandler := handler.NewGitHandler(slackMock, userMock, validEmojis(), []string{"hotels-and-ancillaries"})

			userMock.EXPECT().IsTeamMember(gomock.Any()).Times(0)
			slackMock.EXPECT().SendMessage(gomock.Any()).Times(0)
			webHookHandler.HandleIssueCommentEvent(prIssueCommentJSONData)
		})

		It("should post comment to slack as a reply when top level pull request comment is added to PR", func() {
			webHookHandler := handler.NewGitHandler(slackMock, userMock, validEmojis(), ignoredReposEmpty)

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
			webHookHandler := handler.NewGitHandler(slackMock, userMock, validEmojis(), ignoredReposEmpty)

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
