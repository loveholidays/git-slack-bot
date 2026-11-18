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

package messagebuilder

import (
	"fmt"
	"git-slack-bot/internal/config"
	gh "github.com/google/go-github/v93/github"
	sl "github.com/slack-go/slack"
	"strings"
)

type MessageBuilder struct{}

func (m *MessageBuilder) BuildPRMessage(userDescriptor string, pullRequest *gh.PullRequest) string {
	return fmt.Sprintf("%s %s:\n%s", userDescriptor, *pullRequest.Title, *pullRequest.HTMLURL)
}
func (m *MessageBuilder) BuildPRStackMessage(stack gh.PullRequestStackDetails, previousText string, emoji config.EmojiConfiguration) string {
	lines := []string{fmt.Sprintf("PR Stack %d:", stack.ID)}
	titleEscaper := strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", "\n", " ", "\r", " ")
	for _, pullRequest := range stack.PullRequests {
		if pullRequest.Draft {
			continue
		}
		title := titleEscaper.Replace(pullRequest.Title)
		previousLine := stackPullRequestLine(previousText, pullRequest.HTMLURL)
		approved := strings.HasSuffix(previousLine, " :"+strings.Trim(emoji.Approve, ":")+":")
		status := pullRequestStatus(pullRequest.State, pullRequest.MergedAt != nil, approved, emoji)
		lines = append(lines, fmt.Sprintf("- <%s|%s>%s", pullRequest.HTMLURL, title, status))
	}
	return strings.Join(lines, "\n")
}

func (m *MessageBuilder) BuildPRStackStatusMessage(message *sl.Message, pullRequest gh.PullRequest, approved bool, emoji config.EmojiConfiguration) string {
	lines := strings.Split(message.Text, "\n")
	linkPrefix := "<" + pullRequest.GetHTMLURL() + "|"
	status := pullRequestStatus(pullRequest.GetState(), pullRequest.MergedAt != nil, approved, emoji)
	for i, line := range lines {
		if !strings.Contains(line, linkPrefix) {
			continue
		}
		linkEnd := strings.LastIndex(line, ">")
		if linkEnd >= 0 {
			lines[i] = line[:linkEnd+1] + status
		}
	}
	return strings.Join(lines, "\n")
}

func (m *MessageBuilder) BuildPRStackReviewMessage(userDescriptor string, pullRequest gh.PullRequest, review gh.PullRequestReview) string {
	state := strings.ReplaceAll(strings.ToLower(review.GetState()), "_", " ")
	message := fmt.Sprintf("%s left a <%s|%s review> for <%s|PR #%d>.", userDescriptor, review.GetHTMLURL(), state, pullRequest.GetHTMLURL(), pullRequest.GetNumber())
	if review.GetBody() != "" {
		message += "\n" + review.GetBody()
	}
	return message
}

func pullRequestStatus(state string, merged, approved bool, emoji config.EmojiConfiguration) string {
	name := ""
	switch {
	case state == "closed" && !merged:
		name = emoji.Close
	case merged:
		name = emoji.Merge
	case approved:
		name = emoji.Approve
	}
	if name == "" {
		return ""
	}
	return " :" + strings.Trim(name, ":") + ":"
}

func stackPullRequestLine(text, url string) string {
	for _, line := range strings.Split(text, "\n") {
		if strings.HasPrefix(line, "- <"+url+"|") {
			return line
		}
	}
	return ""
}

func (m *MessageBuilder) BuildPRCommentMessage(userDescriptor string, event gh.PullRequestReviewCommentEvent) string {
	return fmt.Sprintf("%s left a <%s|comment>:\n> @L%v %s\n%s", userDescriptor, event.Comment.GetHTMLURL(), event.Comment.GetLine(), event.GetComment().GetPath(), event.Comment.GetBody())
}

func (m *MessageBuilder) BuildPRReviewMessage(userDescriptor string, review *gh.PullRequestReview) string {
	return fmt.Sprintf("%s left a <%s|review>:\n%s", userDescriptor, review.GetHTMLURL(), review.GetBody())
}

func (m *MessageBuilder) BuildIssueCommentMessage(userDescriptor string, event gh.IssueCommentEvent) string {
	return fmt.Sprintf("%s left a <%s|comment>:\n%s", userDescriptor, event.Comment.GetHTMLURL(), event.Comment.GetBody())
}
