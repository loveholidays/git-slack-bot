//nolint:tagliatelle // YAML configuration uses camelCase keys.
package config

import (
	"fmt"
	"regexp"
	"slices"
)

type NotificationRule struct {
	Relationship         string   `yaml:"relationship"`
	Repositories         []string `yaml:"repositories"`
	ExcludedRepositories []string `yaml:"excludedRepositories"`
	Destination          string   `yaml:"destination"`
	ChannelID            string   `yaml:"channelID"`
}

func (rule NotificationRule) MatchesRepository(repository string) bool {
	return matchesRepositories(rule.Repositories, repository, true) && !matchesRepositories(rule.ExcludedRepositories, repository, false)
}

func matchesRepositories(repositories []string, repository string, matchEmpty bool) bool {
	return (matchEmpty && len(repositories) == 0) || slices.Contains(repositories, "*") || slices.Contains(repositories, repository)
}

func (configuration Configuration) Validate() error {
	if configuration.Notifications == nil && configuration.Slack.ChannelID == "" {
		return fmt.Errorf("slack.channelID is required when notifications is omitted")
	}
	for index, rule := range configuration.Notifications {
		if err := rule.validate(); err != nil {
			return fmt.Errorf("notifications[%d]: %w", index, err)
		}
	}
	return nil
}

func (rule NotificationRule) validate() error {
	if rule.Relationship != "author" && rule.Relationship != "assignee" {
		return fmt.Errorf("relationship must be author or assignee")
	}
	if err := rule.validateDestination(); err != nil {
		return err
	}
	return validateRepositories(append(slices.Clone(rule.Repositories), rule.ExcludedRepositories...))
}

func (rule NotificationRule) validateDestination() error {
	switch rule.Destination {
	case "channel":
		if rule.ChannelID == "" {
			return fmt.Errorf("channelID is required for channel destinations")
		}
	case "directMessage":
		if rule.ChannelID != "" {
			return fmt.Errorf("channelID is only valid for channel destinations")
		}
	default:
		return fmt.Errorf("destination must be channel or directMessage")
	}
	return nil
}

func validateRepositories(repositories []string) error {
	pattern := regexp.MustCompile(`^[a-zA-Z0-9_.-]+/[a-zA-Z0-9_.-]+$`)
	for _, repository := range repositories {
		if repository != "*" && !pattern.MatchString(repository) {
			return fmt.Errorf("repository %q must be owner/repository or *", repository)
		}
	}
	return nil
}
