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
	"git-slack-bot/internal/config"
	"git-slack-bot/internal/handler"
	mock_handler "git-slack-bot/internal/handler/mocks"
	mock_slack "git-slack-bot/internal/slack/mocks"
	"git-slack-bot/internal/user"
	"strings"
	"testing"
	"time"

	gh "github.com/google/go-github/v93/github"
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
		disabled         bool
	}{
		{name: "default author notification", repository: "loveholidays/backend", author: "member", action: "opened", disabled: true, wantNotification: true},
		{name: "default Flux assignee ignored at creation", repository: "loveholidays/flux", author: "external", assignees: []string{"member"}, action: "opened", disabled: true},
		{name: "default Flux assignee ignored later", repository: "loveholidays/flux", author: "external", assignees: []string{"member"}, action: "assigned", assignedUser: "member", disabled: true},
		{name: "default Flux author assignment ignored", repository: "loveholidays/flux", author: "member", assignees: []string{"member"}, action: "assigned", assignedUser: "member", disabled: true},
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
			controller := gomock.NewController(t)
			slackMock := mock_slack.NewMockInteractor(controller)
			notifierMock := mock_handler.NewMockPullRequestNotifier(controller)
			userService := user.NewService(slackMock, []string{"member", "another-member"}, nil, nil, nil)
			gitHandler := handler.NewGitHandler(slackMock, userService, notifierMock, config.Configuration{GitHub: config.GitHubConfiguration{NotifyAssignedFluxPRs: !testCase.disabled}})
			pullRequest := assignmentPullRequest(testCase.author, testCase.assignees)
			pullRequest.Draft = new(testCase.draft)
			pullRequest.State = new(testCase.state)
			event := gh.PullRequestEvent{
				Action:      new(testCase.action),
				Repo:        assignmentRepository(testCase.repository),
				PullRequest: pullRequest,
				Assignee:    &gh.User{Login: new(testCase.assignedUser)},
			}
			if testCase.wantNotification {
				notifierMock.EXPECT().NotifyPullRequestOpened(slackMock, pullRequest)
			}
			gitHandler.HandlePullRequestEvent(assignmentJSON(t, event))
		})
	}
}

