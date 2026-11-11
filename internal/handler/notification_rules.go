package handler

import (
	"git-slack-bot/internal/config"
	"git-slack-bot/internal/slack"
	gh "github.com/google/go-github/v56/github"
	"log/slog"
)

func (g *GitHandler) ruleRecipients(rule config.NotificationRule, pullRequest *gh.PullRequest) []string {
	users := pullRequest.Assignees
	if rule.Relationship == "author" {
		users = []*gh.User{pullRequest.GetUser()}
	}
	var recipients []string
	for _, candidate := range users {
		if g.userService.IsTeamMember(candidate.GetLogin()) {
			recipients = append(recipients, candidate.GetLogin())
		}
	}
	return recipients
}

func (g *GitHandler) hasMatchingRule(repository *gh.Repository, pullRequest *gh.PullRequest) bool {
	for _, rule := range g.githubConfiguration.NotificationRules {
		if rule.MatchesRepository(repository.GetFullName()) && len(g.ruleRecipients(rule, pullRequest)) > 0 {
			return true
		}
	}
	return false
}

func (g *GitHandler) matchesAssignmentRule(event *gh.PullRequestEvent) bool {
	if !g.userService.IsTeamMember(event.GetAssignee().GetLogin()) {
		return false
	}
	for _, rule := range g.githubConfiguration.NotificationRules {
		if rule.Relationship == "assignee" && rule.MatchesRepository(event.GetRepo().GetFullName()) {
			return true
		}
	}
	return false
}

func (g *GitHandler) forEachRuleDestination(repository *gh.Repository, pullRequest *gh.PullRequest, handle func(*GitHandler)) {
	destinations := map[string]bool{}
	for _, connector := range g.matchingRuleConnectors(repository, pullRequest) {
		id := connector.ConversationID()
		if destinations[id] {
			continue
		}
		destinations[id] = true
		destination := *g
		destination.slackConnector = connector
		handle(&destination)
	}
}

func (g *GitHandler) matchingRuleConnectors(repository *gh.Repository, pullRequest *gh.PullRequest) []slack.Interactor {
	var connectors []slack.Interactor
	for _, rule := range g.githubConfiguration.NotificationRules {
		if rule.MatchesRepository(repository.GetFullName()) {
			connectors = append(connectors, g.ruleConnectors(rule, pullRequest)...)
		}
	}
	return connectors
}

func (g *GitHandler) ruleConnectors(rule config.NotificationRule, pullRequest *gh.PullRequest) []slack.Interactor {
	recipients := g.ruleRecipients(rule, pullRequest)
	if len(recipients) == 0 {
		return nil
	}
	if rule.Destination == "channel" {
		return []slack.Interactor{g.slackConnector.Channel(rule.ChannelID)}
	}
	var connectors []slack.Interactor
	seen := map[string]bool{}
	for _, login := range recipients {
		if seen[login] {
			continue
		}
		seen[login] = true
		if connector := g.directMessageRecipient(login); connector != nil {
			connectors = append(connectors, connector)
		}
	}
	return connectors
}

func (g *GitHandler) directMessageRecipient(login string) slack.Interactor {
	userID, err := g.userService.GetSlackUserID(login)
	if err != nil {
		slog.Error("Could not resolve PR recipient on Slack", slog.String("user", login), slog.Any("error", err))
		return nil
	}
	connector, err := g.slackConnector.DirectMessage(userID)
	if err != nil {
		slog.Error("Could not open PR recipient direct message", slog.String("user", login), slog.Any("error", err))
		return nil
	}
	return connector
}
