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
	"errors"
	"fmt"
	"git-slack-bot/internal/config"
	"git-slack-bot/internal/github"
	"git-slack-bot/internal/slack"
	"git-slack-bot/internal/user"
	"log/slog"

	gh "github.com/google/go-github/v93/github"
	sl "github.com/slack-go/slack"
)

//go:generate mockgen -destination=./mocks/stacked_pr_notifier.go . StackMessageBuilder

type StackMessageBuilder interface {
	BuildPRStackMessage(stack gh.PullRequestStackDetails, previousText string, emoji config.EmojiConfiguration) string
	BuildPRStackStatusMessage(message *sl.Message, pullRequest gh.PullRequest, approved bool, emoji config.EmojiConfiguration) string
	BuildPRStackReviewMessage(userDescriptor string, pullRequest gh.PullRequest, review gh.PullRequestReview) string
}

type stackedPRNotifier struct {
	githubConnector github.Interactor
	messageBuilder  StackMessageBuilder
	userService     user.Service
	emoji           config.EmojiConfiguration
}

func NewStackedPRNotifier(githubConnector github.Interactor, messageBuilder StackMessageBuilder, userService user.Service, emoji config.EmojiConfiguration) PullRequestNotifier {
	return &stackedPRNotifier{
		githubConnector: githubConnector,
		messageBuilder:  messageBuilder,
		userService:     userService,
		emoji:           emoji,
	}
}

func (n *stackedPRNotifier) NotifyPullRequestOpened(connector slack.Interactor, pullRequest *gh.PullRequest) {
	repository := pullRequest.GetBase().GetRepo()
	stack, err := n.githubConnector.GetStack(repository.GetOwner().GetLogin(), repository.GetName(), pullRequest.Stack.GetNumber())
	if err != nil {
		slog.Error("Failed to retrieve pull request stack", slog.Any("error", err))
		return
	}
	if stack == nil {
		return
	}
	messageKey := fmt.Sprintf("PR Stack %d:", pullRequest.Stack.GetID())
	message, err := connector.GetMessage(messageKey)
	if err != nil && !errors.Is(err, slack.ErrMessageNotFound) {
		slog.Error("Could not find message", slog.Any("messageKey", messageKey), slog.Any("error", err))
		return
	}
	previousText := ""
	if message != nil {
		previousText = message.Text
	}
	notification := slack.Notification{
		Key:  messageKey,
		Text: n.messageBuilder.BuildPRStackMessage(*stack, previousText, n.emoji),
	}
	connector.SendOrUpdateMessage(notification)
}

func (n *stackedPRNotifier) NotifyPullRequestClosed(connector slack.Interactor, pullRequest *gh.PullRequest) {
	if pullRequest.GetDraft() {
		return
	}
	n.updatePullRequestStatus(connector, pullRequest)
}

func (n *stackedPRNotifier) NotifyPullRequestReopened(connector slack.Interactor, pullRequest *gh.PullRequest) {
	n.updatePullRequestStatus(connector, pullRequest)
}

func (n *stackedPRNotifier) updatePullRequestStatus(connector slack.Interactor, pullRequest *gh.PullRequest) {
	message := n.findStackMessage(connector, pullRequest)
	if message == nil {
		return
	}
	n.updateStackMessage(connector, message, pullRequest, false)
}

func (n *stackedPRNotifier) updateStackMessage(connector slack.Interactor, message *sl.Message, pullRequest *gh.PullRequest, approved bool) {
	notification := slack.Notification{
		Key:  fmt.Sprintf("PR Stack %d:", pullRequest.Stack.GetID()),
		Text: n.messageBuilder.BuildPRStackStatusMessage(message, *pullRequest, approved, n.emoji),
	}
	connector.SendOrUpdateMessage(notification)
}

func (n *stackedPRNotifier) NotifyReviewSubmitted(connector slack.Interactor, pullRequest *gh.PullRequest, review *gh.PullRequestReview) {
	message := n.findStackMessage(connector, pullRequest)
	if message == nil {
		return
	}
	if review.GetState() == approved {
		n.updateStackMessage(connector, message, pullRequest, true)
	}
	if review.GetBody() != "" {
		reply := n.messageBuilder.BuildPRStackReviewMessage(n.userService.GetUserDescriptor(review.GetUser().GetLogin()), *pullRequest, *review)
		connector.SendReply(message, reply)
	}
}

func (n *stackedPRNotifier) NotifyReviewDismissed(connector slack.Interactor, pullRequest *gh.PullRequest) {
	message := n.findStackMessage(connector, pullRequest)
	if message == nil {
		return
	}
	n.updateStackMessage(connector, message, pullRequest, false)
}

func (n *stackedPRNotifier) NotifyComment(connector slack.Interactor, pullRequest *gh.PullRequest, comment string) {
	message := n.findStackMessage(connector, pullRequest)
	if message == nil {
		return
	}
	reply := fmt.Sprintf("<%s|PR #%d>:\n%s", pullRequest.GetHTMLURL(), pullRequest.GetNumber(), comment)
	connector.SendReply(message, reply)
}

func (n *stackedPRNotifier) findStackMessage(connector slack.Interactor, pullRequest *gh.PullRequest) *sl.Message {
	messageKey := fmt.Sprintf("PR Stack %d:", pullRequest.Stack.GetID())
	message, err := connector.GetMessage(messageKey)
	if err != nil {
		slog.Error("Could not find message", slog.Any("messageKey", messageKey), slog.Any("error", err))
		return nil
	}
	return message
}
