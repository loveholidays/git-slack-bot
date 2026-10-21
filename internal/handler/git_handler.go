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
	slackConnector slack.Interactor
	messageBuilder messageBuilder.MessageBuilder
	userService    user.Service
	emoji          config.EmojiConfiguration
	ignoredRepos   []string
}

func NewGitHandler(slackConnector slack.Interactor, userService user.Service, emoji config.EmojiConfiguration, ignoredRepos []string) *GitHandler {
	return &GitHandler{
		slackConnector: slackConnector,
		messageBuilder: messageBuilder.MessageBuilder{},
		userService:    userService,
		emoji:          emoji,
		ignoredRepos:   ignoredRepos,
	}
}

func (g *GitHandler) HandlePullRequestEvent(body []byte) {
	event := parseGitEvent[gh.PullRequestEvent](body)
	if event == nil {
		return
	}
	pullRequest := event.PullRequest

	if !g.shouldTrackPullRequest(event.Repo, pullRequest) {
		return
	}

	switch event.GetAction() {
	case opened, readyForReview:
		g.notifyPullRequest(pullRequest)
	case assigned:
		if g.isTeamAssignment(event) {
			g.notifyPullRequest(pullRequest)
		}
	case closed:
		g.closePullRequest(pullRequest)
	case reopened:
		g.reopenPullRequest(pullRequest)
	}
}

func (g *GitHandler) reopenPullRequest(pullRequest *gh.PullRequest) {
	messageKey := fmt.Sprintf("<%s>", pullRequest.GetHTMLURL())
	slackMessage, err := g.slackConnector.GetMessage(messageKey)
	if err != nil {
		slog.Error("Could not find message", slog.Any("messageKey", messageKey), slog.Any("error", err))
		return
	}
	g.slackConnector.RemoveReactionFromMessage("x", slackMessage)
}

func (g *GitHandler) notifyPullRequest(pullRequest *gh.PullRequest) {
	if pullRequest.GetDraft() || pullRequest.GetState() == closed {
		return
	}
	messageKey := fmt.Sprintf("<%s>", pullRequest.GetHTMLURL())
	message := g.messageBuilder.BuildPRMessage(g.userService.GetUserDescriptor(pullRequest.GetUser().GetLogin()), pullRequest)
	g.slackConnector.SendMessageOnce(slack.Notification{Key: messageKey, Text: message, CreatedAt: pullRequest.GetCreatedAt().Time})
}

func (g *GitHandler) closePullRequest(pullRequest *gh.PullRequest) {
	if pullRequest.GetDraft() {
		return
	}
	messageKey := fmt.Sprintf("<%s>", pullRequest.GetHTMLURL())
	slackMessage, err := g.slackConnector.GetMessage(messageKey)
	if err != nil {
		slog.Error("Could not find message", slog.Any("messageKey", messageKey), slog.Any("error", err))
		return
	}
	if pullRequest.MergedAt != nil {
		g.slackConnector.AddReactionToMessage(g.emoji.Merge, slackMessage)
	} else {
		g.slackConnector.AddReactionToMessage(g.emoji.Close, slackMessage)
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

	if !g.shouldTrackPullRequest(event.Repo, pullRequest) {
		return
	}

	if g.userService.IsIgnoredReviewUser(*event.Review.User.Login) {
		return
	}

	switch event.GetAction() {
	case submitted:
		g.submitReview(pullRequest, event.Review)
	case dismissed:
		g.dismissReview(pullRequest)
	}
}

func (g *GitHandler) submitReview(pullRequest *gh.PullRequest, review *gh.PullRequestReview) {
	messageKey := fmt.Sprintf("<%s>", pullRequest.GetHTMLURL())
	slackMessage, err := g.slackConnector.GetMessage(messageKey)
	if err != nil {
		slog.Error("Could not find message", slog.Any("messageKey", messageKey), slog.Any("error", err))
		return
	}
	if review.GetState() == approved {
		g.slackConnector.AddReactionToMessage(g.emoji.Approve, slackMessage)
	}
	if review.GetBody() != "" {
		g.slackConnector.SendReply(slackMessage, g.messageBuilder.BuildPRReviewMessage(g.userService.GetUserDescriptor(review.GetUser().GetLogin()), review))
	}
}

func (g *GitHandler) dismissReview(pullRequest *gh.PullRequest) {
	messageKey := fmt.Sprintf("<%s>", pullRequest.GetHTMLURL())
	slackMessage, err := g.slackConnector.GetMessage(messageKey)
	if err != nil {
		slog.Error("Could not find message", slog.Any("messageKey", messageKey), slog.Any("error", err))
		return
	}
	g.slackConnector.RemoveReactionFromMessage(g.emoji.Approve, slackMessage)
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
	if !g.isRelevantComment(event.Repo, pullRequest.User, pullRequest.Assignees, commentAuthor) {
		return
	}
	userDescriptor := g.userService.GetUserDescriptor(commentAuthor)
	if event.Issue != nil {
		comment := &gh.IssueComment{User: event.Comment.User, Body: event.Comment.Body, HTMLURL: event.Comment.HTMLURL}
		message := g.messageBuilder.BuildIssueCommentMessage(userDescriptor, gh.IssueCommentEvent{Comment: comment})
		g.sendCommentReply(pullRequest.GetHTMLURL(), message)
		return
	}
	message := g.messageBuilder.BuildPRCommentMessage(userDescriptor, event.PullRequestReviewCommentEvent)
	g.sendCommentReply(pullRequest.GetHTMLURL(), message)
}

func (g *GitHandler) isRelevantComment(repository *gh.Repository, author *gh.User, assignees []*gh.User, commentAuthor string) bool {
	return !g.isIgnoredRepo(repository.GetName()) &&
		g.isRelevantPullRequest(repository, author, assignees) &&
		!g.userService.IsIgnoredCommentUser(commentAuthor)
}

func (g *GitHandler) sendCommentReply(pullRequestURL, message string) {
	messageKey := fmt.Sprintf("<%s>", pullRequestURL)
	slackMessage, err := g.slackConnector.GetMessage(messageKey)
	if err != nil {
		slog.Error("Could not find message", slog.Any("messageKey", messageKey), slog.Any("error", err))
		return
	}
	g.slackConnector.SendReply(slackMessage, message)
}

func (g *GitHandler) isIgnoredRepo(repoName string) bool {
	return slices.Contains(g.ignoredRepos, repoName)
}

func (g *GitHandler) isRelevantPullRequest(repository *gh.Repository, author *gh.User, assignees []*gh.User) bool {
	if g.userService.IsTeamMember(author.GetLogin()) {
		return true
	}
	if repository.GetFullName() != "loveholidays/flux" {
		return false
	}
	for _, assignee := range assignees {
		if g.userService.IsTeamMember(assignee.GetLogin()) {
			return true
		}
	}
	return false
}

func parseGitEvent[Event any](body []byte) *Event {
	var event Event
	if err := json.Unmarshal(body, &event); err != nil {
		slog.Error("Error parsing request body", slog.Any("error", err))
		return nil
	}
	return &event
}

func (g *GitHandler) isTeamAssignment(event *gh.PullRequestEvent) bool {
	return event.GetRepo().GetFullName() == "loveholidays/flux" &&
		g.userService.IsTeamMember(event.GetAssignee().GetLogin())
}

func (g *GitHandler) shouldTrackPullRequest(repository *gh.Repository, pullRequest *gh.PullRequest) bool {
	return !g.isIgnoredRepo(repository.GetName()) &&
		g.isRelevantPullRequest(repository, pullRequest.GetUser(), pullRequest.Assignees)
}
