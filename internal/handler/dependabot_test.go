package handler_test

import (
	"git-slack-bot/internal/config"
	"git-slack-bot/internal/handler"
	mock_slack "git-slack-bot/internal/slack/mocks"
	"git-slack-bot/internal/user"
	"testing"

	gh "github.com/google/go-github/v56/github"
	"github.com/slack-go/slack"
	"go.uber.org/mock/gomock"
)

func TestDependabotNotifications(t *testing.T) {
	for _, testCase := range []struct {
		name, author, repository, action           string
		draft, ignored, excluded, disabled, notify bool
	}{
		{name: "Shopping repository", author: "dependabot[bot]", repository: "loveholidays/backend", action: "opened", notify: true},
		{name: "ready for review", author: "dependabot[bot]", repository: "loveholidays/backend", action: "ready_for_review", notify: true},
		{name: "draft", author: "dependabot[bot]", repository: "loveholidays/backend", action: "opened", draft: true},
		{name: "other repository", author: "dependabot[bot]", repository: "loveholidays/other", action: "opened"},
		{name: "other owner", author: "dependabot[bot]", repository: "another-org/backend", action: "opened"},
		{name: "global exclusion", author: "dependabot[bot]", repository: "loveholidays/backend", action: "opened", ignored: true},
		{name: "rule exclusion", author: "dependabot[bot]", repository: "loveholidays/backend", action: "opened", excluded: true},
		{name: "other bot", author: "renovate[bot]", repository: "loveholidays/backend", action: "opened"},
		{name: "human author", author: "external", repository: "loveholidays/backend", action: "opened"},
		{name: "opt in required", author: "dependabot[bot]", repository: "loveholidays/backend", action: "opened", disabled: true},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			root := mock_slack.NewMockInteractor(gomock.NewController(t))
			destination := mock_slack.NewMockInteractor(gomock.NewController(t))
			rule := config.NotificationRule{Relationship: "dependabot", Repositories: []string{"loveholidays/backend"}, Destination: "channel", ChannelID: "reviews"}
			if testCase.excluded {
				rule.ExcludedRepositories = []string{"loveholidays/backend"}
			}
			configuration := config.Configuration{Slack: config.SlackConfiguration{ChannelID: "reviews"}}
			if !testCase.disabled {
				// Matching rules targeting the same channel must still produce only one post.
				configuration.Notifications = []config.NotificationRule{rule, rule}
			}
			if testCase.ignored {
				configuration.GitHub.IgnoredRepos = []string{"backend"}
			}
			if testCase.notify {
				root.EXPECT().Channel("reviews").Return(destination).Times(1)
				destination.EXPECT().SendMessageOnce(gomock.Any())
			}
			users := user.NewService(root, []string{"member"}, nil, nil, nil)
			gitHandler := handler.NewGitHandler(root, users, validEmojis(), configuration)
			pr := assignmentPullRequest(testCase.author, nil)
			pr.Draft = gh.Bool(testCase.draft)
			gitHandler.HandlePullRequestEvent(assignmentJSON(t, gh.PullRequestEvent{Action: gh.String(testCase.action), Repo: assignmentRepository(testCase.repository), PullRequest: pr}))
		})
	}
}

func TestDependabotThreadsAndReactions(t *testing.T) {
	rule := config.NotificationRule{Relationship: "dependabot", Repositories: []string{"example/deployments"}, Destination: "channel", ChannelID: "reviews"}
	gitHandler, destinations := ruleTestHandler(t, []config.NotificationRule{rule})
	destination := destinations["reviews"]
	message := &slack.Message{Msg: slack.Msg{Timestamp: "123.456"}}
	destination.EXPECT().GetMessage(gomock.Any()).Return(message, nil).Times(4)
	destination.EXPECT().AddReactionToMessage("+1", message)
	destination.EXPECT().AddReactionToMessage("x", message)
	destination.EXPECT().RemoveReactionFromMessage("x", message)
	destination.EXPECT().SendReply(message, gomock.Any())
	repository := assignmentRepository("example/deployments")
	pr := assignmentPullRequest("dependabot[bot]", nil)
	gitHandler.HandlePullRequestReviewEvent(assignmentJSON(t, gh.PullRequestReviewEvent{Action: gh.String("submitted"), Repo: repository, PullRequest: pr, Review: &gh.PullRequestReview{User: &gh.User{Login: gh.String("external")}, State: gh.String("approved")}}))
	gitHandler.HandleIssueCommentEvent(assignmentJSON(t, gh.IssueCommentEvent{Action: gh.String("created"), Repo: repository, Issue: &gh.Issue{User: pr.User, HTMLURL: pr.HTMLURL}, Comment: &gh.IssueComment{User: &gh.User{Login: gh.String("external")}}}))
	gitHandler.HandlePullRequestEvent(assignmentJSON(t, gh.PullRequestEvent{Action: gh.String("closed"), Repo: repository, PullRequest: pr}))
	gitHandler.HandlePullRequestEvent(assignmentJSON(t, gh.PullRequestEvent{Action: gh.String("reopened"), Repo: repository, PullRequest: pr}))
}