func TestFluxAssigneeReactionsAndThreads(t *testing.T) {
	for _, testCase := range []struct {
		name string
		run  func(*testing.T, *handler.GitHandler, *mock_handler.MockPullRequestNotifier, *mock_slack.MockInteractor, *gh.PullRequest)
	}{
		{name: "merged", run: func(t *testing.T, gitHandler *handler.GitHandler, notifierMock *mock_handler.MockPullRequestNotifier, slackMock *mock_slack.MockInteractor, pullRequest *gh.PullRequest) {
			t.Helper()
			pullRequest.MergedAt = pullRequest.CreatedAt
			notifierMock.EXPECT().NotifyPullRequestClosed(slackMock, pullRequest)
			gitHandler.HandlePullRequestEvent(assignmentJSON(t, gh.PullRequestEvent{Action: new("closed"), Repo: assignmentRepository("loveholidays/flux"), PullRequest: pullRequest}))
		}},
		{name: "closed", run: func(t *testing.T, gitHandler *handler.GitHandler, notifierMock *mock_handler.MockPullRequestNotifier, slackMock *mock_slack.MockInteractor, pullRequest *gh.PullRequest) {
			t.Helper()
			notifierMock.EXPECT().NotifyPullRequestClosed(slackMock, pullRequest)
			gitHandler.HandlePullRequestEvent(assignmentJSON(t, gh.PullRequestEvent{Action: new("closed"), Repo: assignmentRepository("loveholidays/flux"), PullRequest: pullRequest}))
		}},
		{name: "reopened", run: func(t *testing.T, gitHandler *handler.GitHandler, notifierMock *mock_handler.MockPullRequestNotifier, slackMock *mock_slack.MockInteractor, pullRequest *gh.PullRequest) {
			t.Helper()
			notifierMock.EXPECT().NotifyPullRequestReopened(slackMock, pullRequest)
			gitHandler.HandlePullRequestEvent(assignmentJSON(t, gh.PullRequestEvent{Action: new("reopened"), Repo: assignmentRepository("loveholidays/flux"), PullRequest: pullRequest}))
		}},
		{name: "approved with review body", run: func(t *testing.T, gitHandler *handler.GitHandler, notifierMock *mock_handler.MockPullRequestNotifier, slackMock *mock_slack.MockInteractor, pullRequest *gh.PullRequest) {
			t.Helper()
			review := &gh.PullRequestReview{State: new("approved"), Body: new("Looks good"), HTMLURL: new("https://github.com/review"), User: &gh.User{Login: new("reviewer")}}
			notifierMock.EXPECT().NotifyReviewSubmitted(slackMock, pullRequest, review)
			gitHandler.HandlePullRequestReviewEvent(assignmentJSON(t, gh.PullRequestReviewEvent{Action: new("submitted"), Repo: assignmentRepository("loveholidays/flux"), PullRequest: pullRequest, Review: review}))
		}},
		{name: "approval dismissed", run: func(t *testing.T, gitHandler *handler.GitHandler, notifierMock *mock_handler.MockPullRequestNotifier, slackMock *mock_slack.MockInteractor, pullRequest *gh.PullRequest) {
			t.Helper()
			notifierMock.EXPECT().NotifyReviewDismissed(slackMock, pullRequest)
			gitHandler.HandlePullRequestReviewEvent(assignmentJSON(t, gh.PullRequestReviewEvent{Action: new("dismissed"), Repo: assignmentRepository("loveholidays/flux"), PullRequest: pullRequest, Review: &gh.PullRequestReview{User: &gh.User{Login: new("reviewer")}}}))
		}},
		{name: "review comment", run: func(t *testing.T, gitHandler *handler.GitHandler, notifierMock *mock_handler.MockPullRequestNotifier, slackMock *mock_slack.MockInteractor, pullRequest *gh.PullRequest) {
			t.Helper()
			notifierMock.EXPECT().NotifyComment(slackMock, pullRequest, gomock.Any())
			gitHandler.HandlePullRequestReviewCommentEvent(assignmentJSON(t, gh.PullRequestReviewCommentEvent{Action: new("created"), Repo: assignmentRepository("loveholidays/flux"), PullRequest: pullRequest, Comment: &gh.PullRequestComment{User: &gh.User{Login: new("reviewer")}}}))
		}},
		{name: "top level comment", run: func(t *testing.T, gitHandler *handler.GitHandler, notifierMock *mock_handler.MockPullRequestNotifier, slackMock *mock_slack.MockInteractor, pullRequest *gh.PullRequest) {
			t.Helper()
			expectedPullRequest := &gh.PullRequest{User: pullRequest.User, Assignees: pullRequest.Assignees, HTMLURL: pullRequest.HTMLURL}
			notifierMock.EXPECT().NotifyComment(slackMock, expectedPullRequest, gomock.Any())
			gitHandler.HandleIssueCommentEvent(assignmentJSON(t, gh.IssueCommentEvent{Action: new("created"), Repo: assignmentRepository("loveholidays/flux"), Issue: &gh.Issue{User: pullRequest.User, Assignees: pullRequest.Assignees, HTMLURL: pullRequest.HTMLURL}, Comment: &gh.IssueComment{User: &gh.User{Login: new("reviewer")}}}))
		}},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			slackMock := mock_slack.NewMockInteractor(controller)
			notifierMock := mock_handler.NewMockPullRequestNotifier(controller)
			userService := user.NewService(slackMock, []string{"member"}, nil, nil, nil)
			gitHandler := handler.NewGitHandler(slackMock, userService, notifierMock, config.Configuration{GitHub: config.GitHubConfiguration{NotifyAssignedFluxPRs: true}})
			testCase.run(t, gitHandler, notifierMock, slackMock, assignmentPullRequest("external", []string{"member"}))
		})
	}
}

