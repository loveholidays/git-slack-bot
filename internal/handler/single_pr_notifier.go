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
	"fmt"
	"git-slack-bot/internal/config"
	messageBuilder "git-slack-bot/internal/messagebuilder"
	"git-slack-bot/internal/slack"
	"git-slack-bot/internal/user"
	"log/slog"

	gh "github.com/google/go-github/v93/github"
)

type singlePRNotifier struct {
	messageBuilder messageBuilder.MessageBuilder
	userService    user.Service
	emoji          config.EmojiConfiguration
}

func NewSinglePRNotifier(userService user.Service, emoji config.EmojiConfiguration) PullRequestNotifier {
	return &singlePRNotifier{
		messageBuilder: messageBuilder.MessageBuilder{},
		userService:    userService,
		emoji:          emoji,
	}
}

func (n *singlePRNotifier) NotifyPullRequestReopened(connector slack.Interactor, pullRequest *gh.PullRequest) {
	messageKey := fmt.Sprintf("<%s>", pullRequest.GetHTMLURL())
	slackMessage, err := connector.GetMessage(messageKey)
	if err != nil {
		slog.Error("Could not find message", slog.Any("messageKey", messageKey), slog.Any("error", err))
		return
	}
	connector.RemoveReactionFromMessage("x", slackMessage)
}

func (n *singlePRNotifier) NotifyPullRequestOpened(connector slack.Interactor, pullRequest *gh.PullRequest) {
	messageKey := fmt.Sprintf("<%s>", pullRequest.GetHTMLURL())
	message := n.messageBuilder.BuildPRMessage(n.userService.GetUserDescriptor(pullRequest.GetUser().GetLogin()), pullRequest)
	connector.SendMessageOnce(slack.Notification{Key: messageKey, Text: message, CreatedAt: pullRequest.GetCreatedAt().Time})
}

func (n *singlePRNotifier) NotifyPullRequestClosed(connector slack.Interactor, pullRequest *gh.PullRequest) {
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
		connector.AddReactionToMessage(n.emoji.Merge, slackMessage)
	} else {
		connector.AddReactionToMessage(n.emoji.Close, slackMessage)
	}
}

func (n *singlePRNotifier) NotifyReviewSubmitted(connector slack.Interactor, pullRequest *gh.PullRequest, review *gh.PullRequestReview) {
	messageKey := fmt.Sprintf("<%s>", pullRequest.GetHTMLURL())
	slackMessage, err := connector.GetMessage(messageKey)
	if err != nil {
		slog.Error("Could not find message", slog.Any("messageKey", messageKey), slog.Any("error", err))
		return
	}
	if review.GetState() == approved {
		connector.AddReactionToMessage(n.emoji.Approve, slackMessage)
	}
	if review.GetBody() != "" {
		connector.SendReply(slackMessage, n.messageBuilder.BuildPRReviewMessage(n.userService.GetUserDescriptor(review.GetUser().GetLogin()), review))
	}
}

func (n *singlePRNotifier) NotifyReviewDismissed(connector slack.Interactor, pullRequest *gh.PullRequest) {
	messageKey := fmt.Sprintf("<%s>", pullRequest.GetHTMLURL())
	slackMessage, err := connector.GetMessage(messageKey)
	if err != nil {
		slog.Error("Could not find message", slog.Any("messageKey", messageKey), slog.Any("error", err))
		return
	}
	connector.RemoveReactionFromMessage(n.emoji.Approve, slackMessage)
}

func (n *singlePRNotifier) NotifyComment(connector slack.Interactor, pullRequest *gh.PullRequest, message string) {
	messageKey := fmt.Sprintf("<%s>", pullRequest.GetHTMLURL())
	slackMessage, err := connector.GetMessage(messageKey)
	if err != nil {
		slog.Error("Could not find message", slog.Any("messageKey", messageKey), slog.Any("error", err))
		return
	}
	connector.SendReply(slackMessage, message)
}
