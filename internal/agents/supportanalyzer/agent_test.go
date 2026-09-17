package supportanalyzer

import (
	"context"
	"encoding/json"
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

// structuredOutputUnavailableMsg is the literal text Ollama returns (via its
// OpenAI-compatible endpoint) when the configured model's quantization
// doesn't support JSON-schema-constrained decoding — confirmed live for some
// quantizations of this course's model family. The repo's default
// OLLAMA_MODEL (internal/infrastructure/llm.LoadConfig) is a GGUF
// quantization that supports it, so this only fires if OLLAMA_MODEL is
// overridden to one that doesn't — a defensive skip, not the expected path.
// There's no typed/sentinel error for this available through the SDK (the
// OpenAI client's own typed API error lives in an internal package), so this
// is a best-effort substring match on the server's own message, never
// asserted on as a real API contract.
const structuredOutputUnavailableMsg = "structured output is unavailable"

// testConfig and ollamaReachable are set once by TestMain, matching
// cmd/echo-agent's pattern: skip (not fail) when the local model backend
// isn't reachable.
var (
	testConfig      llm.Config
	ollamaReachable bool
)

func TestMain(m *testing.M) {
	testConfig = llm.LoadConfig()
	ollamaReachable = llm.OllamaReachable(testConfig, 2*time.Second)
	os.Exit(m.Run())
}

// analyzeTicket drives the real BuildRootAgent directly through runner,
// bypassing the launcher's console/web rendering. It returns the raw JSON
// string the SDK wrote to event.Actions.StateDelta[OutputKey] on the final
// response event.
func analyzeTicket(ctx context.Context, llmModel model.LLM, ticket string) (string, error) {
	analyzerAgent, err := BuildRootAgent(llmModel)
	if err != nil {
		return "", err
	}

	r, err := runner.NewInMemory("support_analyzer_app", analyzerAgent)
	if err != nil {
		return "", err
	}

	msg := genai.NewContentFromText(ticket, genai.RoleUser)
	for event, err := range r.Run(ctx, "test_user", "test_session", msg, agent.RunConfig{}) {
		if err != nil {
			return "", err
		}
		if !event.IsFinalResponse() {
			continue
		}
		if s, ok := event.Actions.StateDelta[OutputKey].(string); ok {
			return s, nil
		}
	}
	return "", nil
}

// TestSupportAnalyzer_ReturnsStructuredAnalysis_Ollama exercises the lab's
// own requirement against the local Ollama backend: the agent must return a
// JSON object with category, sentiment, and summary, saved into session
// state under "last_ticket_analysis". It skips when TEST_BACKEND excludes
// Ollama (see llm.SelectedTestBackend), or when Ollama isn't reachable, so
// CI or another machine doesn't break the suite.
func TestSupportAnalyzer_ReturnsStructuredAnalysis_Ollama(t *testing.T) {
	if !llm.SelectedTestBackend().IncludesOllama() {
		t.Skip("skipping: TEST_BACKEND excludes ollama")
	}
	if !ollamaReachable {
		t.Skip("skipping: local Ollama server (" + testConfig.OllamaBaseURL + ") is not reachable")
	}

	cfg := testConfig
	cfg.ModelType = llm.ModelTypeOllama
	assertSupportAnalyzerReturnsStructuredAnalysis(t, cfg)
}

// TestSupportAnalyzer_ReturnsStructuredAnalysis_Gemini is the same behavior,
// against the real Gemini backend. It skips when TEST_BACKEND excludes
// Gemini, or when GOOGLE_AI_STUDIO_API_KEY isn't set, since a Gemini
// credential isn't required for this local-first course.
func TestSupportAnalyzer_ReturnsStructuredAnalysis_Gemini(t *testing.T) {
	if !llm.SelectedTestBackend().IncludesGemini() {
		t.Skip("skipping: TEST_BACKEND excludes gemini")
	}
	if testConfig.GoogleAPIKey == "" {
		t.Skip("skipping: GOOGLE_AI_STUDIO_API_KEY is not set")
	}

	cfg := testConfig
	cfg.ModelType = llm.ModelTypeGemini
	assertSupportAnalyzerReturnsStructuredAnalysis(t, cfg)
}

// assertSupportAnalyzerReturnsStructuredAnalysis builds the model from cfg
// and runs the lab's structured-output requirement against it — shared by
// both the Ollama and Gemini variants, not pinning exact wording (an LLM's
// phrasing isn't reproducible), just structural validity and that sentiment
// tracks the ticket's tone.
func assertSupportAnalyzerReturnsStructuredAnalysis(t *testing.T, cfg llm.Config) {
	t.Helper()

	llmModel, _, err := llm.BuildModel(t.Context(), cfg)
	if err != nil {
		t.Fatalf("BuildModel() error = %v", err)
	}

	tests := []struct {
		name   string
		ticket string
	}{
		{
			name:   "angry technical complaint",
			ticket: "My screen is completely broken and I'm very angry about it!",
		},
		{
			name:   "neutral billing question",
			ticket: "Can you tell me when my next invoice is due?",
		},
	}

	results := make(map[string]SupportAnalysis, len(tests))

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			raw, err := analyzeTicket(t.Context(), llmModel, tt.ticket)
			if err != nil && strings.Contains(err.Error(), structuredOutputUnavailableMsg) {
				t.Skipf("skipping: %s does not support structured output (%v) — unset OLLAMA_MODEL or point it at a GGUF quantization like the repo default, qwen3.8:27b (see docs/module-04/README.md)", cfg.OllamaModel, err)
			}
			if err != nil {
				t.Fatalf("analyzeTicket(%q) error = %v", tt.ticket, err)
			}
			if raw == "" {
				t.Fatalf("analyzeTicket(%q) produced no state delta for %q", tt.ticket, OutputKey)
			}

			var got SupportAnalysis
			if err := json.Unmarshal([]byte(raw), &got); err != nil {
				t.Fatalf("json.Unmarshal(%q) error = %v", raw, err)
			}
			if got.Category == "" || got.Sentiment == "" || got.Summary == "" {
				t.Fatalf("SupportAnalysis = %+v, want all fields non-empty", got)
			}
			results[tt.name] = got
		})
	}

	if len(results) < len(tests) {
		// One or more subtests skipped (e.g. the configured model doesn't
		// support structured output) — nothing to compare.
		return
	}
	if angry, neutral := results["angry technical complaint"], results["neutral billing question"]; angry.Sentiment == neutral.Sentiment {
		t.Errorf("expected differing sentiment between an angry and a neutral ticket, both got %q", angry.Sentiment)
	}
}
