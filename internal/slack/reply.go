/*
git-slack-bot
Copyright (C) 2026 loveholidays

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

import (
	"errors"
	"log/slog"
	"strings"

	"github.com/slack-go/slack"
)

func (sc *Connector) UpdateReply(parent *slack.Message, commentURL, message string) {
	reply, err := sc.findReply(parent, commentURL)
	if errors.Is(err, errMessageNotFound) {
		return
	}
	if err == nil {
		_, _, _, err = sc.client.UpdateMessage(sc.channelID, reply.Timestamp, slack.MsgOptionText(message, false))
	}
	if err != nil {
		slog.Error("Failed to update Slack comment reply", slog.Any("error", err))
	}
}

func (sc *Connector) DeleteReply(parent *slack.Message, commentURL string) {
	reply, err := sc.findReply(parent, commentURL)
	if errors.Is(err, errMessageNotFound) {
		return
	}
	if err == nil {
		_, _, err = sc.client.DeleteMessage(sc.channelID, reply.Timestamp)
	}
	if err != nil {
		slog.Error("Failed to delete Slack comment reply", slog.Any("error", err))
	}
}

func (sc *Connector) findReply(parent *slack.Message, commentURL string) (*slack.Message, error) {
	if commentURL == "" {
		return nil, errMessageNotFound
	}
	parameters := &slack.GetConversationRepliesParameters{ChannelID: sc.channelID, Timestamp: parent.Timestamp}
	for {
		messages, _, cursor, err := sc.client.GetConversationReplies(parameters)
		if err != nil {
			return nil, err
		}
		if reply := findReplyInMessages(messages, parent, commentURL); reply != nil {
			return reply, nil
		}
		if cursor == "" {
			return nil, errMessageNotFound
		}
		parameters.Cursor = cursor
	}
}

func findReplyInMessages(messages []slack.Message, parent *slack.Message, commentURL string) *slack.Message {
	for _, message := range messages {
		if message.Timestamp == parent.Timestamp {
			continue
		}
		if !hasSameMessageAuthor(parent, &message) {
			continue
		}
		header, _, _ := strings.Cut(message.Text, "\n")
		if strings.Contains(header, "<"+commentURL+"|comment>") {
			return &message
		}
	}
	return nil
}

func hasSameMessageAuthor(parent, message *slack.Message) bool {
	isSameBot := parent.BotID != "" && message.BotID == parent.BotID
	isSameUser := parent.User != "" && message.User == parent.User
	return isSameBot || isSameUser
}
