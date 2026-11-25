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

package handler

import (
	"encoding/json"
	"git-slack-bot/internal/config"
	messageBuilder "git-slack-bot/internal/messagebuilder"
	"git-slack-bot/internal/slack"
	"git-slack-bot/internal/user"
	"log/slog"
	"slices"

	gh "github.com/google/go-github/v93/github"
)

const (
	closed         string = "closed"
	opened         string = "opened"
	assigned       string = "assigned"
	readyForReview string = "ready_for_review"
	reopened       string = "reopened"
	submitted      string = "submitted"
	dismissed      string = "dismissed"
	approved       string = "approved"
)

type GitEventHandler interface {
	HandlePullRequestEvent(body []byte)
	HandlePullRequestReviewEvent(body []byte)
	HandlePullRequestReviewCommentEvent(body []byte)
	HandleIssueCommentEvent(body []byte)
}

type PullRequestNotifier interface {
	NotifyPullRequestOpened(connector slack.Interactor, pullRequest *gh.PullRequest)
	NotifyPullRequestClosed(connector slack.Interactor, pullRequest *gh.PullRequest)
	NotifyPullRequestReopened(connector slack.Interactor, pullRequest *gh.PullRequest)
	NotifyReviewSubmitted(connector slack.Interactor, pullRequest *gh.PullRequest, review *gh.PullRequestReview)
	NotifyReviewDismissed(connector slack.Interactor, pullRequest *gh.PullRequest)
	NotifyComment(connector slack.Interactor, pullRequest *gh.PullRequest, message string)
}

type GitHandler struct {
	slackConnector    slack.Interactor
	messageBuilder    messageBuilder.MessageBuilder
	userService       user.Service
	notifier          PullRequestNotifier
	notificationRules []config.NotificationRule
	ignoredRepos      []string
}

func NewGitHandler(slackConnector slack.Interactor, userService user.Service, notifier PullRequestNotifier, configuration config.Configuration) *GitHandler {
	return &GitHandler{
		slackConnector:    slackConnector,
		messageBuilder:    messageBuilder.MessageBuilder{},
		userService:       userService,
		notifier:          notifier,
		notificationRules: configuration.NotificationRules(),
		ignoredRepos:      configuration.GitHub.IgnoredRepos,
	}
}

func (g *GitHandler) HandlePullRequestEvent(body []byte) {
	event := parseGitEvent[gh.PullRequestEvent](body)
	if event == nil {
		return
	}
	pullRequest := event.PullRequest

	if !g.shouldHandlePullRequestEvent(event) {
		return
	}

	for _, connector := range g.notificationDestinations(event.Repo, pullRequest, event.GetAction() == assigned) {
		switch event.GetAction() {
		case closed:
			g.notifier.NotifyPullRequestClosed(connector, pullRequest)
		case reopened:
			g.notifier.NotifyPullRequestReopened(connector, pullRequest)
		default:
			g.notifier.NotifyPullRequestOpened(connector, pullRequest)
		}
	}
}

func (g *GitHandler) HandlePullRequestReviewEvent(body []byte) {
	var event gh.PullRequestReviewEvent
	err := json.Unmarshal(body, &event)
	if err != nil {
		slog.Error("Error parsing request body", slog.Any("body", string(body)), slog.Any("error", err))
		return
	}
	pullRequest := event.PullRequest

	if g.isIgnoredRepo(event.Repo.GetName()) || g.userService.IsIgnoredReviewUser(event.Review.GetUser().GetLogin()) {
		return
	}

	for _, connector := range g.notificationDestinations(event.Repo, pullRequest, false) {
		switch event.GetAction() {
		case submitted:
			g.notifier.NotifyReviewSubmitted(connector, pullRequest, event.Review)
		case dismissed:
			g.notifier.NotifyReviewDismissed(connector, pullRequest)
		}
	}
}

type commentEvent struct {
	gh.PullRequestReviewCommentEvent
	Issue *gh.Issue `json:"issue"`
}

func (g *GitHandler) HandlePullRequestReviewCommentEvent(body []byte) {
	g.handleCommentEvent(body)
}

func (g *GitHandler) HandleIssueCommentEvent(body []byte) {
	g.handleCommentEvent(body)
}

func (g *GitHandler) handleCommentEvent(body []byte) {
	event := parseGitEvent[commentEvent](body)
	if event == nil {
		return
	}
	pullRequest := event.GetPullRequest()
	if pullRequest == nil {
		if event.Issue == nil {
			return
		}
		pullRequest = &gh.PullRequest{User: event.Issue.User, Assignees: event.Issue.Assignees, HTMLURL: event.Issue.HTMLURL}
	}
	commentAuthor := event.GetComment().GetUser().GetLogin()
	if g.isIgnoredRepo(event.Repo.GetName()) || g.userService.IsIgnoredCommentUser(commentAuthor) {
		return
	}
	for _, connector := range g.notificationDestinations(event.Repo, pullRequest, false) {
		g.notifier.NotifyComment(connector, pullRequest, g.commentMessage(event))
	}
}

func (g *GitHandler) commentMessage(event *commentEvent) string {
	userDescriptor := g.userService.GetUserDescriptor(event.GetComment().GetUser().GetLogin())
	if event.Issue != nil {
		comment := &gh.IssueComment{User: event.Comment.User, Body: event.Comment.Body, HTMLURL: event.Comment.HTMLURL}
		return g.messageBuilder.BuildIssueCommentMessage(userDescriptor, gh.IssueCommentEvent{Comment: comment})
	}
	return g.messageBuilder.BuildPRCommentMessage(userDescriptor, event.PullRequestReviewCommentEvent)
}

func (g *GitHandler) isIgnoredRepo(repoName string) bool {
	return slices.Contains(g.ignoredRepos, repoName)
}

func parseGitEvent[Event any](body []byte) *Event {
	var event Event
	if err := json.Unmarshal(body, &event); err != nil {
		slog.Error("Error parsing request body", slog.Any("error", err))
		return nil
	}
	return &event
}

func (g *GitHandler) shouldHandlePullRequestEvent(event *gh.PullRequestEvent) bool {
	switch event.GetAction() {
	case opened, readyForReview, assigned:
		if event.PullRequest.GetDraft() || event.PullRequest.GetState() == closed {
			return false
		}
		return event.GetAction() != assigned || g.userService.IsTeamMember(event.GetAssignee().GetLogin())
	case closed, reopened:
		return true
	default:
		return false
	}
}
