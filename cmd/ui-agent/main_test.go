package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/v8tix/adk-training-go/internal/agents/uiagent"
	"github.com/v8tix/adk-training-go/internal/infrastructure/llm"
	"google.golang.org/adk/v2/agent"
	"google.golang.org/adk/v2/server/adkrest"
	"google.golang.org/adk/v2/session"
)

var testConfig llm.Config

func TestMain(m *testing.M) {
	testConfig = llm.LoadConfig()
	m.Run()
}

// newTestServer builds the real adkrest.Server directly (bypassing
// cmd/launcher's own CLI machinery) and wraps it in httptest.NewServer —
// confirmed via handler.go that adkrest.NewServer returns a plain
// http.Handler, so this is a genuine in-process HTTP server, not a mock.
//
// Built this way, routes are served at their own bare paths (/run_sse,
// /apps/...) — the "/api" prefix the browser client actually uses
// (cmd/ui-client-server's static/index.html) comes only from
// cmd/launcher/web/api's own routing wrapper (-path_prefix, default "/api"),
// confirmed in api.go's registerAPIRoutes/rewriteRedirects, not from
// adkrest.NewServer itself. This test proves the underlying server
// behavior the browser client depends on; it does not exercise the
// launcher's own /api mounting, which is trivial routing with nothing of
// this module's own lesson in it.
func newTestServer(t *testing.T, cfg llm.Config) *httptest.Server {
	t.Helper()

	llmModel, _, err := llm.BuildModel(t.Context(), cfg)
	if err != nil {
		t.Fatalf("BuildModel() error = %v", err)
	}
	rootAgent, err := uiagent.BuildRootAgent(llmModel)
	if err != nil {
		t.Fatalf("BuildRootAgent() error = %v", err)
	}

	srv, err := adkrest.NewServer(adkrest.ServerConfig{
		SessionService: session.InMemoryService(),
		AgentLoader:    agent.NewSingleLoader(rootAgent),
		// SSEWriteTimeout defaults to zero, an immediate deadline that
		// breaks every /run_sse response — confirmed live, this test failed
		// with a client-side EOF and a server-side "superfluous
		// WriteHeader" log line before this was set.
		SSEWriteTimeout: 60 * time.Second,
	})
	if err != nil {
		t.Fatalf("adkrest.NewServer() error = %v", err)
	}

	ts := httptest.NewServer(srv)
	t.Cleanup(ts.Close)
	return ts
}

// createSession mirrors the JS client's own ensureSession() — a POST to the
// session-management endpoint with an empty JSON body.
func createSession(t *testing.T, baseURL, appName, userID, sessionID string) {
	t.Helper()
	url := baseURL + "/apps/" + appName + "/users/" + userID + "/sessions/" + sessionID
	resp, err := http.Post(url, "application/json", strings.NewReader("{}"))
	if err != nil {
		t.Fatalf("creating session: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("creating session: status = %d, want %d", resp.StatusCode, http.StatusOK)
	}
}

// sseEvent is the subset of the server's own Event JSON this test needs —
// matching the JS client's own parsing shape exactly, camelCase fields per
// google.golang.org/adk/v2/server/adkrest/internal/models/event.go's own
// json tags.
type sseEvent struct {
	Content *struct {
		Parts []struct {
			Text    string `json:"text"`
			Thought bool   `json:"thought"`
		} `json:"parts"`
	} `json:"content"`
}

// runSSEAndCollectParts POSTs to /run_sse with the real camelCase body
// shape the browser client sends, reads the whole SSE response, and parses
// every "data: " line — mirroring the JS client's own parsing logic in Go,
// for regression-test purposes only (the JS itself is verified live in a
// real browser, not unit-tested — this repo has no JS test runner).
func runSSEAndCollectParts(t *testing.T, baseURL, appName, userID, sessionID, message string) []sseEvent {
	t.Helper()

	reqBody, err := json.Marshal(map[string]any{
		"appName":   appName,
		"userId":    userID,
		"sessionId": sessionID,
		"newMessage": map[string]any{
			"role":  "user",
			"parts": []map[string]string{{"text": message}},
		},
	})
	if err != nil {
		t.Fatalf("marshaling request: %v", err)
	}

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Post(baseURL+"/run_sse", "application/json", bytes.NewReader(reqBody))
	if err != nil {
		t.Fatalf("POST /run_sse: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("POST /run_sse: status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var events []sseEvent
	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := scanner.Text()
		data, ok := strings.CutPrefix(line, "data: ")
		if !ok {
			continue
		}
		var ev sseEvent
		if err := json.Unmarshal([]byte(data), &ev); err != nil {
			t.Fatalf("unmarshaling SSE event %q: %v", data, err)
		}
		events = append(events, ev)
	}
	if err := scanner.Err(); err != nil {
		t.Fatalf("reading SSE stream: %v", err)
	}
	return events
}

// TestRunSSE_StreamsBothThoughtAndFinalParts is the real proof this
// module's own client-side risk is genuine, not hypothetical: the default
// thinking-capable Ollama model emits at least one part marked
// "thought":true in the real SSE stream, alongside the real final answer —
// a naive client rendering every part's text would leak raw reasoning into
// the chat UI.
func TestRunSSE_StreamsBothThoughtAndFinalParts(t *testing.T) {
	if !llm.SelectedTestBackend().IncludesOllama() {
		t.Skip("skipping: TEST_BACKEND excludes ollama")
	}
	if !llm.OllamaReachable(testConfig, 2*time.Second) {
		t.Skip("skipping: local Ollama server (" + testConfig.OllamaBaseURL + ") is not reachable")
	}

	cfg := testConfig
	cfg.ModelType = llm.ModelTypeOllama
	ts := newTestServer(t, cfg)

	const (
		appName   = "ui_agent"
		userID    = "test_user"
		sessionID = "test_session"
	)
	createSession(t, ts.URL, appName, userID, sessionID)

	events := runSSEAndCollectParts(t, ts.URL, appName, userID, sessionID, "Say hello in exactly three words.")

	var sawThought bool
	var finalAnswer string
	for _, ev := range events {
		if ev.Content == nil {
			continue
		}
		for _, p := range ev.Content.Parts {
			if p.Thought {
				sawThought = true
				continue
			}
			if p.Text != "" {
				finalAnswer = p.Text
			}
		}
	}

	if finalAnswer == "" {
		t.Fatal("no non-thought answer text found in the SSE stream")
	}
	if !sawThought {
		t.Error("no thought=true part found in the SSE stream — the client-side filtering this module's lab teaches would have nothing to filter; re-check the model actually used is thinking-capable")
	}
}
