package main

import (
	"context"
	"git-slack-bot/internal/config"
	"git-slack-bot/internal/github"
	"git-slack-bot/internal/handler"
	"git-slack-bot/internal/slack"
	"git-slack-bot/internal/user"
	"log/slog"
	"net/http"
	"os"
	"time"

	config_loader "github.com/loveholidays/go-config-loader"
	sl "github.com/slack-go/slack"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))

	cfg, err := config_loader.LoadConfiguration[config.Configuration](os.Getenv("CONFIG_PATH"))
	if err != nil {
		slog.Error("Failed to load configuration", slog.Any("error", err))
		os.Exit(1)
	}

	externalSlackClient := sl.New(cfg.Slack.Token)
	slackConnector := slack.NewSlackConnector(cfg.Slack, externalSlackClient)

	ctx := context.Background()
	gitHubClient := github.NewExternalClient(ctx, cfg.GitHub.Token)
	gitHubConnector, err := github.NewGitHubConnector(ctx, cfg.GitHub, gitHubClient)
	if err != nil {
		slog.Error("Failed to establish GitHub connection", slog.Any("error", err))
	}

	userService := user.NewService(slackConnector, gitHubConnector.GetTeamMembers(), cfg.Slack.GithubEmailToSlackEmail, cfg.GitHub.IgnoredCommentUsers, cfg.GitHub.IgnoredReviewUsers)
	emojiConfiguration := cfg.Slack.EmojiConfiguration
	if emojiConfiguration.Approve == "" {
		emojiConfiguration.Approve = "+1"
	}
	if emojiConfiguration.Merge == "" {
		emojiConfiguration.Merge = "merged"
	}
	if emojiConfiguration.Close == "" {
		emojiConfiguration.Close = "x"
	}
	gitHandler := handler.NewGitHandler(slackConnector, userService, emojiConfiguration, cfg.GitHub.IgnoredRepos)
	webhookEventHandler := handler.NewWebhookEventHandler([]byte(cfg.GitHub.SecretKey), gitHandler)
	http.HandleFunc("/git-event", webhookEventHandler.HandleWebhook)
	http.HandleFunc("/", webhookEventHandler.HandleHeathCheck)

	server := &http.Server{
		Addr:              ":8080",
		ReadHeaderTimeout: time.Second * 3,
	}

	err = server.ListenAndServe()
	if err != nil {
		slog.Error("Server error", slog.Any("error", err))
	}
}
