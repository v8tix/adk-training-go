package marketrouter

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

func skipOnQuota(t *testing.T, err error) {
	t.Helper()
	if err != nil && strings.Contains(err.Error(), "RESOURCE_EXHAUSTED") {
		t.Skipf("skipping: Gemini free-tier quota exhausted for today (%v) — external rate limit, not a code defect", err)
	}
}

func skipIfNoOllama(t *testing.T) llm.Config {
	t.Helper()
	if !llm.SelectedTestBackend().IncludesOllama() {
		t.Skip("skipping: TEST_BACKEND excludes ollama")
	}
	if !ollamaReachable {
		t.Skip("skipping: local Ollama server (" + testConfig.OllamaBaseURL + ") is not reachable")
	}
	cfg := testConfig
	cfg.ModelType = llm.ModelTypeOllama
	return cfg
}

func skipIfNoGemini(t *testing.T) llm.Config {
	t.Helper()
	if !llm.SelectedTestBackend().IncludesGemini() {
		t.Skip("skipping: TEST_BACKEND excludes gemini")
	}
	if testConfig.GoogleAPIKey == "" {
		t.Skip("skipping: GOOGLE_AI_STUDIO_API_KEY is not set")
	}
	cfg := testConfig
	cfg.ModelType = llm.ModelTypeGemini
	return cfg
}

// askMarketRouter drives the real BuildRootAgent through runner for a
// single user message, returning the final response text so the caller can
// confirm which specialist actually answered.
func askMarketRouter(ctx context.Context, llmModel model.LLM, appName, message string) (string, error) {
	rootAgent, err := BuildRootAgent(llmModel)
	if err != nil {
		return "", err
	}

	r, err := runner.NewInMemory(appName, rootAgent)
	if err != nil {
		return "", err
	}

	msg := genai.NewContentFromText(message, genai.RoleUser)
	var finalText string
	for event, runErr := range r.Run(ctx, "test_user", "test_session", msg, agent.RunConfig{}) {
		if runErr != nil {
			return "", runErr
		}
		if event.Content == nil {
			continue
		}
		for _, p := range event.Content.Parts {
			if event.IsFinalResponse() && p.Text != "" && !p.Thought {
				finalText = p.Text
			}
		}
	}
	return finalText, nil
}

// TestMarketRouter_RoutesCorrectly_Ollama confirms two different currency
// requests each reach their own, distinct specialist — proof the routing
// decision genuinely follows the classifier's live output, not a
// coincidence — through local Ollama.
func TestMarketRouter_RoutesCorrectly_Ollama(t *testing.T) {
	assertRoutesCorrectly(t, skipIfNoOllama(t))
}

// TestMarketRouter_RoutesCorrectly_Gemini is the same behavior against real
// Gemini.
func TestMarketRouter_RoutesCorrectly_Gemini(t *testing.T) {
	assertRoutesCorrectly(t, skipIfNoGemini(t))
}

func assertRoutesCorrectly(t *testing.T, cfg llm.Config) {
	t.Helper()

	llmModel, _, err := llm.BuildModel(t.Context(), cfg)
	if err != nil {
		t.Fatalf("BuildModel() error = %v", err)
	}

	allMarkers := []string{"USD Analysis:", "EUR Analysis:", "GBP Analysis:"}

	cases := []struct {
		message    string
		wantMarker string
	}{
		{message: "I'd like an analysis in British Pounds please.", wantMarker: "GBP Analysis:"},
		{message: "Give me an analysis in US Dollars.", wantMarker: "USD Analysis:"},
		{message: "Give me an analysis in Euros please.", wantMarker: "EUR Analysis:"},
	}

	for _, tc := range cases {
		t.Run(tc.wantMarker, func(t *testing.T) {
			text, err := askMarketRouter(t.Context(), llmModel, "market_router_test_app", tc.message)
			skipOnQuota(t, err)
			if err != nil {
				t.Fatalf("askMarketRouter() error = %v", err)
			}
			if text == "" {
				t.Fatal("no final response text observed")
			}
			if !strings.Contains(text, tc.wantMarker) {
				t.Errorf("response %q does not contain expected marker %q — wrong specialist may have handled the request", text, tc.wantMarker)
			}
			for _, marker := range allMarkers {
				if marker != tc.wantMarker && strings.Contains(text, marker) {
					t.Errorf("response %q unexpectedly contains marker %q from a different specialist", text, marker)
				}
			}
		})
	}
}
