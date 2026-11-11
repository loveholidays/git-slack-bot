package handler

import (
	"git-slack-bot/internal/config"
	"git-slack-bot/internal/slack"
	gh "github.com/google/go-github/v56/github"
	"log/slog"
)

type notificationTarget struct {
	channelID string
	login     string
}

func (g *GitHandler) notificationDestinations(repository *gh.Repository, pullRequest *gh.PullRequest, assignment bool) []slack.Interactor {
	if g.isIgnoredRepo(repository.GetName()) {
		return nil
	}
	targets, matchesAssignment := g.notificationTargets(repository, pullRequest)
	if assignment && !matchesAssignment {
		return nil
	}
	return g.resolveTargets(targets)
}

func (g *GitHandler) resolveTargets(targets []notificationTarget) []slack.Interactor {
	var destinations []slack.Interactor
	seenTargets := map[notificationTarget]bool{}
	seenConversations := map[string]bool{}
	for _, target := range targets {
		if seenTargets[target] {
			continue
		}
		seenTargets[target] = true
		connector := g.resolveTarget(target)
		if connector == nil {
			continue
		}
		conversation := target.channelID
		if target.login != "" {
			conversation = connector.ConversationID()
		}
		if seenConversations[conversation] {
			continue
		}
		seenConversations[conversation] = true
		destinations = append(destinations, connector)
	}
	return destinations
}

func (g *GitHandler) notificationTargets(repository *gh.Repository, pullRequest *gh.PullRequest) (targets []notificationTarget, matchesAssignment bool) {
	for _, rule := range g.notificationRules {
		if !rule.MatchesRepository(repository.GetFullName()) {
			continue
		}
		matched := g.ruleTargets(rule, pullRequest)
		if rule.Relationship == "assignee" && len(matched) > 0 {
			matchesAssignment = true
		}
		targets = append(targets, matched...)
	}
	return targets, matchesAssignment
}

func (g *GitHandler) ruleTargets(rule config.NotificationRule, pullRequest *gh.PullRequest) []notificationTarget {
	users := pullRequest.Assignees
	if rule.Relationship == "author" {
		users = []*gh.User{pullRequest.GetUser()}
	}
	var targets []notificationTarget
	for _, candidate := range users {
		if !g.userService.IsTeamMember(candidate.GetLogin()) {
			continue
		}
		if rule.Destination == "channel" {
			return []notificationTarget{{channelID: rule.ChannelID}}
		}
		targets = append(targets, notificationTarget{login: candidate.GetLogin()})
	}
	return targets
}

func (g *GitHandler) resolveTarget(target notificationTarget) slack.Interactor {
	if target.login != "" {
		return g.directMessageRecipient(target.login)
	}
	if target.channelID == "" {
		return g.slackConnector
	}
	return g.slackConnector.Channel(target.channelID)
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
