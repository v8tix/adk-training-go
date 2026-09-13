package main

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/v8tix/adk-training-go/internal/infrastructure/llm"
	"github.com/v8tix/adk-training-go/internal/infrastructure/prompts"
	"google.golang.org/adk/v2/agent"
	"google.golang.org/adk/v2/model"
	"google.golang.org/adk/v2/runner"
	"google.golang.org/genai"
)

// structuredOutputUnavailableMsg is the literal text Ollama returns (via its
// OpenAI-compatible endpoint) when the configured model's quantization
// doesn't support JSON-schema-constrained decoding — confirmed live for this
// machine's MLX presets (nvfp4/mxfp8). The repo's default OLLAMA_MODEL
// (internal/infrastructure/llm.LoadConfig) is a GGUF quantization that
// supports it, so this only fires if OLLAMA_MODEL is overridden back to an
// MLX preset — a defensive skip, not the expected path. There's no
// typed/sentinel error for this available through the SDK (the OpenAI
// client's own typed API error lives in an internal package), so this is a
// best-effort substring match on the server's own message, never asserted
// on as a real API contract.
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

// analyzeTicket drives the real buildRootAgent (from main.go) directly
// through runner, bypassing the launcher's console/web rendering. It
// returns the raw JSON string the SDK wrote to
// event.Actions.StateDelta[outputKey] on the final response event.
func analyzeTicket(ctx context.Context, llmModel model.LLM, ticket string) (string, error) {
	instruction, err := prompts.Get(promptNamespace + "/support_analyzer_instruction")
	if err != nil {
		return "", err
	}

	analyzerAgent, err := buildRootAgent(llmModel, instruction)
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
		if s, ok := event.Actions.StateDelta[outputKey].(string); ok {
			return s, nil
		}
	}
	return "", nil
}

// TestSupportAnalyzer_ReturnsStructuredAnalysis exercises the lab's own
// requirement: the agent must return a JSON object with category, sentiment,
// and summary, saved into session state under "last_ticket_analysis" — not
// pinning exact wording (an LLM's phrasing isn't reproducible), just
// structural validity and that sentiment tracks the ticket's tone.
func TestSupportAnalyzer_ReturnsStructuredAnalysis(t *testing.T) {
	if !ollamaReachable {
		t.Skip("skipping: local Ollama server (" + testConfig.OllamaBaseURL + ") is not reachable")
	}

	llmModel, _, err := llm.BuildModel(t.Context(), testConfig)
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
				t.Skipf("skipping: %s does not support structured output (%v) — unset OLLAMA_MODEL or point it at a GGUF quantization like the repo default, qwen3.8:27b (see docs/module-4/README.md)", testConfig.OllamaModel, err)
			}
			if err != nil {
				t.Fatalf("analyzeTicket(%q) error = %v", tt.ticket, err)
			}
			if raw == "" {
				t.Fatalf("analyzeTicket(%q) produced no state delta for %q", tt.ticket, outputKey)
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
