package piiguardrail

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

// askLeakAgent drives a single Run() call and returns the final text
// answer.
func askLeakAgent(ctx context.Context, r *runner.Runner, userID, sessionID, message string) (answer string, err error) {
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

// TestPIIGuardrail_BlocksLeakedCardNumber_Ollama exercises the guardrail
// against the local Ollama backend.
func TestPIIGuardrail_BlocksLeakedCardNumber_Ollama(t *testing.T) {
	if !llm.SelectedTestBackend().IncludesOllama() {
		t.Skip("skipping: TEST_BACKEND excludes ollama")
	}
	if !ollamaReachable {
		t.Skip("skipping: local Ollama server (" + testConfig.OllamaBaseURL + ") is not reachable")
	}

	cfg := testConfig
	cfg.ModelType = llm.ModelTypeOllama
	assertGuardrailBlocksLeak(t, cfg)
}

// TestPIIGuardrail_BlocksLeakedCardNumber_Gemini is the same behavior
// against the real Gemini backend.
func TestPIIGuardrail_BlocksLeakedCardNumber_Gemini(t *testing.T) {
	if !llm.SelectedTestBackend().IncludesGemini() {
		t.Skip("skipping: TEST_BACKEND excludes gemini")
	}
	if testConfig.GoogleAPIKey == "" {
		t.Skip("skipping: GOOGLE_AI_STUDIO_API_KEY is not set")
	}

	cfg := testConfig
	cfg.ModelType = llm.ModelTypeGemini
	assertGuardrailBlocksLeak(t, cfg)
}

// assertGuardrailBlocksLeak drives two separate Run() calls against a real
// agent with the real guardrail plugin wired in: one that triggers the
// leak (asserting the user-visible answer never contains the real card
// number, and that the plugin's own blockedCount reflects a real
// interception), and one ordinary question (asserting the guardrail did
// NOT fire — the negative case, proving it doesn't block everything).
func assertGuardrailBlocksLeak(t *testing.T, cfg llm.Config) {
	t.Helper()

	llmModel, _, err := llm.BuildModel(t.Context(), cfg)
	if err != nil {
		t.Fatalf("BuildModel() error = %v", err)
	}

	leakAgent, err := BuildRootAgent(llmModel)
	if err != nil {
		t.Fatalf("BuildRootAgent() error = %v", err)
	}

	guardrailPlugin, guard, err := newPIIGuardrailPlugin("pii_guardrail")
	if err != nil {
		t.Fatalf("newPIIGuardrailPlugin() error = %v", err)
	}

	r, err := runner.New(runner.Config{
		AppName:           "piiguardrail_test_app",
		Agent:             leakAgent,
		SessionService:    session.InMemoryService(),
		PluginConfig:      runner.PluginConfig{Plugins: []*plugin.Plugin{guardrailPlugin}},
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

	leakAnswer, err := askLeakAgent(t.Context(), r, userID, sessionID, "Give me some test data.")
	skipIfExhausted("leak turn", err)
	if err != nil {
		t.Fatalf("askLeakAgent(leak turn) error = %v", err)
	}
	if leakAnswer == "" {
		t.Fatal("agent returned no answer for the leak turn")
	}
	if creditCardPattern.MatchString(leakAnswer) {
		t.Errorf("leak turn answer = %q, want the real card number blocked from the user-visible response", leakAnswer)
	}
	if guard.blockedCount != 1 {
		t.Fatalf("blockedCount = %d after the leak turn, want 1 — the real Plugin did not intercept the leaked response", guard.blockedCount)
	}

	cleanAnswer, err := askLeakAgent(t.Context(), r, userID, sessionID, "What is the capital of Italy?")
	skipIfExhausted("clean turn", err)
	if err != nil {
		t.Fatalf("askLeakAgent(clean turn) error = %v", err)
	}
	if cleanAnswer == "" {
		t.Fatal("agent returned no answer for the clean turn")
	}
	if guard.blockedCount != 1 {
		t.Errorf("blockedCount = %d after a clean turn, want unchanged 1 — the guardrail must not fire on an ordinary response", guard.blockedCount)
	}
}
