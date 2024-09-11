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
