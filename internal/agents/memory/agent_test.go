package memory

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/v8tix/adk-training-go/internal/infrastructure/llm"
	"google.golang.org/adk/v2/agent"
	"google.golang.org/adk/v2/runner"
	"google.golang.org/genai"
)

// testConfig and ollamaReachable are set once by TestMain, matching
// internal/agents/calculator's pattern.
var (
	testConfig      llm.Config
	ollamaReachable bool
)

func TestMain(m *testing.M) {
	testConfig = llm.LoadConfig()
	ollamaReachable = llm.OllamaReachable(testConfig, 2*time.Second)
	os.Exit(m.Run())
}

// askMemoryAgent drives the real BuildRootAgent through a single Run() call.
// It returns the final text answer, plus wantToolCalled's actual
// FunctionResponse.Response map, if that tool was really called (nil
// otherwise). The text alone is not sufficient proof a tool fired: runner.Run
// re-sends the full session history to the model on every call, so a model
// could answer "What is my name?" correctly by re-reading its own prior
// reply in that history, without ever calling recall_name — the same gap
// internal/agents/calculator/agent_test.go's askCalculator was written to
// close for its own tool, via the same FunctionResponse check.
func askMemoryAgent(ctx context.Context, r *runner.Runner, sessionID, question, wantToolCalled string) (answer string, toolResponse map[string]any, err error) {
	msg := genai.NewContentFromText(question, genai.RoleUser)
	for event, err := range r.Run(ctx, "test_user", sessionID, msg, agent.RunConfig{}) {
		if err != nil {
			return "", nil, err
		}
		if event.Content == nil {
			continue
		}
		for _, p := range event.Content.Parts {
			if p.FunctionResponse != nil && p.FunctionResponse.Name == wantToolCalled {
				toolResponse = p.FunctionResponse.Response
			}
			if event.IsFinalResponse() && p.Text != "" && !p.Thought {
				answer = p.Text
			}
		}
	}
	return answer, toolResponse, nil
}

// TestMemory_RemembersNameAcrossTurns_Ollama exercises this module's own
// finding against the local Ollama backend: state written in one Run() call
// is visible in a later, separate Run() call against the same session —
// confirmed live (module-10). Custom function tools, state access included,
// need no cloud fallback, matching module-9's precedent.
func TestMemory_RemembersNameAcrossTurns_Ollama(t *testing.T) {
	if !llm.SelectedTestBackend().IncludesOllama() {
		t.Skip("skipping: TEST_BACKEND excludes ollama")
	}
	if !ollamaReachable {
		t.Skip("skipping: local Ollama server (" + testConfig.OllamaBaseURL + ") is not reachable")
	}

	cfg := testConfig
	cfg.ModelType = llm.ModelTypeOllama
	assertMemoryRemembersName(t, cfg)
}

// TestMemory_RemembersNameAcrossTurns_Gemini is the same behavior against
// the real Gemini backend.
func TestMemory_RemembersNameAcrossTurns_Gemini(t *testing.T) {
	if !llm.SelectedTestBackend().IncludesGemini() {
		t.Skip("skipping: TEST_BACKEND excludes gemini")
	}
	if testConfig.GoogleAPIKey == "" {
		t.Skip("skipping: GOOGLE_AI_STUDIO_API_KEY is not set")
	}

	cfg := testConfig
	cfg.ModelType = llm.ModelTypeGemini
	assertMemoryRemembersName(t, cfg)
}

// assertMemoryRemembersName builds the model from cfg and runs two SEPARATE
// Run() calls against the same session — not two messages within one call —
// since cross-Run()-boundary persistence is the actual behavior this module
// is about, confirmed live in Phase 1's probe.
func assertMemoryRemembersName(t *testing.T, cfg llm.Config) {
	t.Helper()

	llmModel, _, err := llm.BuildModel(t.Context(), cfg)
	if err != nil {
		t.Fatalf("BuildModel() error = %v", err)
	}

	memAgent, err := BuildRootAgent(llmModel)
	if err != nil {
		t.Fatalf("BuildRootAgent() error = %v", err)
	}

	r, err := runner.NewInMemory("memory_test_app", memAgent)
	if err != nil {
		t.Fatalf("building runner: %v", err)
	}

	const sessionID = "test_session"

	if _, _, err := askMemoryAgent(t.Context(), r, sessionID, "Hi, I'm Mario.", "store_name"); err != nil {
		if strings.Contains(err.Error(), "RESOURCE_EXHAUSTED") {
			t.Skipf("skipping: Gemini free-tier quota exhausted for today (%v) — this is an external rate limit, not a code defect; re-run once it resets", err)
		}
		t.Fatalf("askMemoryAgent(turn 1) error = %v", err)
	}

	answer, toolResponse, err := askMemoryAgent(t.Context(), r, sessionID, "What is my name?", "recall_name")
	if err != nil && strings.Contains(err.Error(), "RESOURCE_EXHAUSTED") {
		t.Skipf("skipping: Gemini free-tier quota exhausted for today (%v) — this is an external rate limit, not a code defect; re-run once it resets", err)
	}
	if err != nil {
		t.Fatalf("askMemoryAgent(turn 2) error = %v", err)
	}
	if answer == "" {
		t.Fatal("agent returned no answer for turn 2")
	}

	// The structural proof: recall_name's own FunctionResponse must carry
	// "Mario" — not just the final answer's text, which runner.Run's
	// full-history replay could produce even if recall_name were broken.
	if toolResponse == nil {
		t.Fatalf("no FunctionResponse from recall_name was observed — turn 2's answer (%q) may have come from chat history, not real state access", answer)
	}
	if got, _ := toolResponse["name"].(string); got != "Mario" {
		t.Errorf("recall_name's FunctionResponse[\"name\"] = %v, want %q", toolResponse["name"], "Mario")
	}
	if !strings.Contains(answer, "Mario") {
		t.Errorf("turn 2 answer = %q, want it to mention Mario", answer)
	}
}
