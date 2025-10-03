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

package user_test

import (
	"errors"
	"git-slack-bot/internal/config"
	mock_slack "git-slack-bot/internal/slack/mocks"
	"git-slack-bot/internal/user"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
)

func TestUser(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "User service tests")
}

var _ = Describe("HandleUser", func() {
	var (
		mockCtrl  *gomock.Controller
		slackMock *mock_slack.MockInteractor
	)

	BeforeEach(func() {
		mockCtrl = gomock.NewController(GinkgoT())
		slackMock = mock_slack.NewMockInteractor(mockCtrl)
	})

	Context("IsTeamMember", func() {
		It("should return true if user is git team member", func() {
			service := user.NewService(nil, []string{"userLogin"}, nil, nil, nil)

			Expect(service.IsTeamMember("userLogin")).To(BeTrue())
		})

		It("should return false if user is not git team member", func() {
			service := user.NewService(nil, []string{"userLogin"}, nil, nil, nil)

			Expect(service.IsTeamMember("differentUserLogin")).To(BeFalse())
		})
	})

	Context("IsIgnoredCommentUser", func() {
		It("should return true if user comment should be ignored", func() {
			service := user.NewService(nil, []string{"userLogin"}, nil, []string{"userLogin"}, nil)

			Expect(service.IsIgnoredCommentUser("userLogin")).To(BeTrue())
		})

		It("should return false if user comment should not be ignored", func() {
			service := user.NewService(nil, []string{"userLogin"}, nil, []string{"userLogin"}, nil)

			Expect(service.IsIgnoredCommentUser("differentUserLogin")).To(BeFalse())
		})
	})

	Context("IsIgnoredReviewUser", func() {
		It("should return true if user comment should be ignored", func() {
			service := user.NewService(nil, []string{"userLogin"}, nil, nil, []string{"userLogin"})

			Expect(service.IsIgnoredReviewUser("userLogin")).To(BeTrue())
		})

		It("should return false if user comment should not be ignored", func() {
			service := user.NewService(nil, []string{"userLogin"}, nil, nil, []string{"userLogin"})

			Expect(service.IsIgnoredReviewUser("differentUserLogin")).To(BeFalse())
		})
	})

	Context("GetUserDescriptor", func() {
		It("should return slack ID if slack ID found", func() {
			emails := []config.GithubEmailToSlackEmail{
				{
					GithubEmail: "userLogin",
					SlackEmail:  "user@user.com",
				},
			}
			service := user.NewService(slackMock, []string{"userLogin"}, emails, nil, nil)

			slackMock.EXPECT().GetUserIDByEmail("user@user.com").Return("123", nil)

			actual := service.GetUserDescriptor("userLogin")

			Expect(actual).To(Equal("<@123>"))
		})

		It("should return github login if there is no mapping between github and slack emails", func() {

			service := user.NewService(slackMock, []string{"userLogin"}, []config.GithubEmailToSlackEmail{}, nil, nil)

			actual := service.GetUserDescriptor("userLogin")

			Expect(actual).To(Equal("userLogin"))
		})

		It("should return github login if slack returns error", func() {
			emails := []config.GithubEmailToSlackEmail{
				{
					GithubEmail: "userLogin",
					SlackEmail:  "user@user.com",
				},
			}
			service := user.NewService(slackMock, []string{"userLogin"}, emails, nil, nil)

			slackMock.EXPECT().GetUserIDByEmail("user@user.com").Return("", errors.New("not found"))

			actual := service.GetUserDescriptor("userLogin")

			Expect(actual).To(Equal("userLogin"))
		})
	})
})
