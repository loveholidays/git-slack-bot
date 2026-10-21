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

package handler_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"git-slack-bot/internal/handler"
	bot_slack "git-slack-bot/internal/slack"
	mock_slack "git-slack-bot/internal/slack/mocks"
	"git-slack-bot/internal/user"
	mock_user "git-slack-bot/internal/user/mocks"
	"strings"
	"testing"
	"time"

	gh "github.com/google/go-github/v56/github"
	"github.com/slack-go/slack"
	"go.uber.org/mock/gomock"
)

func TestAssigneeNotifications(t *testing.T) {
	for _, testCase := range []struct {
		name             string
		repository       string
		author           string
		assignees        []string
		action           string
		assignedUser     string
		draft            bool
		state            string
		wantNotification bool
	}{
		{name: "existing team author", repository: "loveholidays/backend", author: "member", action: "opened", wantNotification: true},
		{name: "Flux team assignee at creation", repository: "loveholidays/flux", author: "external", assignees: []string{"member"}, action: "opened", wantNotification: true},
		{name: "Flux team assignee added later", repository: "loveholidays/flux", author: "external", assignees: []string{"member"}, action: "assigned", assignedUser: "member", wantNotification: true},
		{name: "Flux team assignee ready for review", repository: "loveholidays/flux", author: "external", assignees: []string{"member"}, action: "ready_for_review", wantNotification: true},
		{name: "multiple Flux assignees", repository: "loveholidays/flux", author: "external", assignees: []string{"external", "member", "another-member"}, action: "opened", wantNotification: true},
		{name: "team author and assignee", repository: "loveholidays/flux", author: "member", assignees: []string{"member"}, action: "opened", wantNotification: true},
		{name: "non Flux assignment", repository: "loveholidays/backend", author: "external", assignees: []string{"member"}, action: "opened"},
		{name: "other owner Flux repository", repository: "another-org/flux", author: "external", assignees: []string{"member"}, action: "opened"},
		{name: "Flux non team assignee", repository: "loveholidays/flux", author: "external", assignees: []string{"external"}, action: "opened"},
		{name: "Flux no assignee", repository: "loveholidays/flux", author: "external", action: "opened"},
		{name: "assignment of non team member", repository: "loveholidays/flux", author: "external", assignees: []string{"member", "external"}, action: "assigned", assignedUser: "external"},
		{name: "team author assignment outside Flux", repository: "loveholidays/backend", author: "member", assignees: []string{"member"}, action: "assigned", assignedUser: "member"},
		{name: "draft at creation", repository: "loveholidays/flux", author: "external", assignees: []string{"member"}, action: "opened", draft: true},
		{name: "closed PR assigned later", repository: "loveholidays/flux", author: "external", assignees: []string{"member"}, action: "assigned", assignedUser: "member", state: "closed"},
		{name: "draft assigned later", repository: "loveholidays/flux", author: "external", assignees: []string{"member"}, action: "assigned", assignedUser: "member", draft: true},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			slackMock := mock_slack.NewMockInteractor(gomock.NewController(t))
			userService := user.NewService(slackMock, []string{"member", "another-member"}, nil, nil, nil)
			gitHandler := handler.NewGitHandler(slackMock, userService, validEmojis(), nil)
			pullRequest := assignmentPullRequest(testCase.author, testCase.assignees)
			pullRequest.Draft = gh.Bool(testCase.draft)
			pullRequest.State = gh.String(testCase.state)
			event := gh.PullRequestEvent{
				Action:      gh.String(testCase.action),
				Repo:        assignmentRepository(testCase.repository),
				PullRequest: pullRequest,
				Assignee:    &gh.User{Login: gh.String(testCase.assignedUser)},
			}
			if testCase.wantNotification {
				slackMock.EXPECT().SendMessageOnce(bot_slack.Notification{Key: "<https://github.com/loveholidays/flux/pull/42>", Text: fmt.Sprintf("%s Update deployment:\nhttps://github.com/loveholidays/flux/pull/42", testCase.author), CreatedAt: pullRequest.GetCreatedAt().Time})
			}
			gitHandler.HandlePullRequestEvent(assignmentJSON(t, event))
		})
	}
}