func assignmentPullRequest(author string, assignees []string) *gh.PullRequest {
	pullRequest := &gh.PullRequest{
		User:      &gh.User{Login: new(author)},
		Title:     new("Update deployment"),
		HTMLURL:   new("https://github.com/loveholidays/flux/pull/42"),
		CreatedAt: &gh.Timestamp{Time: time.Date(2026, time.October, 21, 12, 0, 0, 0, time.UTC)},
	}
	for _, assignee := range assignees {
		pullRequest.Assignees = append(pullRequest.Assignees, &gh.User{Login: new(assignee)})
	}
	return pullRequest
}

func assignmentRepository(fullName string) *gh.Repository {
	return &gh.Repository{FullName: new(fullName), Name: new(strings.Split(fullName, "/")[1])}
}

func assignmentJSON(t *testing.T, event any) []byte {
	t.Helper()
	body, err := json.Marshal(event)
	if err != nil {
		t.Fatal(err)
	}
	return body
}

func TestMalformedGitEvents(t *testing.T) {
	controller := gomock.NewController(t)
	slackMock := mock_slack.NewMockInteractor(controller)
	notifierMock := mock_handler.NewMockPullRequestNotifier(controller)
	userService := user.NewService(slackMock, []string{"member"}, nil, nil, nil)
	gitHandler := handler.NewGitHandler(slackMock, userService, notifierMock, config.Configuration{GitHub: config.GitHubConfiguration{NotifyAssignedFluxPRs: true}})
	for _, handle := range []func([]byte){gitHandler.HandlePullRequestEvent, gitHandler.HandlePullRequestReviewEvent, gitHandler.HandlePullRequestReviewCommentEvent, gitHandler.HandleIssueCommentEvent} {
		handle([]byte("{"))
	}
}

func TestDisabledFluxAssignmentTracking(t *testing.T) {
	controller := gomock.NewController(t)
	slackMock := mock_slack.NewMockInteractor(controller)
	notifierMock := mock_handler.NewMockPullRequestNotifier(controller)
	userService := user.NewService(slackMock, []string{"member"}, nil, nil, nil)
	gitHandler := handler.NewGitHandler(slackMock, userService, notifierMock, config.Configuration{GitHub: config.GitHubConfiguration{}})
	pullRequest := assignmentPullRequest("external", []string{"member"})
	repository := assignmentRepository("loveholidays/flux")
	gitHandler.HandlePullRequestEvent(assignmentJSON(t, gh.PullRequestEvent{Action: new("closed"), Repo: repository, PullRequest: pullRequest}))
	gitHandler.HandlePullRequestReviewEvent(assignmentJSON(t, gh.PullRequestReviewEvent{Action: new("submitted"), Repo: repository, PullRequest: pullRequest, Review: &gh.PullRequestReview{State: new("approved"), User: &gh.User{Login: new("reviewer")}}}))
	gitHandler.HandlePullRequestReviewCommentEvent(assignmentJSON(t, gh.PullRequestReviewCommentEvent{Action: new("created"), Repo: repository, PullRequest: pullRequest, Comment: &gh.PullRequestComment{User: &gh.User{Login: new("reviewer")}}}))
	gitHandler.HandleIssueCommentEvent(assignmentJSON(t, gh.IssueCommentEvent{Action: new("created"), Repo: repository, Issue: &gh.Issue{User: pullRequest.User, Assignees: pullRequest.Assignees, HTMLURL: pullRequest.HTMLURL}, Comment: &gh.IssueComment{User: &gh.User{Login: new("reviewer")}}}))
}

func authorChannelRule() config.NotificationRule {
	return config.NotificationRule{Relationship: "author", Repositories: []string{"*"}, Destination: "channel", ChannelID: "reviews"}
}

