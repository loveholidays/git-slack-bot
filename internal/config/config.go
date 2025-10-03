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

//nolint:tagliatelle //Yaml camel case instead of snake case
package config

type Configuration struct {
	GitHub GitHubConfiguration `yaml:"github"  required:"true"`
	Slack  SlackConfiguration  `yaml:"slack"  required:"true"`
}

type GitHubConfiguration struct {
	Token               string   `yaml:"token"  required:"true"`
	Team                string   `yaml:"team"  required:"true"`
	Org                 string   `yaml:"org"  required:"true"`
	IgnoredPRUsers      []string `yaml:"ignoredPRUsers"`
	IgnoredRepos        []string `yaml:"ignoredRepos"`
	SecretKey           string   `yaml:"secretKey"  required:"true"`
	IgnoredCommentUsers []string `yaml:"ignoredCommentUsers"`
	IgnoredReviewUsers  []string `yaml:"ignoredReviewUsers"`
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
