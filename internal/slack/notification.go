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

import (
	"errors"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/slack-go/slack"
)

type Notification struct {
	Key       string
	Text      string
	CreatedAt time.Time
}

var errMessageNotFound = errors.New("could not find message")

func (sc *Connector) SendMessageOnce(notification Notification) {
	sc.notificationMutex.Lock()
	defer sc.notificationMutex.Unlock()

	_, err := sc.findNotification(notification)
	if err == nil {
		return
	}
	if !errors.Is(err, errMessageNotFound) {
		slog.Error("Could not check for an existing notification", slog.Any("error", err))
		return
	}
	sc.SendMessage(notification.Text)
}

func (sc *Connector) findNotification(notification Notification) (*slack.Message, error) {
	parameters := &slack.GetConversationHistoryParameters{ChannelID: sc.channelID}
	if !notification.CreatedAt.IsZero() {
		parameters.Oldest = strconv.FormatInt(notification.CreatedAt.Unix(), 10)
	}
	for {
		history, err := sc.client.GetConversationHistory(parameters)
		if err != nil {
			return nil, err
		}
		if message := notification.findInHistory(history.Messages); message != nil {
			return message, nil
		}
		if history.ResponseMetaData.NextCursor == "" {
			return nil, errMessageNotFound
		}
		parameters.Cursor = history.ResponseMetaData.NextCursor
	}
}

func (notification Notification) findInHistory(messages []slack.Message) *slack.Message {
	for _, message := range messages {
		if strings.Contains(message.Text, notification.Key) {
			return &message
		}
	}
	return nil
}
