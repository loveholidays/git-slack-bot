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

package slack_test

import (
	"errors"
	"git-slack-bot/internal/config"
	"git-slack-bot/internal/slack"
	mock_slack "git-slack-bot/internal/slack/mocks"
	"testing"

	sl "github.com/slack-go/slack"
	"go.uber.org/mock/gomock"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestSlack(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Github tests")
}

var _ = Describe("Slack notifications", func() {
	var (
		mockCtrl   *gomock.Controller
		mockClient *mock_slack.MockClient
		connector  *slack.Connector
	)

	BeforeEach(func() {
		mockCtrl = gomock.NewController(GinkgoT())
		mockClient = mock_slack.NewMockClient(mockCtrl)
		cfg := config.SlackConfiguration{
			Token:     "AnyToken",
			ChannelID: "AnyID",
		}
		connector = slack.NewSlackConnector(cfg, mockClient)
	})

	Context("SendOrUpdateMessage", func() {
		It("should create a message when the stack has not been posted", func() {
			notification := slack.Notification{Key: "PR Stack 123:", Text: "PR Stack 123:\n- <https://github.com/loveholidays/flux/pull/42|Do XY> (open)"}
			history := &sl.GetConversationHistoryResponse{}
			parameters := &sl.GetConversationHistoryParameters{ChannelID: "AnyID"}
			mockClient.EXPECT().GetConversationHistory(parameters).Return(history, nil)
			mockClient.EXPECT().PostMessage("AnyID", gomock.Any()).Return("AnyID", "123.456", nil)

			connector.SendOrUpdateMessage(notification)
		})

		It("should update the existing stack message at its original timestamp", func() {
			notification := slack.Notification{Key: "PR Stack 123:", Text: "PR Stack 123:\n- Updated PR"}
			history := &sl.GetConversationHistoryResponse{Messages: []sl.Message{
				{Msg: sl.Msg{Text: "PR Stack 1230:", Timestamp: "111.111"}},
				{Msg: sl.Msg{Text: "PR Stack 123:\n- Original PR", Timestamp: "123.456"}},
			}}
			parameters := &sl.GetConversationHistoryParameters{ChannelID: "AnyID"}
			mockClient.EXPECT().GetConversationHistory(parameters).Return(history, nil)
			mockClient.EXPECT().UpdateMessage("AnyID", "123.456", gomock.Any()).Return("AnyID", "123.456", notification.Text, nil)

			connector.SendOrUpdateMessage(notification)
		})

		It("should leave an identical stack message unchanged", func() {
			notification := slack.Notification{Key: "PR Stack 123:", Text: "PR Stack 123:\n- Original PR"}
			history := &sl.GetConversationHistoryResponse{Messages: []sl.Message{{Msg: sl.Msg{Text: notification.Text, Timestamp: "123.456"}}}}
			parameters := &sl.GetConversationHistoryParameters{ChannelID: "AnyID"}
			mockClient.EXPECT().GetConversationHistory(parameters).Return(history, nil)

			connector.SendOrUpdateMessage(notification)
		})

		It("should stop without posting when history lookup fails", func() {
			notification := slack.Notification{Key: "PR Stack 123:", Text: "PR Stack 123:\n- Original PR"}
			lookupError := errors.New("Slack unavailable")
			parameters := &sl.GetConversationHistoryParameters{ChannelID: "AnyID"}
			mockClient.EXPECT().GetConversationHistory(parameters).Return(nil, lookupError)

			connector.SendOrUpdateMessage(notification)
		})
	})

	It("returns correct message from history", func() {
		response := sl.GetConversationHistoryResponse{
			Messages: []sl.Message{
				{
					Msg: sl.Msg{
						Text: "Some other message",
					},
				},
				{
					Msg: sl.Msg{
						Text: "Some message with the correct key",
					},
				},
			},
		}

		mockClient.EXPECT().GetConversationHistory(gomock.Any()).Return(&response, nil)

		message, err := connector.GetMessage("Some message")
		Expect(err).ToNot(HaveOccurred())
		Expect(message.Text).To(Equal("Some message with the correct key"))
	})
})
