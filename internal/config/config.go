//nolint:tagliatelle //Yaml camel case instead of snake case
package config

import "time"

type Configuration struct {
	GitHub GitHubConfiguration `yaml:"github"  required:"true"`
	Slack  SlackConfiguration  `yaml:"slack"  required:"true"`
	Sentry *Sentry             `yaml:"sentry"`
}

type GitHubConfiguration struct {
	Token               string   `yaml:"token"  required:"true"`
	Team                string   `yaml:"team"  required:"true"`
	Org                 string   `yaml:"org"  required:"true"`
	IgnoredPRUsers      []string `yaml:"ignoredPRUsers"`
	IgnoredRepos        []string `yaml:"ignoredRepos"`
	SecretKey           string   `yaml:"secretKey"  required:"true"`
	IgnoredCommentUsers []string `yaml:"ignoredCommentUsers"`
}

type SlackConfiguration struct {
	Token                   string                    `yaml:"token"  required:"true"`
	ChannelID               string                    `yaml:"channelID"  required:"true"`
	GithubEmailToSlackEmail []GithubEmailToSlackEmail `yaml:"githubEmailToSlackEmail"`
	EmojiConfiguration      EmojiConfiguration        `yaml:"emoji"`
}

type EmojiConfiguration struct {
	Approve string `yaml:"approve"`
	Merge   string `yaml:"merge"`
	Close   string `yaml:"close"`
}

type GithubEmailToSlackEmail struct {
	GithubEmail string `yaml:"githubEmail"`
	SlackEmail  string `yaml:"slackEmail"`
}

type Sentry struct {
	Dsn                  string        `yaml:"dsn"`
	Debug                bool          `yaml:"debug"`
	EnableTracing        bool          `yaml:"enable_tracing"`
	TracesSampleRate     float64       `yaml:"traces_sample_rate"`
	ProfilesSampleRate   float64       `yaml:"profiles_sample_rate"`
	LoggingLevel         string        `yaml:"logging_level"`
	FlushTimeoutDuration time.Duration `yaml:"flush_timeout_duration"`
}
