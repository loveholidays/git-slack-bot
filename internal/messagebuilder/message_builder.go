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

package messagebuilder

import (
	"fmt"
	gh "github.com/google/go-github/v56/github"
)

type MessageBuilder struct{}

func (m *MessageBuilder) BuildPRMessage(userDescriptor string, pullRequest *gh.PullRequest) string {
	return fmt.Sprintf("%s %s:\n%s", userDescriptor, *pullRequest.Title, *pullRequest.HTMLURL)
}
func (m *MessageBuilder) BuildPRCommentMessage(userDescriptor string, event gh.PullRequestReviewCommentEvent) string {
	return fmt.Sprintf("%s left a <%s|comment>:\n> @L%v %s\n%s", userDescriptor, event.Comment.GetHTMLURL(), event.Comment.GetLine(), event.GetComment().GetPath(), event.Comment.GetBody())
}

func (m *MessageBuilder) BuildIssueCommentMessage(userDescriptor string, event gh.IssueCommentEvent) string {
	return fmt.Sprintf("%s left a <%s|comment>:\n%s", userDescriptor, event.Comment.GetHTMLURL(), event.Comment.GetBody())
}
