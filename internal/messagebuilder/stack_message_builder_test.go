package messagebuilder

import (
	"git-slack-bot/internal/config"

	gh "github.com/google/go-github/v93/github"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/slack-go/slack"
)

var _ = Describe("Stack message formatting", func() {
	var (
		builder MessageBuilder
		emoji   config.EmojiConfiguration
		pr      gh.PullRequest
		message *slack.Message
	)
	BeforeEach(func() {
		builder = MessageBuilder{}
		emoji = config.EmojiConfiguration{Approve: "approved-custom", Merge: "merged-custom", Close: "closed-custom"}
		pr = gh.PullRequest{Number: new(42), HTMLURL: new("https://github.com/loveholidays/flux/pull/42"), State: new("open")}
		message = &slack.Message{Msg: slack.Msg{Text: "PR Stack 123:\n- <https://github.com/loveholidays/flux/pull/50|First PR> :merged-custom:\n- <https://github.com/loveholidays/flux/pull/42|Second PR> :approved-custom:"}}
	})

	It("should rebuild in API order, escape titles, omit drafts and preserve approvals", func() {
		previous := "PR Stack 123:\n- <https://github.com/loveholidays/flux/pull/40|Merged PR> :approved-custom:\n- <https://github.com/loveholidays/flux/pull/42|Second PR> :approved-custom:\n- <https://github.com/loveholidays/flux/pull/50|First PR> :approved-custom:"
		stack := gh.PullRequestStackDetails{ID: 123, PullRequests: []*gh.PullRequestStackEntry{
			{Number: 50, Title: "First & <important>\nPR", HTMLURL: "https://github.com/loveholidays/flux/pull/50", State: "open"},
			{Number: 51, Title: "Draft PR", Draft: true},
			{Number: 42, Title: "Second PR", HTMLURL: pr.GetHTMLURL(), State: "closed"},
			{Number: 40, Title: "Merged PR", HTMLURL: "https://github.com/loveholidays/flux/pull/40", State: "closed", MergedAt: &gh.Timestamp{}},
		}}
		expected := "PR Stack 123:\n- <https://github.com/loveholidays/flux/pull/50|First &amp; &lt;important&gt; PR> :approved-custom:\n- <https://github.com/loveholidays/flux/pull/42|Second PR> :closed-custom:\n- <https://github.com/loveholidays/flux/pull/40|Merged PR> :merged-custom:"
		actual := builder.BuildPRStackMessage(stack, previous, emoji)
		Expect(actual).To(Equal(expected))
	})

	It("should render a new open PR without a status marker", func() {
		stack := gh.PullRequestStackDetails{ID: 123, PullRequests: []*gh.PullRequestStackEntry{{Number: 42, Title: "Second PR", HTMLURL: pr.GetHTMLURL(), State: "open"}}}
		expected := "PR Stack 123:\n- <https://github.com/loveholidays/flux/pull/42|Second PR>"
		actual := builder.BuildPRStackMessage(stack, "", emoji)
		Expect(actual).To(Equal(expected))
	})

	It("should replace approval with the close emoji without changing other PRs", func() {
		pr.State = new("closed")
		expected := "PR Stack 123:\n- <https://github.com/loveholidays/flux/pull/50|First PR> :merged-custom:\n- <https://github.com/loveholidays/flux/pull/42|Second PR> :closed-custom:"
		actual := builder.BuildPRStackStatusMessage(message, pr, true, emoji)
		Expect(actual).To(Equal(expected))
	})

	It("should render only the merge emoji when a merged PR also has an approval", func() {
		pr.State = new("closed")
		pr.MergedAt = &gh.Timestamp{}
		expected := "PR Stack 123:\n- <https://github.com/loveholidays/flux/pull/50|First PR> :merged-custom:\n- <https://github.com/loveholidays/flux/pull/42|Second PR> :merged-custom:"
		actual := builder.BuildPRStackStatusMessage(message, pr, true, emoji)
		Expect(actual).To(Equal(expected))
	})

	It("should remove the close emoji when a PR reopens", func() {
		message.Text = "PR Stack 123:\n- <https://github.com/loveholidays/flux/pull/50|First PR> :merged-custom:\n- <https://github.com/loveholidays/flux/pull/42|Second PR> :closed-custom:"
		expected := "PR Stack 123:\n- <https://github.com/loveholidays/flux/pull/50|First PR> :merged-custom:\n- <https://github.com/loveholidays/flux/pull/42|Second PR>"
		actual := builder.BuildPRStackStatusMessage(message, pr, false, emoji)
		Expect(actual).To(Equal(expected))
	})

	It("should remove the approval emoji when a review is dismissed", func() {
		expected := "PR Stack 123:\n- <https://github.com/loveholidays/flux/pull/50|First PR> :merged-custom:\n- <https://github.com/loveholidays/flux/pull/42|Second PR>"
		actual := builder.BuildPRStackStatusMessage(message, pr, false, emoji)
		Expect(actual).To(Equal(expected))
	})

	It("should keep exactly one approval emoji for repeated approvals", func() {
		actual := builder.BuildPRStackStatusMessage(message, pr, true, emoji)
		Expect(actual).To(Equal(message.Text))
	})

	It("should identify the PR and approval in a review reply with a body", func() {
		review := gh.PullRequestReview{State: new("approved"), Body: new("Looks good"), HTMLURL: new("https://github.com/review")}
		expected := "<@reviewer> left a <https://github.com/review|approved review> for <https://github.com/loveholidays/flux/pull/42|PR #42>.\nLooks good"
		actual := builder.BuildPRStackReviewMessage("<@reviewer>", pr, review)
		Expect(actual).To(Equal(expected))
	})

	It("should format an approval without adding an empty body", func() {
		review := gh.PullRequestReview{State: new("approved"), HTMLURL: new("https://github.com/review")}
		expected := "<@reviewer> left a <https://github.com/review|approved review> for <https://github.com/loveholidays/flux/pull/42|PR #42>."
		actual := builder.BuildPRStackReviewMessage("<@reviewer>", pr, review)
		Expect(actual).To(Equal(expected))
	})

	It("should format a changes-requested review", func() {
		review := gh.PullRequestReview{State: new("changes_requested"), Body: new("Please update this"), HTMLURL: new("https://github.com/review")}
		expected := "<@reviewer> left a <https://github.com/review|changes requested review> for <https://github.com/loveholidays/flux/pull/42|PR #42>.\nPlease update this"
		actual := builder.BuildPRStackReviewMessage("<@reviewer>", pr, review)
		Expect(actual).To(Equal(expected))
	})
})
