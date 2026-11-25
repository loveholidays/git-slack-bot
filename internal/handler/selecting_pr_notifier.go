package handler

import (
	"git-slack-bot/internal/github"
	"git-slack-bot/internal/slack"
	"log/slog"

	gh "github.com/google/go-github/v93/github"
)

type selectingPRNotifier struct {
	single          PullRequestNotifier
	stacked         PullRequestNotifier
	githubConnector github.Interactor
}

func NewSelectingPRNotifier(single, stacked PullRequestNotifier, githubConnector github.Interactor) PullRequestNotifier {
	return &selectingPRNotifier{single: single, stacked: stacked, githubConnector: githubConnector}
}

func (n *selectingPRNotifier) NotifyPullRequestOpened(connector slack.Interactor, pullRequest *gh.PullRequest) {
	n.notifierFor(pullRequest).NotifyPullRequestOpened(connector, pullRequest)
}

func (n *selectingPRNotifier) NotifyPullRequestClosed(connector slack.Interactor, pullRequest *gh.PullRequest) {
	n.notifierFor(pullRequest).NotifyPullRequestClosed(connector, pullRequest)
}

func (n *selectingPRNotifier) NotifyPullRequestReopened(connector slack.Interactor, pullRequest *gh.PullRequest) {
	n.notifierFor(pullRequest).NotifyPullRequestReopened(connector, pullRequest)
}

func (n *selectingPRNotifier) NotifyReviewSubmitted(connector slack.Interactor, pullRequest *gh.PullRequest, review *gh.PullRequestReview) {
	n.notifierFor(pullRequest).NotifyReviewSubmitted(connector, pullRequest, review)
}

func (n *selectingPRNotifier) NotifyReviewDismissed(connector slack.Interactor, pullRequest *gh.PullRequest) {
	n.notifierFor(pullRequest).NotifyReviewDismissed(connector, pullRequest)
}

func (n *selectingPRNotifier) NotifyComment(connector slack.Interactor, pullRequest *gh.PullRequest, message string) {
	resolved, err := n.commentPullRequest(pullRequest)
	if err != nil {
		slog.Error("Could not retrieve pull request", slog.Any("error", err))
		return
	}
	if resolved == nil {
		return
	}
	n.notifierFor(resolved).NotifyComment(connector, resolved, message)
}

func (n *selectingPRNotifier) notifierFor(pullRequest *gh.PullRequest) PullRequestNotifier {
	if pullRequest.Stack != nil {
		return n.stacked
	}
	return n.single
}

func (n *selectingPRNotifier) commentPullRequest(pullRequest *gh.PullRequest) (*gh.PullRequest, error) {
	if pullRequest.Stack != nil || pullRequest.Base != nil {
		return pullRequest, nil
	}
	return n.githubConnector.GetPullRequest(pullRequest.GetHTMLURL())
}
