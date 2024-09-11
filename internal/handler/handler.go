package handler

import (
	"log/slog"
	"net/http"

	gh "github.com/google/go-github/v56/github"
)

type GithubPullRequestAction string

type Header string

const (
	pullRequestEvent              string = "pull_request"
	pullRequestReviewEvent        string = "pull_request_review"
	pullRequestReviewCommentEvent string = "pull_request_review_comment"
	issueCommentEvent             string = "issue_comment"
)

type WebhookHandler struct {
	secretKey  []byte
	gitHandler GitEventHandler
}

func NewWebhookEventHandler(secretKey []byte, gitHandler GitEventHandler) *WebhookHandler {
	return &WebhookHandler{
		secretKey:  secretKey,
		gitHandler: gitHandler,
	}
}

func (h *WebhookHandler) HandleHeathCheck(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func (h *WebhookHandler) HandleWebhook(w http.ResponseWriter, r *http.Request) {
	body, err := gh.ValidatePayload(r, h.secretKey)
	if err != nil {
		slog.Error("Error validating message", slog.Any("error", err))
		http.Error(w, "Error reading request body", http.StatusInternalServerError)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	slog.Debug("webhook", slog.String("github-event", r.Header.Get("X-GitHub-Event")), slog.String("body", string(body)))

	switch r.Header.Get("X-GitHub-Event") {
	case pullRequestEvent:
		h.gitHandler.HandlePullRequestEvent(body)
	case pullRequestReviewEvent:
		h.gitHandler.HandlePullRequestReviewEvent(body)
	case pullRequestReviewCommentEvent:
		h.gitHandler.HandlePullRequestReviewCommentEvent(body)
	case issueCommentEvent:
		h.gitHandler.HandleIssueCommentEvent(body)
	}
	w.WriteHeader(http.StatusOK)
}