func TestFluxAssigneeReactionsAndThreads(t *testing.T) {
	for _, testCase := range []struct {
		name string
		run  func(*testing.T, *handler.GitHandler, *mock_slack.MockInteractor, *slack.Message, *gh.PullRequest)
	}{
		{name: "merged", run: func(t *testing.T, gitHandler *handler.GitHandler, slackMock *mock_slack.MockInteractor, message *slack.Message, pullRequest *gh.PullRequest) {
			t.Helper()
			pullRequest.MergedAt = &gh.Timestamp{Time: time.Now()}
			slackMock.EXPECT().AddReactionToMessage("merged", message)
			gitHandler.HandlePullRequestEvent(assignmentJSON(t, gh.PullRequestEvent{Action: gh.String("closed"), Repo: assignmentRepository("loveholidays/flux"), PullRequest: pullRequest}))
		}},
		{name: "closed", run: func(t *testing.T, gitHandler *handler.GitHandler, slackMock *mock_slack.MockInteractor, message *slack.Message, pullRequest *gh.PullRequest) {
			t.Helper()
			slackMock.EXPECT().AddReactionToMessage("x", message)
			gitHandler.HandlePullRequestEvent(assignmentJSON(t, gh.PullRequestEvent{Action: gh.String("closed"), Repo: assignmentRepository("loveholidays/flux"), PullRequest: pullRequest}))
		}},
		{name: "reopened", run: func(t *testing.T, gitHandler *handler.GitHandler, slackMock *mock_slack.MockInteractor, message *slack.Message, pullRequest *gh.PullRequest) {
			t.Helper()
			slackMock.EXPECT().RemoveReactionFromMessage("x", message)
			gitHandler.HandlePullRequestEvent(assignmentJSON(t, gh.PullRequestEvent{Action: gh.String("reopened"), Repo: assignmentRepository("loveholidays/flux"), PullRequest: pullRequest}))
		}},
		{name: "approved with review body", run: func(t *testing.T, gitHandler *handler.GitHandler, slackMock *mock_slack.MockInteractor, message *slack.Message, pullRequest *gh.PullRequest) {
			t.Helper()
			slackMock.EXPECT().AddReactionToMessage("+1", message)
			slackMock.EXPECT().SendReply(message, "reviewer left a <https://github.com/review|review>:\nLooks good")
			gitHandler.HandlePullRequestReviewEvent(assignmentJSON(t, gh.PullRequestReviewEvent{Action: gh.String("submitted"), Repo: assignmentRepository("loveholidays/flux"), PullRequest: pullRequest, Review: &gh.PullRequestReview{State: gh.String("approved"), Body: gh.String("Looks good"), HTMLURL: gh.String("https://github.com/review"), User: &gh.User{Login: gh.String("reviewer")}}}))
		}},
		{name: "approval dismissed", run: func(t *testing.T, gitHandler *handler.GitHandler, slackMock *mock_slack.MockInteractor, message *slack.Message, pullRequest *gh.PullRequest) {
			t.Helper()
			slackMock.EXPECT().RemoveReactionFromMessage("+1", message)
			gitHandler.HandlePullRequestReviewEvent(assignmentJSON(t, gh.PullRequestReviewEvent{Action: gh.String("dismissed"), Repo: assignmentRepository("loveholidays/flux"), PullRequest: pullRequest, Review: &gh.PullRequestReview{User: &gh.User{Login: gh.String("reviewer")}}}))
		}},
		{name: "review comment", run: func(t *testing.T, gitHandler *handler.GitHandler, slackMock *mock_slack.MockInteractor, message *slack.Message, pullRequest *gh.PullRequest) {
			t.Helper()
			slackMock.EXPECT().SendReply(message, gomock.Any())
			gitHandler.HandlePullRequestReviewCommentEvent(assignmentJSON(t, gh.PullRequestReviewCommentEvent{Action: gh.String("created"), Repo: assignmentRepository("loveholidays/flux"), PullRequest: pullRequest, Comment: &gh.PullRequestComment{User: &gh.User{Login: gh.String("reviewer")}}}))
		}},
		{name: "top level comment", run: func(t *testing.T, gitHandler *handler.GitHandler, slackMock *mock_slack.MockInteractor, message *slack.Message, pullRequest *gh.PullRequest) {
			t.Helper()
			slackMock.EXPECT().SendReply(message, gomock.Any())
			gitHandler.HandleIssueCommentEvent(assignmentJSON(t, gh.IssueCommentEvent{Action: gh.String("created"), Repo: assignmentRepository("loveholidays/flux"), Issue: &gh.Issue{User: pullRequest.User, Assignees: pullRequest.Assignees, HTMLURL: pullRequest.HTMLURL}, Comment: &gh.IssueComment{User: &gh.User{Login: gh.String("reviewer")}}}))
		}},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			slackMock := mock_slack.NewMockInteractor(gomock.NewController(t))
			userService := user.NewService(slackMock, []string{"member"}, nil, nil, nil)
			gitHandler := handler.NewGitHandler(slackMock, userService, validEmojis(), nil)
			message := &slack.Message{Msg: slack.Msg{Timestamp: "123.456"}}
			slackMock.EXPECT().GetMessage("<https://github.com/loveholidays/flux/pull/42>").Return(message, nil)
			testCase.run(t, gitHandler, slackMock, message, assignmentPullRequest("external", []string{"member"}))
		})
	}
}

