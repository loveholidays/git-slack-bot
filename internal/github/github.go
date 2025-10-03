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

package github

//go:generate mockgen -destination=./mocks/github.go . Client,Interactor

import (
	"context"
	"errors"
	"git-slack-bot/internal/config"
	"log/slog"

	"github.com/google/go-github/v56/github"
	"golang.org/x/oauth2"
)

type Client interface {
	ListTeams(ctx context.Context, org string, options *github.ListOptions) ([]*github.Team, error)
	ListTeamMembers(ctx context.Context, team, orgID int64, opt *github.TeamListTeamMembersOptions) ([]*github.User, error)
	GetOrg(ctx context.Context, orgName string) (*github.Organization, error)
}

type ExternalClient struct {
	client *github.Client
}

func NewExternalClient(ctx context.Context, gitHubToken string) *ExternalClient {
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: gitHubToken},
	)
	tc := oauth2.NewClient(ctx, ts)
	client := github.NewClient(tc)
	return &ExternalClient{
		client: client,
	}
}

func (c *ExternalClient) ListTeams(ctx context.Context, org string, options *github.ListOptions) ([]*github.Team, error) {
	teams, _, err := c.client.Teams.ListTeams(ctx, org, options)
	return teams, err
}

func (c *ExternalClient) ListTeamMembers(ctx context.Context, team, orgID int64, opt *github.TeamListTeamMembersOptions) ([]*github.User, error) {
	members, _, err := c.client.Teams.ListTeamMembersByID(ctx, orgID, team, opt)
	return members, err
}

func (c *ExternalClient) GetOrg(ctx context.Context, orgName string) (*github.Organization, error) {
	org, _, err := c.client.Organizations.Get(ctx, orgName)
	return org, err
}

type Interactor interface {
	GetTeamMembers() []string
}

type Connector struct {
	ctx           context.Context
	client        Client
	repoOwner     string
	orgID         int64
	teamID        int64
	userBlackList []string
}

func NewGitHubConnector(ctx context.Context, cfg config.GitHubConfiguration, client Client) (*Connector, error) {
	org, err := client.GetOrg(ctx, cfg.Org)
	if err != nil {
		return nil, err
	}
	teams, err := client.ListTeams(ctx, cfg.Org, &github.ListOptions{PerPage: 9999})
	if err != nil {
		return nil, err
	}
	for _, team := range teams {
		if *team.Name == cfg.Team {
			return &Connector{
				ctx:           ctx,
				client:        client,
				repoOwner:     cfg.Org,
				orgID:         *org.ID,
				teamID:        *team.ID,
				userBlackList: cfg.IgnoredPRUsers,
			}, nil
		}
	}
	return nil, errors.New("did not find team in organisation")
}

func (ghc *Connector) GetTeamMembers() []string {
	usersFromAPI, err := ghc.client.ListTeamMembers(ghc.ctx, ghc.teamID, ghc.orgID, &github.TeamListTeamMembersOptions{
		ListOptions: github.ListOptions{
			PerPage: 999,
		},
	})
	if err != nil {
		slog.Error("Failed to retrieve team members", slog.Any("error", err))
		return nil
	}

	var users []string
	for _, user := range usersFromAPI {
		blacklisted := false
		for _, blacklist := range ghc.userBlackList {
			if *user.Login == blacklist {
				blacklisted = true
				break
			}
		}
		if !blacklisted {
			users = append(users, *user.Login)
		}
	}

	return users
}