func assigneeDMRule() config.NotificationRule {
	return config.NotificationRule{Relationship: "assignee", Repositories: []string{"example/other", "example/deployments"}, Destination: "directMessage"}
}

func TestNotificationRules(t *testing.T) {
	excluded := authorChannelRule()
	excluded.ExcludedRepositories = []string{"example/deployments"}
	authorDM := authorChannelRule()
	authorDM.Destination, authorDM.ChannelID = "directMessage", ""
	assigneeChannel := assigneeDMRule()
	assigneeChannel.Destination, assigneeChannel.ChannelID = "channel", "reviews"
	otherOwner := assigneeDMRule()
	otherOwner.Repositories = []string{"another-org/deployments"}
	for _, testCase := range []struct {
		name         string
		rules        []config.NotificationRule
		author       string
		assignees    []string
		destinations []string
	}{
		{name: "author and assignee both match", rules: []config.NotificationRule{authorChannelRule(), assigneeDMRule()}, author: "member", assignees: []string{"member"}, destinations: []string{"reviews", "dm-member"}},
		{name: "author DM", rules: []config.NotificationRule{authorDM}, author: "member", destinations: []string{"dm-member"}},
		{name: "assignee channel", rules: []config.NotificationRule{assigneeChannel}, author: "external", assignees: []string{"member", "second"}, destinations: []string{"reviews"}},
		{name: "overlapping channels", rules: []config.NotificationRule{authorChannelRule(), assigneeChannel}, author: "member", assignees: []string{"member"}, destinations: []string{"reviews"}},
		{name: "overlapping DMs", rules: []config.NotificationRule{authorDM, assigneeDMRule()}, author: "member", assignees: []string{"member"}, destinations: []string{"dm-member"}},
		{name: "multiple assignees", rules: []config.NotificationRule{assigneeDMRule()}, author: "external", assignees: []string{"member", "second", "member", "external"}, destinations: []string{"dm-member", "dm-second"}},
		{name: "exclusion only affects its rule", rules: []config.NotificationRule{excluded, assigneeDMRule()}, author: "member", assignees: []string{"member"}, destinations: []string{"dm-member"}},
		{name: "repository owner must match", rules: []config.NotificationRule{otherOwner}, author: "external", assignees: []string{"member"}},
		{name: "non member", rules: []config.NotificationRule{authorChannelRule(), assigneeDMRule()}, author: "external", assignees: []string{"external"}},
		{name: "explicit empty disables legacy", rules: []config.NotificationRule{}, author: "member", assignees: []string{"member"}},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			gitHandler, notifierMock, destinations := ruleTestHandler(t, testCase.rules)
			pullRequest := assignmentPullRequest(testCase.author, testCase.assignees)
			for _, id := range testCase.destinations {
				notifierMock.EXPECT().NotifyPullRequestOpened(destinations[id], pullRequest)
			}
			gitHandler.HandlePullRequestEvent(assignmentJSON(t, gh.PullRequestEvent{Action: new("opened"), Repo: assignmentRepository("example/deployments"), PullRequest: pullRequest}))
		})
	}
}

func ruleTestHandler(t *testing.T, rules []config.NotificationRule) (gitHandler *handler.GitHandler, notifierMock *mock_handler.MockPullRequestNotifier, destinations map[string]*mock_slack.MockInteractor) {
	t.Helper()
	controller := gomock.NewController(t)
	root := mock_slack.NewMockInteractor(controller)
	notifierMock = mock_handler.NewMockPullRequestNotifier(controller)
	destinations = map[string]*mock_slack.MockInteractor{}
	for _, id := range []string{"reviews", "dm-member", "dm-second"} {
		destination := mock_slack.NewMockInteractor(controller)
		destination.EXPECT().ConversationID().Return(id).AnyTimes()
		destinations[id] = destination
	}
	root.EXPECT().Channel("reviews").Return(destinations["reviews"]).AnyTimes()
	for _, login := range []string{"member", "second"} {
		root.EXPECT().GetUserIDByEmail(login+"@example.com").Return(login, nil).AnyTimes()
		root.EXPECT().DirectMessage(login).Return(destinations["dm-"+login], nil).AnyTimes()
	}
	users := user.NewService(root, []string{"member", "second"}, []config.GithubEmailToSlackEmail{{GithubEmail: "member", SlackEmail: "member@example.com"}, {GithubEmail: "second", SlackEmail: "second@example.com"}}, nil, nil)
	configuration := config.Configuration{Notifications: rules, GitHub: config.GitHubConfiguration{NotifyAssignedFluxPRs: true}}
	return handler.NewGitHandler(root, users, notifierMock, configuration), notifierMock, destinations
}

