package handler_test

import (
	"git-slack-bot/internal/config"
	"git-slack-bot/internal/handler"
	mock_slack "git-slack-bot/internal/slack/mocks"
	"git-slack-bot/internal/user"
	gh "github.com/google/go-github/v56/github"
	sl "github.com/slack-go/slack"
	"go.uber.org/mock/gomock"
	"testing"
)

func authorChannelRule() config.NotificationRule {
	return config.NotificationRule{Relationship: "author", Repositories: []string{"*"}, Destination: "channel", ChannelID: "reviews"}
}

func assigneeDMRule() config.NotificationRule {
	return config.NotificationRule{Relationship: "assignee", Repositories: []string{"example/deployments"}, Destination: "directMessage"}
}

func TestNotificationRules(t *testing.T) {
	excluded := authorChannelRule()
	excluded.ExcludedRepositories = []string{"example/deployments"}
	authorDM := authorChannelRule()
	authorDM.Destination, authorDM.ChannelID = "directMessage", ""
	assigneeChannel := assigneeDMRule()
	assigneeChannel.Destination, assigneeChannel.ChannelID = "channel", "reviews"
	for _, testCase := range []struct {
		name         string
		rules        []config.NotificationRule
		author       string
		assignees    []string
		destinations []string
	}{
		{name: "author channel", rules: []config.NotificationRule{authorChannelRule()}, author: "member", destinations: []string{"reviews"}},
		{name: "assignee DM", rules: []config.NotificationRule{assigneeDMRule()}, author: "external", assignees: []string{"member"}, destinations: []string{"dm-member"}},
		{name: "author and assignee both match", rules: []config.NotificationRule{authorChannelRule(), assigneeDMRule()}, author: "member", assignees: []string{"member"}, destinations: []string{"reviews", "dm-member"}},
		{name: "author DM", rules: []config.NotificationRule{authorDM}, author: "member", destinations: []string{"dm-member"}},
		{name: "assignee channel", rules: []config.NotificationRule{assigneeChannel}, author: "external", assignees: []string{"member", "second"}, destinations: []string{"reviews"}},
		{name: "overlapping channels", rules: []config.NotificationRule{authorChannelRule(), assigneeChannel}, author: "member", assignees: []string{"member"}, destinations: []string{"reviews"}},
		{name: "overlapping DMs", rules: []config.NotificationRule{authorDM, assigneeDMRule()}, author: "member", assignees: []string{"member"}, destinations: []string{"dm-member"}},
		{name: "multiple assignees", rules: []config.NotificationRule{assigneeDMRule()}, author: "external", assignees: []string{"member", "second", "member"}, destinations: []string{"dm-member", "dm-second"}},
		{name: "exclusion", rules: []config.NotificationRule{excluded}, author: "member"},
		{name: "exclusion only affects its rule", rules: []config.NotificationRule{excluded, assigneeDMRule()}, author: "member", assignees: []string{"member"}, destinations: []string{"dm-member"}},
		{name: "non member", rules: []config.NotificationRule{authorChannelRule(), assigneeDMRule()}, author: "external", assignees: []string{"external"}},
		{name: "explicit empty disables legacy", rules: []config.NotificationRule{}, author: "member", assignees: []string{"member"}},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			gitHandler, destinations := ruleTestHandler(t, testCase.rules)
			for _, id := range testCase.destinations {
				destinations[id].EXPECT().SendMessageOnce(gomock.Any())
			}
			gitHandler.HandlePullRequestEvent(assignmentJSON(t, gh.PullRequestEvent{Action: gh.String("opened"), Repo: assignmentRepository("example/deployments"), PullRequest: assignmentPullRequest(testCase.author, testCase.assignees)}))
		})
	}
}

func ruleTestHandler(t *testing.T, rules []config.NotificationRule) (gitHandler *handler.GitHandler, destinations map[string]*mock_slack.MockInteractor) {
	t.Helper()
	controller := gomock.NewController(t)
	root := mock_slack.NewMockInteractor(controller)
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
	configuration := config.GitHubConfiguration{NotificationRules: rules, NotifyAssignedFluxPRs: true}
	return handler.NewGitHandler(root, users, validEmojis(), configuration), destinations
}

func TestRuleThreadsFollowBothDestinations(t *testing.T) {
	gitHandler, destinations := ruleTestHandler(t, []config.NotificationRule{authorChannelRule(), assigneeDMRule(), assigneeDMRule()})
	for _, id := range []string{"reviews", "dm-member"} {
		message := &sl.Message{Msg: sl.Msg{Timestamp: "123.456"}}
		destination := destinations[id]
		destination.EXPECT().GetMessage(gomock.Any()).Return(message, nil).Times(3)
		destination.EXPECT().SendReply(message, gomock.Any()).Times(2)
		destination.EXPECT().AddReactionToMessage("+1", message)
		destination.EXPECT().AddReactionToMessage("x", message)
	}
	repository := assignmentRepository("example/deployments")
	pr := assignmentPullRequest("member", []string{"member"})
	gitHandler.HandlePullRequestReviewEvent(assignmentJSON(t, gh.PullRequestReviewEvent{Action: gh.String("submitted"), Repo: repository, PullRequest: pr, Review: &gh.PullRequestReview{User: &gh.User{Login: gh.String("external")}, State: gh.String("approved"), Body: gh.String("Looks good"), HTMLURL: gh.String("https://github.com/review")}}))
	gitHandler.HandleIssueCommentEvent(assignmentJSON(t, gh.IssueCommentEvent{Action: gh.String("created"), Repo: repository, Issue: &gh.Issue{User: pr.User, Assignees: pr.Assignees, HTMLURL: pr.HTMLURL}, Comment: &gh.IssueComment{User: &gh.User{Login: gh.String("external")}}}))
	gitHandler.HandlePullRequestEvent(assignmentJSON(t, gh.PullRequestEvent{Action: gh.String("closed"), Repo: repository, PullRequest: pr}))
}

func TestLegacyAssignedPRsKeepChannelDelivery(t *testing.T) {
	connector := mock_slack.NewMockInteractor(gomock.NewController(t))
	users := user.NewService(connector, []string{"member"}, nil, nil, nil)
	gitHandler := handler.NewGitHandler(connector, users, validEmojis(), config.GitHubConfiguration{NotifyAssignedFluxPRs: true})
	connector.EXPECT().SendMessageOnce(gomock.Any())
	gitHandler.HandlePullRequestEvent(assignmentJSON(t, gh.PullRequestEvent{Action: gh.String("assigned"), Assignee: &gh.User{Login: gh.String("member")}, Repo: assignmentRepository("loveholidays/flux"), PullRequest: assignmentPullRequest("external", []string{"member"})}))
}
