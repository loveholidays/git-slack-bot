package sentry

import (
	"git-slack-bot/internal/config"
	"github.com/getsentry/sentry-go"
	slogmulti "github.com/samber/slog-multi"
	slogsentry "github.com/samber/slog-sentry"
	"log/slog"
	"os"
	"time"
)

type Client struct {
	flushDuration time.Duration
}

func Init(cfg config.Sentry) (*Client, error) {
	slog.Info("Initialising sentry")
	var loggingLevel = slog.LevelError

	err := loggingLevel.UnmarshalText([]byte(cfg.LoggingLevel))
	if err != nil {
		return nil, err
	}

	logger := slog.New(
		slogmulti.Fanout(
			slogsentry.Option{Level: loggingLevel}.NewSentryHandler(),
			slog.NewJSONHandler(os.Stdout, nil),
		),
	)
	slog.SetDefault(logger)

	err = sentry.Init(sentry.ClientOptions{
		Dsn:                cfg.Dsn,
		Debug:              cfg.Debug,
		EnableTracing:      cfg.EnableTracing,
		TracesSampleRate:   cfg.TracesSampleRate,
		ProfilesSampleRate: cfg.ProfilesSampleRate,
	})

	if err != nil {
		return nil, err
	}

	return &Client{flushDuration: cfg.FlushTimeoutDuration}, nil
}

func (c *Client) CleanUp() {
	sentry.Flush(c.flushDuration)
}
