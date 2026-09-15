package factfinder

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/v8tix/adk-training-go/internal/infrastructure/llm"
	"google.golang.org/adk/v2/agent"
	"google.golang.org/adk/v2/model"
	"google.golang.org/adk/v2/runner"
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

// TestBuildRootAgent_Constructs is a structural regression guard,
// independent of any live call.
func TestBuildRootAgent_Constructs(t *testing.T) {
	cfg := testConfig
	cfg.ModelType = llm.ModelTypeGemini
	cfg.GoogleAPIKey = "unused-for-construction"
	llmModel, _, err := llm.BuildModel(t.Context(), cfg)
	if err != nil {
		t.Fatalf("BuildModel() error = %v", err)
	}

	a, err := BuildRootAgent(llmModel)
	if err != nil {
		t.Fatalf("BuildRootAgent() error = %v", err)
	}
	if a == nil {
		t.Fatal("BuildRootAgent() returned a nil agent")
	}
}

// askFactFinder drives the real BuildRootAgent directly through runner,
// returning the agent's final text answer plus lookup_wikipedia's own
// FunctionResponse.Response map, if the tool was really called (nil
// otherwise) — a structural proof the tool fired, matching calculator's own
// askCalculator pattern (module-9).
func askFactFinder(ctx context.Context, llmModel model.LLM, question string) (answer string, toolResponse map[string]any, err error) {
	ffAgent, err := BuildRootAgent(llmModel)
	if err != nil {
		return "", nil, err
	}

	r, err := runner.NewInMemory("factfinder_test_app", ffAgent)
	if err != nil {
		return "", nil, err
	}

	msg := genai.NewContentFromText(question, genai.RoleUser)
	for event, err := range r.Run(ctx, "test_user", "test_session", msg, agent.RunConfig{}) {
		if err != nil {
			return "", nil, err
		}
		if event.Content == nil {
			continue
		}
		for _, p := range event.Content.Parts {
			if p.FunctionResponse != nil && p.FunctionResponse.Name == "lookup_wikipedia" {
				toolResponse = p.FunctionResponse.Response
			}
			if event.IsFinalResponse() && p.Text != "" && !p.Thought {
				answer = p.Text
			}
		}
	}
	return answer, toolResponse, nil
}

func skipOnQuota(t *testing.T, err error) {
	t.Helper()
	if err != nil && strings.Contains(err.Error(), "RESOURCE_EXHAUSTED") {
		t.Skipf("skipping: Gemini free-tier quota exhausted for today (%v) — external rate limit, not a code defect", err)
	}
}

// TestFactFinder_LooksUpWikipedia_Ollama confirms the plain custom function
// tool works through local Ollama — no cloud fallback needed, matching
// module-9's own calculator precedent for a tool with no built-in
// involved.
func TestFactFinder_LooksUpWikipedia_Ollama(t *testing.T) {
	if !llm.SelectedTestBackend().IncludesOllama() {
		t.Skip("skipping: TEST_BACKEND excludes ollama")
	}
	if !ollamaReachable {
		t.Skip("skipping: local Ollama server (" + testConfig.OllamaBaseURL + ") is not reachable")
	}
	cfg := testConfig
	cfg.ModelType = llm.ModelTypeOllama
	assertLooksUpWikipedia(t, cfg)
}

// TestFactFinder_LooksUpWikipedia_Gemini is the same behavior against real
// Gemini.
func TestFactFinder_LooksUpWikipedia_Gemini(t *testing.T) {
	if !llm.SelectedTestBackend().IncludesGemini() {
		t.Skip("skipping: TEST_BACKEND excludes gemini")
	}
	if testConfig.GoogleAPIKey == "" {
		t.Skip("skipping: GOOGLE_AI_STUDIO_API_KEY is not set")
	}
	cfg := testConfig
	cfg.ModelType = llm.ModelTypeGemini
	assertLooksUpWikipedia(t, cfg)
}

// assertLooksUpWikipedia confirms the agent actually invokes
// lookup_wikipedia and the final answer reflects a real, current summary —
// not the model's own guess, and not an exact pinned string, since
// Wikipedia content can change.
func assertLooksUpWikipedia(t *testing.T, cfg llm.Config) {
	t.Helper()

	llmModel, _, err := llm.BuildModel(t.Context(), cfg)
	if err != nil {
		t.Fatalf("BuildModel() error = %v", err)
	}

	answer, toolResponse, err := askFactFinder(t.Context(), llmModel, "Who was Marie Curie?")
	skipOnQuota(t, err)
	if err != nil {
		t.Fatalf("askFactFinder() error = %v", err)
	}
	if answer == "" {
		t.Fatal("agent returned no answer")
	}

	if toolResponse == nil {
		t.Fatalf("no FunctionResponse from lookup_wikipedia was observed — the agent likely answered from its own training data instead of calling the tool")
	}
	if status, _ := toolResponse["status"].(string); status != "success" {
		t.Fatalf("lookup_wikipedia status = %q, want %q (response: %v)", status, "success", toolResponse)
	}
	summary, _ := toolResponse["summary"].(string)
	if !strings.Contains(summary, "Curie") {
		t.Errorf("lookup_wikipedia summary = %q, want it to mention %q", summary, "Curie")
	}
}
