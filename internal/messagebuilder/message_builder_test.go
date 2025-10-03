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
	_ "embed"
	"encoding/json"
	gh "github.com/google/go-github/v56/github"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"testing"
)

func TestMessageBuilder(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Message builder tests")
}

var (
	//go:embed example-pr-opened.json
	prJSONData []byte
	//go:embed example-pr-comment.json
	prCommentJSONData []byte
	//go:embed example-pr-top-level-comment.json
	issueCommentJSONData []byte
)

var _ = Describe("Message Builder", func() {

	It("should build a PR message", func() {
		messageBuilder := MessageBuilder{}
		var pullRequestEvent gh.PullRequestEvent
		err := json.Unmarshal(prJSONData, &pullRequestEvent)
		Expect(err).ToNot(HaveOccurred())

		actual := messageBuilder.BuildPRMessage("@George", pullRequestEvent.PullRequest)

		expected := `@George [GS] Test slack id change:
https://github.com/loveholidays/hotels-and-ancillaries/pull/808`

		Expect(actual).To(Equal(expected))
	})

	It("should build a PR comment message", func() {
		messageBuilder := MessageBuilder{}

		var pullRequestComment gh.PullRequestReviewCommentEvent
		err := json.Unmarshal(prCommentJSONData, &pullRequestComment)
		Expect(err).ToNot(HaveOccurred())

		actual := messageBuilder.BuildPRCommentMessage("@George", pullRequestComment)

		expected := `@George left a <https://github.com/loveholidays/hotels-and-ancillaries/pull/808#discussion_r1394053818|comment>:
> @L1 src/main/resources/application.properties
Wow comments work too now?`

		Expect(actual).To(Equal(expected))
	})

	It("should build an issue comment message", func() {
		messageBuilder := MessageBuilder{}

		var issueComment gh.IssueCommentEvent
		err := json.Unmarshal(issueCommentJSONData, &issueComment)
		Expect(err).ToNot(HaveOccurred())

		actual := messageBuilder.BuildIssueCommentMessage("@George", issueComment)

		expected := `@George left a <https://github.com/loveholidays/hotels-and-ancillaries/pull/1015#issuecomment-1924011855|comment>:
Just leaving a top level comment here`

		Expect(actual).To(Equal(expected))
	})
})
