package handler_test

import (
	"bytes"
	"git-slack-bot/internal/handler"
	mock_handler "git-slack-bot/internal/handler/mocks"
	"net/http"
	"net/http/httptest"
	"testing"

	"go.uber.org/mock/gomock"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestHandler(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Github tests")
}

var _ = Describe("HandleWebhook", func() {
	var (
		gitHandlerMock *mock_handler.MockGitEventHandler
	)

	BeforeEach(func() {
		gitHandlerMock = mock_handler.NewMockGitEventHandler(gomock.NewController(GinkgoT()))
	})

	It("should handle pull request event", func() {
		webhookHandler := handler.NewWebhookEventHandler([]byte("It's a Secret to Everybody"), gitHandlerMock)

		headers := http.Header{}
		headers.Add("Content-Type", "application/json")
		headers.Add("X-Hub-Signature", "sha256=757107ea0eb2509fc211221cce984b8a37570b6d7586c22c46f4379c8b043e17")
		headers.Add("X-Github-Event", "pull_request")

		body := []byte("Hello, World!")

		request, err := http.NewRequest(http.MethodPost, "process-git-event", bytes.NewReader(body))
		request.Header = headers
		Expect(err).ToNot(HaveOccurred())

		writer := httptest.NewRecorder()

		gitHandlerMock.EXPECT().HandlePullRequestEvent(body).Times(1)
		gitHandlerMock.EXPECT().HandlePullRequestReviewEvent(body).Times(0)
		gitHandlerMock.EXPECT().HandlePullRequestReviewCommentEvent(body).Times(0)
		gitHandlerMock.EXPECT().HandleIssueCommentEvent(body).Times(0)

		webhookHandler.HandleWebhook(writer, request)

		Expect(writer.Code).To(Equal(http.StatusOK))
	})

	It("should handle pull request review event", func() {
		webhookHandler := handler.NewWebhookEventHandler([]byte("It's a Secret to Everybody"), gitHandlerMock)

		headers := http.Header{}
		headers.Add("Content-Type", "application/json")
		headers.Add("X-Hub-Signature", "sha256=757107ea0eb2509fc211221cce984b8a37570b6d7586c22c46f4379c8b043e17")
		headers.Add("X-Github-Event", "pull_request_review")

		body := []byte("Hello, World!")

		request, err := http.NewRequest(http.MethodPost, "process-git-event", bytes.NewReader(body))
		request.Header = headers
		Expect(err).ToNot(HaveOccurred())

		writer := httptest.NewRecorder()

		gitHandlerMock.EXPECT().HandlePullRequestEvent(body).Times(0)
		gitHandlerMock.EXPECT().HandlePullRequestReviewEvent(body).Times(1)
		gitHandlerMock.EXPECT().HandlePullRequestReviewCommentEvent(body).Times(0)
		gitHandlerMock.EXPECT().HandleIssueCommentEvent(body).Times(0)

		webhookHandler.HandleWebhook(writer, request)

		Expect(writer.Code).To(Equal(http.StatusOK))
	})

	It("should handle pull request review comment event", func() {
		webhookHandler := handler.NewWebhookEventHandler([]byte("It's a Secret to Everybody"), gitHandlerMock)

		headers := http.Header{}
		headers.Add("Content-Type", "application/json")
		headers.Add("X-Hub-Signature", "sha256=757107ea0eb2509fc211221cce984b8a37570b6d7586c22c46f4379c8b043e17")
		headers.Add("X-Github-Event", "pull_request_review_comment")

		body := []byte("Hello, World!")

		request, err := http.NewRequest(http.MethodPost, "process-git-event", bytes.NewReader(body))
		request.Header = headers
		Expect(err).ToNot(HaveOccurred())

		writer := httptest.NewRecorder()

		gitHandlerMock.EXPECT().HandlePullRequestEvent(body).Times(0)
		gitHandlerMock.EXPECT().HandlePullRequestReviewEvent(body).Times(0)
		gitHandlerMock.EXPECT().HandlePullRequestReviewCommentEvent(body).Times(1)
		gitHandlerMock.EXPECT().HandleIssueCommentEvent(body).Times(0)

		webhookHandler.HandleWebhook(writer, request)

		Expect(writer.Code).To(Equal(http.StatusOK))
	})

	It("should handle issue comment event", func() {
		webhookHandler := handler.NewWebhookEventHandler([]byte("It's a Secret to Everybody"), gitHandlerMock)

		headers := http.Header{}
		headers.Add("Content-Type", "application/json")
		headers.Add("X-Hub-Signature", "sha256=757107ea0eb2509fc211221cce984b8a37570b6d7586c22c46f4379c8b043e17")
		headers.Add("X-Github-Event", "issue_comment")

		body := []byte("Hello, World!")

		request, err := http.NewRequest(http.MethodPost, "process-git-event", bytes.NewReader(body))
		request.Header = headers
		Expect(err).ToNot(HaveOccurred())

		writer := httptest.NewRecorder()

		gitHandlerMock.EXPECT().HandlePullRequestEvent(body).Times(0)
		gitHandlerMock.EXPECT().HandlePullRequestReviewEvent(body).Times(0)
		gitHandlerMock.EXPECT().HandlePullRequestReviewCommentEvent(body).Times(0)
		gitHandlerMock.EXPECT().HandleIssueCommentEvent(body).Times(1)

		webhookHandler.HandleWebhook(writer, request)

		Expect(writer.Code).To(Equal(http.StatusOK))
	})
})
