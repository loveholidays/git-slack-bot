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

package slack_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"git-slack-bot/internal/config"
	"git-slack-bot/internal/slack"
	mock_slack "git-slack-bot/internal/slack/mocks"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	sl "github.com/slack-go/slack"
	"go.uber.org/mock/gomock"
)

func TestConcurrentNotificationsPostOnce(t *testing.T) {
	client := mock_slack.NewMockClient(gomock.NewController(t))
	connector := slack.NewSlackConnector(config.SlackConfiguration{ChannelID: "reviews"}, client)
	createdAt := time.Unix(1792584000, 0)
	messageKey := "<https://github.com/loveholidays/flux/pull/42>"
	var postedMessages []sl.Message
	client.EXPECT().GetConversationHistory(gomock.Any()).DoAndReturn(func(parameters *sl.GetConversationHistoryParameters) (*sl.GetConversationHistoryResponse, error) {
		if parameters.ChannelID != "reviews" || parameters.Oldest != "1792584000" {
			t.Errorf("unexpected history parameters: %+v", parameters)
		}
		return &sl.GetConversationHistoryResponse{Messages: postedMessages}, nil
	}).Times(10)
	client.EXPECT().PostMessage("reviews", gomock.Any()).DoAndReturn(func(_ string, _ ...sl.MsgOption) (string, string, error) {
		postedMessages = []sl.Message{{Msg: sl.Msg{Text: "Deployment " + messageKey}}}
		return "reviews", "123.456", nil
	}).Times(1)

	var notifications sync.WaitGroup
	for range 10 {
		notifications.Go(func() {
			connector.SendMessageOnce(slack.Notification{Key: messageKey, Text: "Deployment " + messageKey, CreatedAt: createdAt})
		})
	}
	notifications.Wait()
}

func TestNotificationFindsExistingPostOnLaterPage(t *testing.T) {
	client := mock_slack.NewMockClient(gomock.NewController(t))
	connector := slack.NewSlackConnector(config.SlackConfiguration{ChannelID: "reviews"}, client)
	messageKey := "<https://github.com/loveholidays/flux/pull/42>"
	gomock.InOrder(
		client.EXPECT().GetConversationHistory(gomock.Any()).DoAndReturn(func(parameters *sl.GetConversationHistoryParameters) (*sl.GetConversationHistoryResponse, error) {
			if parameters.Cursor != "" {
				t.Errorf("unexpected first cursor: %s", parameters.Cursor)
			}
			history := &sl.GetConversationHistoryResponse{}
			history.ResponseMetaData.NextCursor = "next-page"
			return history, nil
		}),
		client.EXPECT().GetConversationHistory(gomock.Any()).DoAndReturn(func(parameters *sl.GetConversationHistoryParameters) (*sl.GetConversationHistoryResponse, error) {
			if parameters.Cursor != "next-page" {
				t.Errorf("unexpected next cursor: %s", parameters.Cursor)
			}
			return &sl.GetConversationHistoryResponse{Messages: []sl.Message{{Msg: sl.Msg{Text: messageKey}}}}, nil
		}),
	)
	connector.SendMessageOnce(slack.Notification{Key: messageKey, Text: messageKey, CreatedAt: time.Now()})
}

func TestNotificationDoesNotPostWhenHistoryFails(t *testing.T) {
	client := mock_slack.NewMockClient(gomock.NewController(t))
	connector := slack.NewSlackConnector(config.SlackConfiguration{ChannelID: "reviews"}, client)
	client.EXPECT().GetConversationHistory(gomock.Any()).Return(nil, errors.New("Slack unavailable"))
	connector.SendMessageOnce(slack.Notification{Key: "PR", Text: "PR", CreatedAt: time.Now()})
}

func TestNotificationRetriesFailedPost(t *testing.T) {
	client := mock_slack.NewMockClient(gomock.NewController(t))
	connector := slack.NewSlackConnector(config.SlackConfiguration{ChannelID: "reviews"}, client)
	client.EXPECT().GetConversationHistory(gomock.Any()).Return(&sl.GetConversationHistoryResponse{}, nil).Times(2)
	gomock.InOrder(
		client.EXPECT().PostMessage("reviews", gomock.Any()).Return("", "", errors.New("Slack unavailable")),
		client.EXPECT().PostMessage("reviews", gomock.Any()).Return("reviews", "123.456", nil),
	)
	connector.SendMessageOnce(slack.Notification{Key: "PR", Text: "PR"})
	connector.SendMessageOnce(slack.Notification{Key: "PR", Text: "PR"})
}

