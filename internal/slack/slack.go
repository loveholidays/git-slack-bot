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

package slack

//go:generate mockgen -destination=./mocks/slack.go . Client,Interactor

import (
	"git-slack-bot/internal/config"
	"github.com/slack-go/slack"
	"log/slog"
	"sync"
)

type Client interface {
	GetConversationReplies(params *slack.GetConversationRepliesParameters) ([]slack.Message, bool, string, error)
	UpdateMessage(channelID, timestamp string, options ...slack.MsgOption) (string, string, string, error)
	DeleteMessage(channelID, timestamp string) (string, string, error)
	GetConversationHistory(params *slack.GetConversationHistoryParameters) (*slack.GetConversationHistoryResponse, error)
	PostMessage(channelID string, options ...slack.MsgOption) (string, string, error)
	AddReaction(name string, item slack.ItemRef) error
	RemoveReaction(name string, item slack.ItemRef) error
	GetUserByEmail(email string) (*slack.User, error)
}

type Interactor interface {
	SendMessageOnce(notification Notification)
	SendReply(slackMessage *slack.Message, message string)
	UpdateReply(parent *slack.Message, commentURL, message string)
	DeleteReply(parent *slack.Message, commentURL string)
	AddReactionToMessage(reaction string, message *slack.Message)
	RemoveReactionFromMessage(reaction string, message *slack.Message)
	GetMessage(messageKey string) (*slack.Message, error)
	GetUserIDByEmail(email string) (string, error)
}

type Connector struct {
	notificationMutex sync.Mutex
	client            Client
	channelID         string
}

func NewSlackConnector(cfg config.SlackConfiguration, client Client) *Connector {
	return &Connector{
		client:    client,
		channelID: cfg.ChannelID,
	}
}

func (sc *Connector) SendMessage(message string) {
	_, _, err := sc.client.PostMessage(sc.channelID, slack.MsgOptionText(message, false))
	if err != nil {
		slog.Error("Failed to send message to slack", slog.String("message", message), slog.Any("error", err))
	}
}

func (sc *Connector) SendReply(slackMessage *slack.Message, messageBody string) {
	_, _, err := sc.client.PostMessage(sc.channelID, slack.MsgOptionText(messageBody, false), slack.MsgOptionTS(slackMessage.Timestamp))
	if err != nil {
		slog.Error("Failed to send message to slack", slog.String("message", messageBody), slog.Any("error", err))
	}
}

func (sc *Connector) AddReactionToMessage(reaction string, message *slack.Message) {
	err := sc.client.AddReaction(reaction, slack.ItemRef{Channel: sc.channelID, Timestamp: message.Timestamp})
	if err != nil {
		slog.Error("Failed to add reaction to message", slog.Any("error", err))
	}
}

func (sc *Connector) RemoveReactionFromMessage(reaction string, message *slack.Message) {
	err := sc.client.RemoveReaction(reaction, slack.ItemRef{Channel: sc.channelID, Timestamp: message.Timestamp})
	if err != nil {
		slog.Error("Failed to remove reaction from message", slog.Any("error", err))
	}
}

func (sc *Connector) GetMessage(messageKey string) (*slack.Message, error) {
	return sc.findNotification(Notification{Key: messageKey})
}

func (sc *Connector) GetUserIDByEmail(email string) (string, error) {
	user, err := sc.client.GetUserByEmail(email)
	if err != nil {
		return "", err
	}

	return user.ID, nil
}
