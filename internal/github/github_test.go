package github_test

import (
	"context"
	"errors"
	"git-slack-bot/internal/config"
	"git-slack-bot/internal/github"
	mock_github "git-slack-bot/internal/github/mocks"
	"testing"

	gh "github.com/google/go-github/v56/github"
	"go.uber.org/mock/gomock"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestGitHub(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Github tests")
}

var _ = Describe("NewConnector", func() {
	var (
		mockCtrl   *gomock.Controller
		mockClient *mock_github.MockClient
	)

	BeforeEach(func() {
		mockCtrl = gomock.NewController(GinkgoT())
		mockClient = mock_github.NewMockClient(mockCtrl)
	})

	It("should should create new connector", func() {
		cfg := config.GitHubConfiguration{
			Token: "anyToken",
			Team:  "TestTeam",
			Org:   "TestOrg",
		}

		orgID := int64(123)
		org := gh.Organization{ID: &orgID}
		mockClient.EXPECT().GetOrg(gomock.Any(), gomock.Any()).Return(&org, nil)

		teamID := int64(234)
		teamName := "TestTeam"
		teams := []*gh.Team{
			{
				ID:   &teamID,
				Name: &teamName,
			},
		}
		mockClient.EXPECT().ListTeams(gomock.Any(), gomock.Any(), gomock.Any()).Return(teams, nil)

		connector, err := github.NewGitHubConnector(context.Background(), cfg, mockClient)
		Expect(err).ToNot(HaveOccurred())
		Expect(connector).ToNot(BeNil())
	})

	It("should fail to create client if team is not in org", func() {
		cfg := config.GitHubConfiguration{
			Token: "anyToken",
			Team:  "TestTeam",
			Org:   "TestOrg",
		}

		orgID := int64(123)
		org := gh.Organization{ID: &orgID}
		mockClient.EXPECT().GetOrg(gomock.Any(), gomock.Any()).Return(&org, nil)

		teamID := int64(234)
		teamName := "OtherTeam"

		teams := []*gh.Team{
			{
				ID:   &teamID,
				Name: &teamName,
			},
		}

		mockClient.EXPECT().ListTeams(gomock.Any(), gomock.Any(), gomock.Any()).Return(teams, nil)

		connector, err := github.NewGitHubConnector(context.Background(), cfg, mockClient)
		Expect(err).To(HaveOccurred())
		Expect(connector).To(BeNil())
	})

	It("should fail to create client if it fails to list teams", func() {
		cfg := config.GitHubConfiguration{
			Token: "anyToken",
			Team:  "TestTeam",
			Org:   "TestOrg",
		}

		orgID := int64(123)
		org := gh.Organization{ID: &orgID}
		mockClient.EXPECT().GetOrg(gomock.Any(), gomock.Any()).Return(&org, nil)

		mockClient.EXPECT().ListTeams(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, errors.New("failed to list teams"))

		connector, err := github.NewGitHubConnector(context.Background(), cfg, mockClient)
		Expect(err).To(HaveOccurred())
		Expect(connector).To(BeNil())
	})

	It("should fail to create client if it fails to find org", func() {
		cfg := config.GitHubConfiguration{
			Token: "anyToken",
			Team:  "TestTeam",
			Org:   "TestOrg",
		}

		mockClient.EXPECT().GetOrg(gomock.Any(), gomock.Any()).Return(nil, errors.New(""))

		connector, err := github.NewGitHubConnector(context.Background(), cfg, mockClient)
		Expect(err).To(HaveOccurred())
		Expect(connector).To(BeNil())
	})
})

var _ = Describe("GetTeamMembers", func() {
	var (
		mockCtrl   *gomock.Controller
		mockClient *mock_github.MockClient
		connector  *github.Connector
	)

	BeforeEach(func() {
		mockCtrl = gomock.NewController(GinkgoT())
		mockClient = mock_github.NewMockClient(mockCtrl)
		cfg := config.GitHubConfiguration{
			Token:          "anyToken",
			Team:           "TestTeam",
			Org:            "TestOrg",
			IgnoredPRUsers: []string{"BlackListed"},
			IgnoredRepos:   []string{"RepoToBeIgnored"},
		}

		orgID := int64(123)
		org := gh.Organization{ID: &orgID}
		mockClient.EXPECT().GetOrg(gomock.Any(), gomock.Any()).Return(&org, nil)
		teamID := int64(234)
		teamName := "TestTeam"

		teams := []*gh.Team{
			{
				ID:   &teamID,
				Name: &teamName,
			},
		}
		mockClient.EXPECT().ListTeams(gomock.Any(), gomock.Any(), gomock.Any()).Return(teams, nil)
		conn, err := github.NewGitHubConnector(context.Background(), cfg, mockClient)
		Expect(err).To(BeNil())
		connector = conn

	})

	It("should not return nil if failed to get team members", func() {
		mockClient.EXPECT().ListTeamMembers(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, errors.New("failed to get team members"))

		teamMembers := connector.GetTeamMembers()

		Expect(teamMembers).To(BeNil())
	})

	It("should return non blacklisted team members", func() {
		nonBlackListedTeamMember := "NonBlackListed"
		blackListedTeamMember := "BlackListed"
		teamMembersFromAPI := []*gh.User{
			{
				Login: &nonBlackListedTeamMember,
			},
			{
				Login: &blackListedTeamMember,
			},
		}
		mockClient.EXPECT().ListTeamMembers(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(teamMembersFromAPI, nil)

		teamMembers := connector.GetTeamMembers()

		expected := []string{"NonBlackListed"}

		Expect(teamMembers).To(Equal(expected))
	})
})