func TestNotificationAfterRestartDoesNotPostAgain(t *testing.T) {
	client := mock_slack.NewMockClient(gomock.NewController(t))
	client.EXPECT().GetConversationHistory(gomock.Any()).Return(&sl.GetConversationHistoryResponse{Messages: []sl.Message{{Msg: sl.Msg{Text: "<PR>"}}}}, nil).Times(2)
	for range 2 {
		connector := slack.NewSlackConnector(config.SlackConfiguration{ChannelID: "reviews"}, client)
		connector.SendMessageOnce(slack.Notification{Key: "<PR>", Text: "<PR>", CreatedAt: time.Now()})
	}
}

func TestSlackThreadAndReactionRequests(t *testing.T) {
	for _, testCase := range []struct {
		name   string
		path   string
		fields map[string]string
		run    func(*slack.Connector, *sl.Message)
	}{
		{name: "reply", path: "/chat.postMessage", fields: map[string]string{"channel": "reviews", "thread_ts": "123.456", "text": "Review comment"}, run: func(connector *slack.Connector, message *sl.Message) { connector.SendReply(message, "Review comment") }},
		{name: "add reaction", path: "/reactions.add", fields: map[string]string{"channel": "reviews", "timestamp": "123.456", "name": "+1"}, run: func(connector *slack.Connector, message *sl.Message) { connector.AddReactionToMessage("+1", message) }},
		{name: "remove reaction", path: "/reactions.remove", fields: map[string]string{"channel": "reviews", "timestamp": "123.456", "name": "+1"}, run: func(connector *slack.Connector, message *sl.Message) {
			connector.RemoveReactionFromMessage("+1", message)
		}},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			for _, success := range []bool{true, false} {
				server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
					checkSlackRequest(t, request, testCase.path, testCase.fields)
					if err := json.NewEncoder(writer).Encode(map[string]any{"ok": success, "error": "test_error"}); err != nil {
						t.Error(err)
					}
				}))
				connector := slack.NewSlackConnector(config.SlackConfiguration{ChannelID: "reviews"}, sl.New("test-token", sl.OptionAPIURL(server.URL+"/")))
				testCase.run(connector, &sl.Message{Msg: sl.Msg{Timestamp: "123.456"}})
				server.Close()
			}
		})
	}
}

func TestSlackUserLookup(t *testing.T) {
	client := mock_slack.NewMockClient(gomock.NewController(t))
	connector := slack.NewSlackConnector(config.SlackConfiguration{ChannelID: "reviews"}, client)
	client.EXPECT().GetUserByEmail("member@example.com").Return(&sl.User{ID: "member-id"}, nil)
	userID, err := connector.GetUserIDByEmail("member@example.com")
	if err != nil || userID != "member-id" {
		t.Fatalf("unexpected user lookup: %q, %v", userID, err)
	}
	lookupError := errors.New("user not found")
	client.EXPECT().GetUserByEmail("missing@example.com").Return(nil, lookupError)
	if _, err := connector.GetUserIDByEmail("missing@example.com"); !errors.Is(err, lookupError) {
		t.Fatalf("expected lookup error, got %v", err)
	}
}

func checkSlackRequest(t *testing.T, request *http.Request, path string, fields map[string]string) {
	t.Helper()
	if request.URL.Path != path {
		t.Errorf("unexpected Slack endpoint: %s", request.URL.Path)
	}
	for field, expected := range fields {
		if actual := request.FormValue(field); actual != expected {
			t.Errorf("unexpected %s: got %q, want %q", field, actual, expected)
		}
	}
}

func TestSlackReactionFailuresAreLogged(t *testing.T) {
	client := mock_slack.NewMockClient(gomock.NewController(t))
	connector := slack.NewSlackConnector(config.SlackConfiguration{ChannelID: "reviews"}, client)
	var logOutput bytes.Buffer
	previousLogger := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&logOutput, nil)))
	t.Cleanup(func() { slog.SetDefault(previousLogger) })
	apiError := errors.New("Slack temporarily unavailable")
	item := sl.ItemRef{Channel: "reviews", Timestamp: "123.456"}
	client.EXPECT().AddReaction("+1", item).Return(apiError)
	client.EXPECT().RemoveReaction("+1", item).Return(apiError)
	message := &sl.Message{Msg: sl.Msg{Timestamp: item.Timestamp}}
	connector.AddReactionToMessage("+1", message)
	connector.RemoveReactionFromMessage("+1", message)
	if strings.Count(logOutput.String(), apiError.Error()) != 2 {
		t.Errorf("expected both reaction failures in logs, got %s", logOutput.String())
	}
}
