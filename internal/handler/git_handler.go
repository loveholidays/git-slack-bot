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
	readyForReview string = "ready_for_review"
	reopened       string = "reopened"
	submitted      string = "submitted"
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
	var event gh.PullRequestEvent
	err := json.Unmarshal(body, &event)
	if err != nil {
		slog.Error("Error parsing request body", slog.Any("error", err), slog.String("body", string(body)))
		return
	}
	pullRequest := event.PullRequest

	if g.isIgnoredRepo(*event.Repo.Name) {
		return
	}

	if !g.userService.IsTeamMember(*pullRequest.User.Login) {
		return
	}

	switch *event.Action {
	case opened, readyForReview:
		if pullRequest.Draft != nil && *pullRequest.Draft {
			return
		}
		githubLogin := *pullRequest.User.Login
		g.slackConnector.SendMessage(g.messageBuilder.BuildPRMessage(g.userService.GetUserDescriptor(githubLogin), pullRequest))
	case closed:
		if pullRequest.Draft != nil && *pullRequest.Draft {
			return
		}
		messageKey := fmt.Sprintf("<%s>", *pullRequest.HTMLURL)
		slackMessage, err := g.slackConnector.GetMessage(messageKey)
		if err != nil {
			slog.Error("Could not find message", slog.Any("messageKey", messageKey), slog.Any("error", err))
			return
		}
		if event.PullRequest.MergedAt != nil {
			g.slackConnector.AddReactionToMessage(g.emoji.Merge, slackMessage)
		} else {
			g.slackConnector.AddReactionToMessage(g.emoji.Close, slackMessage)
		}
	case reopened:
		messageKey := fmt.Sprintf("<%s>", *pullRequest.HTMLURL)
		slackMessage, err := g.slackConnector.GetMessage(messageKey)
		if err != nil {
			slog.Error("Could not find message", slog.Any("messageKey", messageKey), slog.Any("error", err))
			return
		}
		g.slackConnector.RemoveReactionFromMessage("x", slackMessage)
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

	if g.isIgnoredRepo(*event.Repo.Name) {
		return
	}

	if !g.userService.IsTeamMember(*pullRequest.User.Login) {
		return
	}

	if g.userService.IsIgnoredReviewUser(*event.Review.User.Login) {
		return
	}

	if *event.Action != submitted || *event.Review.State != approved {
		return
	}
	messageKey := fmt.Sprintf("<%s>", *pullRequest.HTMLURL)
	slackMessage, err := g.slackConnector.GetMessage(messageKey)
	if err != nil {
		slog.Error("Could not find message", slog.Any("messageKey", messageKey), slog.Any("error", err))
		return
	}
	g.slackConnector.AddReactionToMessage(g.emoji.Approve, slackMessage)
}

func (g *GitHandler) HandlePullRequestReviewCommentEvent(body []byte) {
	var event gh.PullRequestReviewCommentEvent
	err := json.Unmarshal(body, &event)
	if err != nil {
		slog.Error("Error parsing request body", slog.Any("body", string(body)), slog.Any("error", err))
		return
	}
	pullRequest := event.PullRequest

	if g.isIgnoredRepo(*event.Repo.Name) {
		return
	}

	if !g.userService.IsTeamMember(*pullRequest.User.Login) {
		return
	}

	if g.userService.IsIgnoredCommentUser(*event.Comment.User.Login) {
		return
	}

	messageKey := fmt.Sprintf("<%s>", *pullRequest.HTMLURL)
	slackMessage, err := g.slackConnector.GetMessage(messageKey)
	if err != nil {
		slog.Error("Could not find message", slog.Any("messageKey", messageKey), slog.Any("error", err))
		return
	}
	g.slackConnector.SendReply(slackMessage, g.messageBuilder.BuildPRCommentMessage(g.userService.GetUserDescriptor(*event.Comment.User.Login), event))
}

func (g *GitHandler) HandleIssueCommentEvent(body []byte) {
	var event gh.IssueCommentEvent
	err := json.Unmarshal(body, &event)
	if err != nil {
		slog.Error("Error parsing request body", slog.Any("body", string(body)), slog.Any("error", err))
		return
	}

	if g.isIgnoredRepo(*event.Repo.Name) {
		return
	}

	if !g.userService.IsTeamMember(*event.Issue.User.Login) {
		return
	}

	if g.userService.IsIgnoredCommentUser(*event.Comment.User.Login) {
		return
	}

	messageKey := fmt.Sprintf("<%s>", *event.Issue.HTMLURL)
	slackMessage, err := g.slackConnector.GetMessage(messageKey)
	if err != nil {
		slog.Error("Could not find message", slog.Any("messageKey", messageKey), slog.Any("error", err))
		return
	}
	g.slackConnector.SendReply(slackMessage, g.messageBuilder.BuildIssueCommentMessage(g.userService.GetUserDescriptor(*event.Comment.User.Login), event))
}

func (g *GitHandler) isIgnoredRepo(repoName string) bool {
	return slices.Contains(g.ignoredRepos, repoName)
}
