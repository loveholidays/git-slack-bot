package slack_test

import (
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

var _ = Describe("GetMessage", func() {
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
