package observability

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/v8tix/adk-training-go/internal/infrastructure/llm"
	"google.golang.org/adk/v2/agent"
	"google.golang.org/adk/v2/plugin"
	"google.golang.org/adk/v2/runner"
	"google.golang.org/adk/v2/session"
	"google.golang.org/genai"
)

var (
	testConfig      llm.Config
	ollamaReachable bool
)

func TestMain(m *testing.M) {
	testConfig = llm.LoadConfig()
	ollamaReachable = llm.OllamaReachable(testConfig, 2*time.Second)
	os.Exit(m.Run())
}

// askObservabilityAgent drives a single Run() call and returns the final
// text answer.
func askObservabilityAgent(ctx context.Context, r *runner.Runner, userID, sessionID, message string) (answer string, err error) {
	msg := genai.NewContentFromText(message, genai.RoleUser)
	for event, err := range r.Run(ctx, userID, sessionID, msg, agent.RunConfig{}) {
		if err != nil {
			return "", err
		}
		if event.Content == nil {
			continue
		}
		for _, p := range event.Content.Parts {
			if event.IsFinalResponse() && p.Text != "" && !p.Thought {
				answer = p.Text
			}
		}
	}
	return answer, nil
}

// TestAlertingPlugin_EscalatesOnConsecutiveErrors_Ollama exercises the
// plugin against the local Ollama backend.
func TestAlertingPlugin_EscalatesOnConsecutiveErrors_Ollama(t *testing.T) {
	if !llm.SelectedTestBackend().IncludesOllama() {
		t.Skip("skipping: TEST_BACKEND excludes ollama")
	}
	if !ollamaReachable {
		t.Skip("skipping: local Ollama server (" + testConfig.OllamaBaseURL + ") is not reachable")
	}

	cfg := testConfig
	cfg.ModelType = llm.ModelTypeOllama
	assertAlertingPluginEscalates(t, cfg)
}

// TestAlertingPlugin_EscalatesOnConsecutiveErrors_Gemini is the same
// behavior against the real Gemini backend.
func TestAlertingPlugin_EscalatesOnConsecutiveErrors_Gemini(t *testing.T) {
	if !llm.SelectedTestBackend().IncludesGemini() {
		t.Skip("skipping: TEST_BACKEND excludes gemini")
	}
	if testConfig.GoogleAPIKey == "" {
		t.Skip("skipping: GOOGLE_AI_STUDIO_API_KEY is not set")
	}

	cfg := testConfig
	cfg.ModelType = llm.ModelTypeGemini
	assertAlertingPluginEscalates(t, cfg)
}

// assertAlertingPluginEscalates drives three separate Run() calls, each
// asking the agent to deliberately fail, then reads the plugin's own
// alertTracker directly — not model text — to confirm the real Plugin
// mechanism (not just the unit-level callback logic already proven in
// alerting_plugin_test.go) genuinely intercepted three consecutive tool
// errors during a real agent run. A fourth, clean turn then proves the
// reset path fires for real too.
func assertAlertingPluginEscalates(t *testing.T, cfg llm.Config) {
	t.Helper()

	llmModel, _, err := llm.BuildModel(t.Context(), cfg)
	if err != nil {
		t.Fatalf("BuildModel() error = %v", err)
	}

	obsAgent, err := BuildRootAgent(llmModel)
	if err != nil {
		t.Fatalf("BuildRootAgent() error = %v", err)
	}

	alertingPlugin, tracker, err := newAlertingPlugin("alerting_plugin")
	if err != nil {
		t.Fatalf("newAlertingPlugin() error = %v", err)
	}

	r, err := runner.New(runner.Config{
		AppName:           "observability_test_app",
		Agent:             obsAgent,
		SessionService:    session.InMemoryService(),
		PluginConfig:      runner.PluginConfig{Plugins: []*plugin.Plugin{alertingPlugin}},
		AutoCreateSession: true,
	})
	if err != nil {
		t.Fatalf("building runner: %v", err)
	}

	const (
		userID    = "test_user"
		sessionID = "test_session"
	)
	skipIfExhausted := func(turn string, err error) {
		t.Helper()
		if err != nil && strings.Contains(err.Error(), "RESOURCE_EXHAUSTED") {
			t.Skipf("skipping: Gemini free-tier quota exhausted for today (%v) — this is an external rate limit, not a code defect; re-run once it resets (%s)", err, turn)
		}
	}

	for i := 1; i <= alertEscalationThreshold; i++ {
		_, err := askObservabilityAgent(t.Context(), r, userID, sessionID, "Please call risky_operation and make it FAIL.")
		skipIfExhausted("failing turn", err)
		if err != nil {
			t.Fatalf("askObservabilityAgent(failing turn %d) error = %v", i, err)
		}
	}

	if tracker.errorCount != alertEscalationThreshold {
		t.Fatalf("tracker.errorCount = %d after %d failing turns, want %d — the real Plugin did not intercept every tool error", tracker.errorCount, alertEscalationThreshold, alertEscalationThreshold)
	}

	_, err = askObservabilityAgent(t.Context(), r, userID, sessionID, "Please call risky_operation without making it fail.")
	skipIfExhausted("clean turn", err)
	if err != nil {
		t.Fatalf("askObservabilityAgent(clean turn) error = %v", err)
	}

	if tracker.errorCount != 0 {
		t.Errorf("tracker.errorCount = %d after a clean turn, want 0 — the real Plugin did not reset on recovery", tracker.errorCount)
	}
}
