package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAssignmentNotificationConfiguration(t *testing.T) {
	for _, testCase := range []struct {
		name      string
		setting   string
		wantPosts int
	}{
		{name: "omitted", setting: "", wantPosts: 0},
		{name: "disabled", setting: "  notifyAssignedFluxPRs: false\n", wantPosts: 0},
		{name: "enabled", setting: "  notifyAssignedFluxPRs: true\n", wantPosts: 1},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			transport := &startupTransport{}
			installStartupDependencies(t, transport)
			configuration := "github:\n  token: test-github-token\n  org: loveholidays\n  team: Trading OS\n  secretKey: test-secret\n" + testCase.setting + "slack:\n  token: test-slack-token\n  channelID: reviews\n  githubEmailToSlackEmail:\n    - githubEmail: member\n      slackEmail: member@example.com\n"
			configurationPath := filepath.Join(t.TempDir(), "config.yaml")
			if err := os.WriteFile(configurationPath, []byte(configuration), 0o600); err != nil {
				t.Fatal(err)
			}
			t.Setenv("CONFIG_PATH", configurationPath)
			listenAndServe = func(server *http.Server) error {
				if server.Addr != ":8080" {
					t.Errorf("unexpected server address: %s", server.Addr)
				}
				sendAssignmentWebhook(t)
				return errors.New("test server stopped")
			}
			main()
			if transport.posts != testCase.wantPosts {
				t.Errorf("got %d Slack posts, want %d", transport.posts, testCase.wantPosts)
			}
		})
	}
}

func installStartupDependencies(t *testing.T, transport http.RoundTripper) {
	t.Helper()
	previousTransport := http.DefaultTransport
	previousMux := http.DefaultServeMux
	previousServer := listenAndServe
	previousLogger := slog.Default()
	http.DefaultTransport = transport
	http.DefaultServeMux = http.NewServeMux()
	t.Cleanup(func() {
		http.DefaultTransport = previousTransport
		http.DefaultServeMux = previousMux
		listenAndServe = previousServer
		slog.SetDefault(previousLogger)
	})
}

func sendAssignmentWebhook(t *testing.T) {
	t.Helper()
	body := `{"action":"assigned","assignee":{"login":"member"},"repository":{"name":"flux","full_name":"loveholidays/flux"},"pull_request":{"user":{"login":"external"},"assignees":[{"login":"member"}],"title":"Update deployment","html_url":"https://github.com/loveholidays/flux/pull/42","created_at":"2026-10-21T12:00:00Z"}}`
	signature := hmac.New(sha256.New, []byte("test-secret"))
	if _, err := signature.Write([]byte(body)); err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/git-event", strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("X-GitHub-Event", "pull_request")
	request.Header.Set("X-Hub-Signature-256", "sha256="+hex.EncodeToString(signature.Sum(nil)))
	response := httptest.NewRecorder()
	http.DefaultServeMux.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Errorf("unexpected webhook response: %d: %s", response.Code, response.Body.String())
	}
}

type startupTransport struct {
	posts int
}

func (transport *startupTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	if err := validateStartupSlackDestination(request); err != nil {
		return nil, err
	}
	responses := map[string]string{
		"/orgs/loveholidays":              `{"id":1}`,
		"/orgs/loveholidays/teams":        `[{"id":2,"name":"Trading OS"}]`,
		"/organizations/1/team/2/members": `[{"login":"member"}]`,
		"/api/users.lookupByEmail":        `{"ok":true,"user":{"id":"member-id"}}`,
		"/api/conversations.open":         `{"ok":true,"channel":{"id":"direct-message"}}`,
		"/api/conversations.history":      `{"ok":true,"messages":[]}`,
		"/api/chat.postMessage":           `{"ok":true,"channel":"direct-message","ts":"123.456"}`,
	}
	body, found := responses[request.URL.Path]
	if !found {
		return nil, fmt.Errorf("unexpected outbound request: %s", request.URL)
	}
	if request.URL.Path == "/api/chat.postMessage" {
		transport.posts++
	}

	return &http.Response{StatusCode: http.StatusOK, Header: http.Header{"Content-Type": {"application/json"}}, Body: io.NopCloser(strings.NewReader(body)), Request: request}, nil
}

func TestInvalidConfiguration(t *testing.T) {
	previousExit := exitProcess
	t.Cleanup(func() { exitProcess = previousExit })
	exitCode := 0
	exitProcess = func(code int) { exitCode = code }
	t.Setenv("CONFIG_PATH", filepath.Join(t.TempDir(), "missing.yaml"))
	main()
	if exitCode != 1 {
		t.Errorf("expected exit status 1, got %d", exitCode)
	}
}

func validateStartupSlackDestination(request *http.Request) error {
	switch request.URL.Path {
	case "/api/conversations.open":
		if request.FormValue("users") != "member-id" {
			return fmt.Errorf("unexpected DM recipient")
		}
	case "/api/conversations.history", "/api/chat.postMessage":
		if request.FormValue("channel") != "direct-message" {
			return fmt.Errorf("unexpected Slack channel")
		}
	}
	return nil
}
