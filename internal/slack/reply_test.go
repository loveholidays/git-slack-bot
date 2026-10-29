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

package slack_test

import (
	"encoding/json"
	"errors"
	"git-slack-bot/internal/config"
	"git-slack-bot/internal/slack"
	mock_slack "git-slack-bot/internal/slack/mocks"
	"net/http"
	"net/http/httptest"
	"testing"

	sl "github.com/slack-go/slack"
	"go.uber.org/mock/gomock"
)

func TestCommentReplyLifecycleAcrossRestarts(t *testing.T) {
	const commentURL = "https://github.com/loveholidays/example/pull/42#issuecomment-100"
	parent := &sl.Message{Msg: sl.Msg{Timestamp: "123.000", BotID: "review-bot"}}
	original := "Reviewer left a <" + commentURL + "|comment>:\nOriginal body"
	updated := "Reviewer left a <" + commentURL + "|comment>:\nUpdated body"
	var reply *sl.Message
	var posts, edits, deletes int
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		response := map[string]any{"ok": true, "channel": "reviews", "ts": "123.001"}
		switch request.URL.Path {
		case "/chat.postMessage":
			checkSlackRequest(t, request, "/chat.postMessage", map[string]string{"channel": "reviews", "thread_ts": parent.Timestamp, "text": original})
			reply = &sl.Message{Msg: sl.Msg{Timestamp: "123.001", BotID: parent.BotID, Text: original}}
			posts++
		case "/conversations.replies":
			checkSlackRequest(t, request, "/conversations.replies", map[string]string{"channel": "reviews", "ts": parent.Timestamp})
			messages := []sl.Message{*parent}
			if reply != nil {
				messages = append(messages, *reply)
			}
			response["messages"] = messages
		case "/chat.update":
			checkSlackRequest(t, request, "/chat.update", map[string]string{"channel": "reviews", "ts": "123.001", "text": updated})
			reply.Text = updated
			edits++
		case "/chat.delete":
			checkSlackRequest(t, request, "/chat.delete", map[string]string{"channel": "reviews", "ts": "123.001"})
			reply = nil
			deletes++
		default:
			t.Errorf("unexpected Slack request: %s", request.URL.Path)
		}
		if err := json.NewEncoder(writer).Encode(response); err != nil {
			t.Error(err)
		}
	}))
	defer server.Close()
	newConnector := func() *slack.Connector {
		return slack.NewSlackConnector(config.SlackConfiguration{ChannelID: "reviews"}, sl.New("test-token", sl.OptionAPIURL(server.URL+"/")))
	}
	newConnector().SendReply(parent, original)
	newConnector().UpdateReply(parent, commentURL, updated)
	newConnector().DeleteReply(parent, commentURL)
	newConnector().DeleteReply(parent, commentURL)
	newConnector().UpdateReply(parent, commentURL, updated)
	if posts != 1 || edits != 1 || deletes != 1 {
		t.Fatalf("expected one post, edit and delete; got %d, %d, %d", posts, edits, deletes)
	}
}

func TestCommentReplyLookupPaginatesAndMatchesBotAndExactURL(t *testing.T) {
	for _, action := range []string{"edited", "deleted"} {
		t.Run(action, func(t *testing.T) {
			client := mock_slack.NewMockClient(gomock.NewController(t))
			connector := slack.NewSlackConnector(config.SlackConfiguration{ChannelID: "reviews"}, client)
			parent := &sl.Message{Msg: sl.Msg{Timestamp: "123.000", User: "review-bot"}}
			const url = "https://github.com/loveholidays/example/pull/42#discussion_r100"
			link := "<" + url + "|comment>"
			client.EXPECT().GetConversationReplies(gomock.Any()).DoAndReturn(func(params *sl.GetConversationRepliesParameters) ([]sl.Message, bool, string, error) {
				if params.ChannelID != "reviews" || params.Timestamp != parent.Timestamp || params.Cursor != "" {
					t.Errorf("unexpected initial lookup: %+v", params)
				}
				return []sl.Message{
					{Msg: sl.Msg{Timestamp: "123.001", User: "human", Text: link}},
					{Msg: sl.Msg{Timestamp: "123.002", User: parent.User, Text: "Other comment\n" + link}},
					{Msg: sl.Msg{Timestamp: "123.003", User: parent.User, Text: "<" + url + "0|comment>"}},
					{Msg: sl.Msg{Timestamp: parent.Timestamp, User: parent.User, Text: link}},
				}, true, "next-page", nil
			})
			client.EXPECT().GetConversationReplies(gomock.Any()).DoAndReturn(func(params *sl.GetConversationRepliesParameters) ([]sl.Message, bool, string, error) {
				if params.Cursor != "next-page" {
					t.Errorf("unexpected next cursor: %s", params.Cursor)
				}
				return []sl.Message{{Msg: sl.Msg{Timestamp: "123.004", User: parent.User, Text: "Reviewer left a " + link + ":\nBody"}}}, false, "", nil
			})
			if action == "edited" {
				client.EXPECT().UpdateMessage("reviews", "123.004", gomock.Any()).Return("reviews", "123.004", "Updated", nil)
				connector.UpdateReply(parent, url, "Updated")
			} else {
				client.EXPECT().DeleteMessage("reviews", "123.004").Return("reviews", "123.004", nil)
				connector.DeleteReply(parent, url)
			}
		})
	}
}

func TestCommentReplyLookupFailureDoesNotPostOrMutate(t *testing.T) {
	client := mock_slack.NewMockClient(gomock.NewController(t))
	connector := slack.NewSlackConnector(config.SlackConfiguration{ChannelID: "reviews"}, client)
	parent := &sl.Message{Msg: sl.Msg{Timestamp: "123.000", BotID: "review-bot"}}
	client.EXPECT().GetConversationReplies(gomock.Any()).Return(nil, false, "", errors.New("Slack unavailable")).Times(2)
	connector.UpdateReply(parent, "https://github.com/comment", "Updated")
	connector.DeleteReply(parent, "https://github.com/comment")
	connector.UpdateReply(parent, "", "Updated")
	connector.DeleteReply(parent, "")
}
