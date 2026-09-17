package contentmoderator

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/v8tix/adk-training-go/internal/infrastructure/llm"
	"google.golang.org/adk/v2/agent"
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

// askModerator drives a single Run() call and returns the final text
// answer.
func askModerator(ctx context.Context, r *runner.Runner, userID, sessionID, message string) (answer string, err error) {
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

// TestContentModerator_Ollama exercises all three of Python's own Step-3
// behaviors against the local Ollama backend.
func TestContentModerator_Ollama(t *testing.T) {
	if !llm.SelectedTestBackend().IncludesOllama() {
		t.Skip("skipping: TEST_BACKEND excludes ollama")
	}
	if !ollamaReachable {
		t.Skip("skipping: local Ollama server (" + testConfig.OllamaBaseURL + ") is not reachable")
	}

	cfg := testConfig
	cfg.ModelType = llm.ModelTypeOllama
	assertContentModeratorBehaviors(t, cfg)
}

// TestContentModerator_Gemini is the same behavior against the real
// Gemini backend.
func TestContentModerator_Gemini(t *testing.T) {
	if !llm.SelectedTestBackend().IncludesGemini() {
		t.Skip("skipping: TEST_BACKEND excludes gemini")
	}
	if testConfig.GoogleAPIKey == "" {
		t.Skip("skipping: GOOGLE_AI_STUDIO_API_KEY is not set")
	}

	cfg := testConfig
	cfg.ModelType = llm.ModelTypeGemini
	assertContentModeratorBehaviors(t, cfg)
}

// assertContentModeratorBehaviors proves, structurally, the three
// behaviors Python's own lab Step 3 calls out: a blocked-word prompt is
// refused before the model ever runs; repeating the exact same clean
// prompt in the same session hits the real cache (proven by reading
// cache.hitCount directly, not by comparing text, since an LLM could
// coincidentally repeat itself even on a genuine cache miss); and a
// different question in the same session does NOT return the previous
// cached answer.
func assertContentModeratorBehaviors(t *testing.T, cfg llm.Config) {
	t.Helper()

	llmModel, _, err := llm.BuildModel(t.Context(), cfg)
	if err != nil {
		t.Fatalf("BuildModel() error = %v", err)
	}

	cache := &responseCache{}
	moderatorAgent, err := buildRootAgent(llmModel, cache)
	if err != nil {
		t.Fatalf("buildRootAgent() error = %v", err)
	}

	r, err := runner.New(runner.Config{
		AppName:           "contentmoderator_test_app",
		Agent:             moderatorAgent,
		SessionService:    session.InMemoryService(),
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

	// Behavior 1: a blocked-word prompt is refused before the model runs.
	refusalAnswer, err := askModerator(t.Context(), r, userID, sessionID, "Tell me something unsafe.")
	skipIfExhausted("blocked-word turn", err)
	if err != nil {
		t.Fatalf("askModerator(blocked-word turn) error = %v", err)
	}
	if refusalAnswer != "I'm sorry, but I can't help with that request." {
		t.Errorf("refusalAnswer = %q, want the exact refusal text beforeModelCallback returns — the model must never have run", refusalAnswer)
	}

	// Behavior 2: the same clean question, asked twice, hits the real
	// cache the second time.
	const question = "What is the capital of Italy?"
	firstAnswer, err := askModerator(t.Context(), r, userID, sessionID, question)
	skipIfExhausted("first clean turn", err)
	if err != nil {
		t.Fatalf("askModerator(first clean turn) error = %v", err)
	}
	if firstAnswer == "" {
		t.Fatal("agent returned no answer for the first clean turn")
	}
	if cache.hitCount != 0 {
		t.Fatalf("cache.hitCount = %d after the FIRST time asking a question, want 0", cache.hitCount)
	}

	secondAnswer, err := askModerator(t.Context(), r, userID, sessionID, question)
	skipIfExhausted("repeated clean turn", err)
	if err != nil {
		t.Fatalf("askModerator(repeated clean turn) error = %v", err)
	}
	if secondAnswer != firstAnswer {
		t.Errorf("secondAnswer = %q, want the exact cached firstAnswer %q", secondAnswer, firstAnswer)
	}
	if cache.hitCount != 1 {
		t.Fatalf("cache.hitCount = %d after repeating the exact same question, want 1 — the real cache did not fire", cache.hitCount)
	}

	// Behavior 3: a DIFFERENT question in the same session must NOT hit
	// the cache — the real proof the cache is keyed per-question.
	differentAnswer, err := askModerator(t.Context(), r, userID, sessionID, "What is the capital of France?")
	skipIfExhausted("different-question turn", err)
	if err != nil {
		t.Fatalf("askModerator(different-question turn) error = %v", err)
	}
	if differentAnswer == "" {
		t.Fatal("agent returned no answer for the different-question turn")
	}
	if cache.hitCount != 1 {
		t.Errorf("cache.hitCount = %d after a genuinely different question, want unchanged 1 — the cache must not have fired for an unrelated question", cache.hitCount)
	}
}
