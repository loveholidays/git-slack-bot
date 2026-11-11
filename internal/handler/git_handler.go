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
	"fmt"
	"git-slack-bot/internal/config"
	messageBuilder "git-slack-bot/internal/messagebuilder"
	"git-slack-bot/internal/slack"
	"git-slack-bot/internal/user"
	"log/slog"
	"slices"

	gh "github.com/google/go-github/v56/github"
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

type GitHandler struct {
	slackConnector    slack.Interactor
	messageBuilder    messageBuilder.MessageBuilder
	userService       user.Service
	emoji             config.EmojiConfiguration
	notificationRules []config.NotificationRule
	ignoredRepos      []string
}

func NewGitHandler(slackConnector slack.Interactor, userService user.Service, emoji config.EmojiConfiguration, configuration config.Configuration) *GitHandler {
	return &GitHandler{
		slackConnector:    slackConnector,
		messageBuilder:    messageBuilder.MessageBuilder{},
		userService:       userService,
		emoji:             emoji,
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
			g.closePullRequest(connector, pullRequest)
		case reopened:
			g.reopenPullRequest(connector, pullRequest)
		default:
			g.notifyPullRequest(connector, pullRequest)
		}
	}
}

func (g *GitHandler) reopenPullRequest(connector slack.Interactor, pullRequest *gh.PullRequest) {
	messageKey := fmt.Sprintf("<%s>", pullRequest.GetHTMLURL())
	slackMessage, err := connector.GetMessage(messageKey)
	if err != nil {
		slog.Error("Could not find message", slog.Any("messageKey", messageKey), slog.Any("error", err))
		return
	}
	connector.RemoveReactionFromMessage("x", slackMessage)
}

func (g *GitHandler) notifyPullRequest(connector slack.Interactor, pullRequest *gh.PullRequest) {
	if pullRequest.GetDraft() || pullRequest.GetState() == closed {
		return
	}
	messageKey := fmt.Sprintf("<%s>", pullRequest.GetHTMLURL())
	message := g.messageBuilder.BuildPRMessage(g.userService.GetUserDescriptor(pullRequest.GetUser().GetLogin()), pullRequest)
	connector.SendMessageOnce(slack.Notification{Key: messageKey, Text: message, CreatedAt: pullRequest.GetCreatedAt().Time})
}

func (g *GitHandler) closePullRequest(connector slack.Interactor, pullRequest *gh.PullRequest) {
	if pullRequest.GetDraft() {
		return
	}
	messageKey := fmt.Sprintf("<%s>", pullRequest.GetHTMLURL())
	slackMessage, err := connector.GetMessage(messageKey)
	if err != nil {
		slog.Error("Could not find message", slog.Any("messageKey", messageKey), slog.Any("error", err))
		return
	}
	if pullRequest.MergedAt != nil {
		connector.AddReactionToMessage(g.emoji.Merge, slackMessage)
	} else {
		connector.AddReactionToMessage(g.emoji.Close, slackMessage)
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
			g.submitReview(connector, pullRequest, event.Review)
		case dismissed:
			g.dismissReview(connector, pullRequest)
		}
	}
}

func (g *GitHandler) submitReview(connector slack.Interactor, pullRequest *gh.PullRequest, review *gh.PullRequestReview) {
	messageKey := fmt.Sprintf("<%s>", pullRequest.GetHTMLURL())
	slackMessage, err := connector.GetMessage(messageKey)
	if err != nil {
		slog.Error("Could not find message", slog.Any("messageKey", messageKey), slog.Any("error", err))
		return
	}
	if review.GetState() == approved {
		connector.AddReactionToMessage(g.emoji.Approve, slackMessage)
	}
	if review.GetBody() != "" {
		connector.SendReply(slackMessage, g.messageBuilder.BuildPRReviewMessage(g.userService.GetUserDescriptor(review.GetUser().GetLogin()), review))
	}
}

func (g *GitHandler) dismissReview(connector slack.Interactor, pullRequest *gh.PullRequest) {
	messageKey := fmt.Sprintf("<%s>", pullRequest.GetHTMLURL())
	slackMessage, err := connector.GetMessage(messageKey)
	if err != nil {
		slog.Error("Could not find message", slog.Any("messageKey", messageKey), slog.Any("error", err))
		return
	}
	connector.RemoveReactionFromMessage(g.emoji.Approve, slackMessage)
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
		g.sendCommentReply(connector, pullRequest.GetHTMLURL(), g.commentMessage(event))
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

func (g *GitHandler) sendCommentReply(connector slack.Interactor, pullRequestURL, message string) {
	messageKey := fmt.Sprintf("<%s>", pullRequestURL)
	slackMessage, err := connector.GetMessage(messageKey)
	if err != nil {
		slog.Error("Could not find message", slog.Any("messageKey", messageKey), slog.Any("error", err))
		return
	}
	connector.SendReply(slackMessage, message)
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
