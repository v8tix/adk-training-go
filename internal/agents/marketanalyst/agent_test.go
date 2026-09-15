package marketanalyst

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

// testConfig and ollamaReachable are set once by TestMain, matching every
// other agent package's pattern.
var (
	testConfig      llm.Config
	ollamaReachable bool
)

func TestMain(m *testing.M) {
	testConfig = llm.LoadConfig()
	ollamaReachable = llm.OllamaReachable(testConfig, 2*time.Second)
	os.Exit(m.Run())
}

// askMarketAnalyst drives the real BuildRootAgent through a single Run()
// call. It returns the final text answer, plus wantToolCalled's actual
// FunctionResponse.Response map (nil if that tool wasn't called) — a real
// structural proof the tool fired, matching internal/agents/calculator's
// own askCalculator pattern.
func askMarketAnalyst(ctx context.Context, r *runner.Runner, question, wantToolCalled string) (answer string, toolResponse map[string]any, err error) {
	msg := genai.NewContentFromText(question, genai.RoleUser)
	for event, err := range r.Run(ctx, "test_user", "test_session", msg, agent.RunConfig{}) {
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

// TestMarketAnalyst_ConvertsCurrency_Ollama exercises this module's own
// finding against the local Ollama backend: a plain function-declared tool
// (no built-in tool involved) works here, confirmed live (module-11) — this
// agent needs no cloud fallback, matching modules 9-10's precedent.
func TestMarketAnalyst_ConvertsCurrency_Ollama(t *testing.T) {
	if !llm.SelectedTestBackend().IncludesOllama() {
		t.Skip("skipping: TEST_BACKEND excludes ollama")
	}
	if !ollamaReachable {
		t.Skip("skipping: local Ollama server (" + testConfig.OllamaBaseURL + ") is not reachable")
	}

	cfg := testConfig
	cfg.ModelType = llm.ModelTypeOllama
	assertMarketAnalystConverts(t, cfg)
}

// TestMarketAnalyst_ConvertsCurrency_Gemini is the same behavior against the
// real Gemini backend.
func TestMarketAnalyst_ConvertsCurrency_Gemini(t *testing.T) {
	if !llm.SelectedTestBackend().IncludesGemini() {
		t.Skip("skipping: TEST_BACKEND excludes gemini")
	}
	if testConfig.GoogleAPIKey == "" {
		t.Skip("skipping: GOOGLE_AI_STUDIO_API_KEY is not set")
	}

	cfg := testConfig
	cfg.ModelType = llm.ModelTypeGemini
	assertMarketAnalystConverts(t, cfg)
}

// assertMarketAnalystConverts builds the model from cfg and confirms the
// agent actually calls get_latest_rates against the real Frankfurter API.
// It asserts on the tool's real FunctionResponse structurally — the
// currency codes present, a positive numeric rate — never an exact pinned
// rate, since real exchange rates change daily and would make this test
// flaky by tomorrow.
func assertMarketAnalystConverts(t *testing.T, cfg llm.Config) {
	t.Helper()

	llmModel, _, err := llm.BuildModel(t.Context(), cfg)
	if err != nil {
		t.Fatalf("BuildModel() error = %v", err)
	}

	marketAgent, err := BuildRootAgent(llmModel)
	if err != nil {
		t.Fatalf("BuildRootAgent() error = %v", err)
	}

	r, err := runner.NewInMemory("market_analyst_test_app", marketAgent)
	if err != nil {
		t.Fatalf("building runner: %v", err)
	}

	answer, toolResponse, err := askMarketAnalyst(t.Context(), r, "Convert 100 USD to EUR.", "get_latest_rates")
	if err != nil && strings.Contains(err.Error(), "RESOURCE_EXHAUSTED") {
		t.Skipf("skipping: Gemini free-tier quota exhausted for today (%v) — this is an external rate limit, not a code defect; re-run once it resets", err)
	}
	if err != nil {
		t.Fatalf("askMarketAnalyst() error = %v", err)
	}
	if answer == "" {
		t.Fatal("agent returned no answer")
	}

	if toolResponse == nil {
		t.Fatalf("no FunctionResponse from get_latest_rates was observed — the agent likely answered %q from its own knowledge instead of calling the tool", answer)
	}
	if got, _ := toolResponse["base"].(string); got != "USD" {
		t.Errorf("get_latest_rates's FunctionResponse[\"base\"] = %v, want %q", toolResponse["base"], "USD")
	}
	rates, ok := toolResponse["rates"].(map[string]any)
	if !ok {
		t.Fatalf("get_latest_rates's FunctionResponse[\"rates\"] is %T, want map[string]any", toolResponse["rates"])
	}
	eurRate, ok := rates["EUR"].(float64)
	if !ok || eurRate <= 0 {
		t.Errorf("get_latest_rates's FunctionResponse[\"rates\"][\"EUR\"] = %v, want a positive number", rates["EUR"])
	}
}
