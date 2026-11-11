package config_test

import (
	"git-slack-bot/internal/config"
	"reflect"
	"testing"
)

func TestNotificationConfigurationValidation(t *testing.T) {
	for _, testCase := range []struct {
		name  string
		rule  config.NotificationRule
		valid bool
	}{
		{name: "author channel", rule: config.NotificationRule{Relationship: "author", Destination: "channel", ChannelID: "reviews"}, valid: true},
		{name: "assignee DM", rule: config.NotificationRule{Relationship: "assignee", Destination: "directMessage", Repositories: []string{"example/deployments"}}, valid: true},
		{name: "invalid relationship", rule: config.NotificationRule{Relationship: "reviewer", Destination: "directMessage"}},
		{name: "invalid destination", rule: config.NotificationRule{Relationship: "author", Destination: "email"}},
		{name: "missing channel", rule: config.NotificationRule{Relationship: "author", Destination: "channel"}},
		{name: "channel on DM", rule: config.NotificationRule{Relationship: "author", Destination: "directMessage", ChannelID: "reviews"}},
		{name: "bare repository", rule: config.NotificationRule{Relationship: "author", Destination: "directMessage", Repositories: []string{"deployments"}}},
		{name: "invalid exclusion", rule: config.NotificationRule{Relationship: "author", Destination: "directMessage", ExcludedRepositories: []string{"example/*"}}},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			configuration := config.Configuration{Notifications: []config.NotificationRule{testCase.rule}}
			if err := configuration.Validate(); (err == nil) != testCase.valid {
				t.Fatalf("unexpected validation result: %v", err)
			}
		})
	}
}

func TestLegacyChannelValidation(t *testing.T) {
	configuration := config.Configuration{}
	if configuration.Validate() == nil {
		t.Fatal("expected legacy channel requirement")
	}
	configuration.Slack.ChannelID = "reviews"
	if err := configuration.Validate(); err != nil {
		t.Fatal(err)
	}
	configuration.Slack.ChannelID = ""
	configuration.Notifications = []config.NotificationRule{}
	if err := configuration.Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestLegacyConfigurationBecomesNotificationRules(t *testing.T) {
	author := config.NotificationRule{Relationship: "author", Destination: "channel", ChannelID: "reviews"}
	assignee := config.NotificationRule{Relationship: "assignee", Repositories: []string{"loveholidays/flux"}, Destination: "channel", ChannelID: "reviews"}
	explicit := config.NotificationRule{Relationship: "assignee", Destination: "directMessage"}
	for _, testCase := range []struct {
		name            string
		legacyAssignees bool
		notifications   []config.NotificationRule
		expected        []config.NotificationRule
	}{
		{name: "legacy author channel", expected: []config.NotificationRule{author}},
		{name: "legacy assignees use same channel", legacyAssignees: true, expected: []config.NotificationRule{author, assignee}},
		{name: "explicit rules replace legacy", legacyAssignees: true, notifications: []config.NotificationRule{explicit}, expected: []config.NotificationRule{explicit}},
		{name: "empty rules disable legacy", legacyAssignees: true, notifications: []config.NotificationRule{}, expected: []config.NotificationRule{}},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			configuration := config.Configuration{GitHub: config.GitHubConfiguration{NotifyAssignedFluxPRs: testCase.legacyAssignees}, Slack: config.SlackConfiguration{ChannelID: "reviews"}, Notifications: testCase.notifications}
			if actual := configuration.NotificationRules(); !reflect.DeepEqual(actual, testCase.expected) {
				t.Fatalf("got %+v, want %+v", actual, testCase.expected)
			}
		})
	}
}