func assignmentPullRequest(author string, assignees []string) *gh.PullRequest {
	pullRequest := &gh.PullRequest{
		User:      &gh.User{Login: gh.String(author)},
		Title:     gh.String("Update deployment"),
		HTMLURL:   gh.String("https://github.com/loveholidays/flux/pull/42"),
		CreatedAt: &gh.Timestamp{Time: time.Date(2026, time.October, 21, 12, 0, 0, 0, time.UTC)},
	}
	for _, assignee := range assignees {
		pullRequest.Assignees = append(pullRequest.Assignees, &gh.User{Login: gh.String(assignee)})
	}
	return pullRequest
}

func assignmentRepository(fullName string) *gh.Repository {
	return &gh.Repository{FullName: gh.String(fullName), Name: gh.String(strings.Split(fullName, "/")[1])}
}

func assignmentJSON(t *testing.T, event any) []byte {
	t.Helper()
	body, err := json.Marshal(event)
	if err != nil {
		t.Fatal(err)
	}
	return body
}

func TestGitEventsWithoutSlackThread(t *testing.T) {
	for _, testCase := range []struct {
		name string
		run  func(*handler.GitHandler)
	}{
		{name: "merged", run: func(gitHandler *handler.GitHandler) { gitHandler.HandlePullRequestEvent(prMergedJSONData) }},
		{name: "reopened", run: func(gitHandler *handler.GitHandler) { gitHandler.HandlePullRequestEvent(prReopenedJSONData) }},
		{name: "approved", run: func(gitHandler *handler.GitHandler) { gitHandler.HandlePullRequestReviewEvent(prApprovedJSONData) }},
		{name: "dismissed", run: func(gitHandler *handler.GitHandler) {
			gitHandler.HandlePullRequestReviewEvent(prReviewDismissedJSONData)
		}},
		{name: "review comment", run: func(gitHandler *handler.GitHandler) {
			gitHandler.HandlePullRequestReviewCommentEvent(prCommentJSONData)
		}},
		{name: "issue comment", run: func(gitHandler *handler.GitHandler) { gitHandler.HandleIssueCommentEvent(prIssueCommentJSONData) }},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			slackMock := mock_slack.NewMockInteractor(controller)
			userMock := mock_user.NewMockService(controller)
			userMock.EXPECT().IsTeamMember(gomock.Any()).Return(true)
			userMock.EXPECT().IsIgnoredReviewUser(gomock.Any()).Return(false).AnyTimes()
			userMock.EXPECT().IsIgnoredCommentUser(gomock.Any()).Return(false).AnyTimes()
			userMock.EXPECT().GetUserDescriptor(gomock.Any()).Return("reviewer").AnyTimes()
			slackMock.EXPECT().GetMessage(gomock.Any()).Return(nil, errors.New("thread unavailable"))
			testCase.run(handler.NewGitHandler(slackMock, userMock, validEmojis(), nil))
		})
	}
}

func TestMalformedGitEvents(t *testing.T) {
	slackMock := mock_slack.NewMockInteractor(gomock.NewController(t))
	userService := user.NewService(slackMock, []string{"member"}, nil, nil, nil)
	gitHandler := handler.NewGitHandler(slackMock, userService, validEmojis(), nil)
	for _, handle := range []func([]byte){gitHandler.HandlePullRequestEvent, gitHandler.HandlePullRequestReviewEvent, gitHandler.HandlePullRequestReviewCommentEvent, gitHandler.HandleIssueCommentEvent} {
		handle([]byte("{"))
	}
}
