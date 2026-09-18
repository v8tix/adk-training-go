package main

import (
	"context"
	"strings"
	"testing"

	"github.com/redis/go-redis/v9"
	tcredis "github.com/testcontainers/testcontainers-go/modules/redis"

	"github.com/v8tix/adk-training-go/internal/infrastructure/llm"
)

// TestParseArgs is the regression guard for a real bug found live: the
// original CLI silently ignored any argument beyond "set"/"ask" — passing
// `persistent-agent set "My favorite color is teal."` set the hardcoded
// "blue" fact anyway, with no error and no indication the extra argument
// did nothing.
func TestParseArgs(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		wantMessage string
		wantErr     bool
	}{
		{name: "set with no color defaults to blue", args: []string{"set"}, wantMessage: "My favorite color is blue."},
		{name: "set with a custom color", args: []string{"set", "teal"}, wantMessage: "My favorite color is teal."},
		{name: "set with a multi-word color", args: []string{"set", "sky", "blue"}, wantMessage: "My favorite color is sky blue."},
		{name: "ask with no arguments", args: []string{"ask"}, wantMessage: "What is my favorite color?"},
		{name: "ask with an unexpected argument errors", args: []string{"ask", "teal"}, wantErr: true},
		{name: "no arguments at all errors", args: []string{}, wantErr: true},
		{name: "unknown subcommand errors", args: []string{"remember"}, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseArgs(tt.args)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("parseArgs(%v) error = nil, want an error", tt.args)
				}
				return
			}
			if err != nil {
				t.Fatalf("parseArgs(%v) unexpected error: %v", tt.args, err)
			}
			if got != tt.wantMessage {
				t.Errorf("parseArgs(%v) = %q, want %q", tt.args, got, tt.wantMessage)
			}
		})
	}
}

// TestPersistentAgent_RemembersAcrossFreshInstances is the real proof of
// this module's lesson: two completely independent runTurn calls — each
// building its own fresh model, agent, session.Service, and runner.Runner,
// exactly as two separate `persistent-agent set`/`persistent-agent ask`
// process invocations would — share state only because they're both backed
// by the same Redis. A fresh session.Service per call is the meaningful
// unit here; actually forking a second OS process would prove the same
// thing with more test-harness complexity for no additional confidence.
func TestPersistentAgent_RemembersAcrossFreshInstances(t *testing.T) {
	ctx := context.Background()
	cfg := llm.LoadConfig()
	ollamaReachable := llm.OllamaReachable(cfg, 0)
	if !llm.SelectedTestBackend().IncludesOllama() || !ollamaReachable {
		t.Skip("skipping: requires a reachable local Ollama server")
	}

	container, err := tcredis.Run(ctx, "redis:7-alpine")
	if err != nil {
		t.Skipf("skipping: could not start a Redis container via Testcontainers (%v) — Docker may be unavailable", err)
	}
	t.Cleanup(func() {
		if err := container.Terminate(ctx); err != nil {
			t.Logf("terminating container: %v", err)
		}
	})

	connStr, err := container.ConnectionString(ctx)
	if err != nil {
		t.Fatalf("ConnectionString() error = %v", err)
	}
	opts, err := redis.ParseURL(connStr)
	if err != nil {
		t.Fatalf("ParseURL() error = %v", err)
	}
	addr := opts.Addr

	if _, err := runTurn(ctx, addr, "My favorite color is teal."); err != nil {
		t.Fatalf("first runTurn() (set) error = %v", err)
	}

	answer, err := runTurn(ctx, addr, "What is my favorite color?")
	if err != nil {
		t.Fatalf("second runTurn() (ask) error = %v", err)
	}
	if !strings.Contains(strings.ToLower(answer), "teal") {
		t.Errorf("answer = %q, want it to mention the favorite color set by a completely separate runTurn call", answer)
	}
}