func TestRuleThreadsFollowBothDestinations(t *testing.T) {
	gitHandler, notifierMock, destinations := ruleTestHandler(t, []config.NotificationRule{authorChannelRule(), assigneeDMRule(), assigneeDMRule()})
	repository := assignmentRepository("example/deployments")
	pr := assignmentPullRequest("member", []string{"member"})
	review := &gh.PullRequestReview{User: &gh.User{Login: new("external")}, State: new("approved"), Body: new("Looks good"), HTMLURL: new("https://github.com/review")}
	expectedCommentPR := &gh.PullRequest{User: pr.User, Assignees: pr.Assignees, HTMLURL: pr.HTMLURL}
	for _, id := range []string{"reviews", "dm-member"} {
		destination := destinations[id]
		notifierMock.EXPECT().NotifyReviewSubmitted(destination, pr, review)
		notifierMock.EXPECT().NotifyComment(destination, expectedCommentPR, gomock.Any())
		notifierMock.EXPECT().NotifyPullRequestClosed(destination, pr)
	}
	gitHandler.HandlePullRequestReviewEvent(assignmentJSON(t, gh.PullRequestReviewEvent{Action: new("submitted"), Repo: repository, PullRequest: pr, Review: review}))
	gitHandler.HandleIssueCommentEvent(assignmentJSON(t, gh.IssueCommentEvent{Action: new("created"), Repo: repository, Issue: &gh.Issue{User: pr.User, Assignees: pr.Assignees, HTMLURL: pr.HTMLURL}, Comment: &gh.IssueComment{User: &gh.User{Login: new("external")}}}))
	gitHandler.HandlePullRequestEvent(assignmentJSON(t, gh.PullRequestEvent{Action: new("closed"), Repo: repository, PullRequest: pr}))
}

func TestConfiguredRepositoryDirectMessageFailuresDoNotPostToChannel(t *testing.T) {
	for _, failure := range []string{"mapping", "lookup", "open"} {
		t.Run(failure, func(t *testing.T) {
			controller := gomock.NewController(t)
			slackMock := mock_slack.NewMockInteractor(controller)
			notifierMock := mock_handler.NewMockPullRequestNotifier(controller)
			var mappings []config.GithubEmailToSlackEmail
			if failure != "mapping" {
				mappings = []config.GithubEmailToSlackEmail{{GithubEmail: "member", SlackEmail: "member@example.com"}}
				if failure == "lookup" {
					slackMock.EXPECT().GetUserIDByEmail("member@example.com").Return("", errors.New("user not found"))
				} else {
					slackMock.EXPECT().GetUserIDByEmail("member@example.com").Return("member-id", nil)
					slackMock.EXPECT().DirectMessage("member-id").Return(nil, errors.New("missing_scope"))
				}
			}
			users := user.NewService(slackMock, []string{"member"}, mappings, nil, nil)
			gitHandler := handler.NewGitHandler(slackMock, users, notifierMock, config.Configuration{Notifications: []config.NotificationRule{assigneeDMRule()}})
			gitHandler.HandlePullRequestEvent(assignmentJSON(t, gh.PullRequestEvent{Action: new("opened"), Repo: assignmentRepository("example/deployments"), PullRequest: assignmentPullRequest("external", []string{"member"})}))
		})
	}
}
