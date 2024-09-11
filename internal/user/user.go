/*
git-slack-bot
Copyright (C) 2024 loveholidays

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

package user

import (
	"errors"
	"fmt"
	"git-slack-bot/internal/config"
	"git-slack-bot/internal/slack"
	"log/slog"
)

type Service interface {
	IsTeamMember(githubLogin string) bool
	GetUserDescriptor(githubLogin string) string
	IsIgnoredCommentUser(githubLogin string) bool
	IsIgnoredReviewUser(githubLogin string) bool
}

type ServiceImpl struct {
	slackConnector      slack.Interactor
	githubToSlackEmails []config.GithubEmailToSlackEmail
	githubTeamMembers   []string
	ignoredCommentUsers []string
	ignoredReviewUsers  []string
}

func NewService(slackConnector slack.Interactor, githubTeamMembers []string, githubToSlackEmails []config.GithubEmailToSlackEmail, ignoredCommentUsers, ignoredReviewUsers []string) Service {
	return &ServiceImpl{
		slackConnector:      slackConnector,
		githubToSlackEmails: githubToSlackEmails,
		githubTeamMembers:   githubTeamMembers,
		ignoredCommentUsers: ignoredCommentUsers,
		ignoredReviewUsers:  ignoredReviewUsers,
	}
}

func (s *ServiceImpl) IsTeamMember(githubLogin string) bool {
	for _, teamMember := range s.githubTeamMembers {
		if githubLogin == teamMember {
			return true
		}
	}
	return false
}

func (s *ServiceImpl) GetUserDescriptor(githubLogin string) string {
	slackUserID, err := s.getSlackUserID(githubLogin)
	if err != nil {
		slog.Warn("Unable to find slack ID for user", slog.Any("user", githubLogin), slog.Any("error", err))
		return githubLogin
	}
	return fmt.Sprintf("<@%s>", slackUserID)
}

func (s *ServiceImpl) getSlackUserID(githubLogin string) (string, error) {
	for _, githubToSlackEmail := range s.githubToSlackEmails {
		if githubToSlackEmail.GithubEmail == githubLogin {
			email, err := s.slackConnector.GetUserIDByEmail(githubToSlackEmail.SlackEmail)
			if err != nil {
				slog.Error("Received error from slack", slog.Any("error", err))
			}
			return email, err
		}
	}
	slog.Warn("could not find slack email for github login", slog.Any("user", githubLogin))
	return "", errors.New("could not find slack email for github login")
}

func (s *ServiceImpl) IsIgnoredCommentUser(githubLogin string) bool {
	for _, user := range s.ignoredCommentUsers {
		if user == githubLogin {
			return true
		}
	}
	return false
}

func (s *ServiceImpl) IsIgnoredReviewUser(githubLogin string) bool {
	for _, user := range s.ignoredReviewUsers {
		if user == githubLogin {
			return true
		}
	}
	return false
}
