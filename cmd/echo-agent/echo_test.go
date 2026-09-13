package main

import (
	"context"
	"os"
	"slices"
	"testing"
	"time"

	"github.com/v8tix/adk-training-go/internal/infrastructure/llm"
	"github.com/v8tix/adk-training-go/internal/infrastructure/prompts"
	"google.golang.org/adk/v2/agent"
	"google.golang.org/adk/v2/model"
	"google.golang.org/adk/v2/runner"
	"google.golang.org/genai"
)

// testConfig and ollamaReachable are set once by TestMain, matching
// cmd/verify-setup's pattern: skip (not fail) when the local model backend
// isn't reachable, so CI or another machine doesn't break the suite.
var (
	testConfig      llm.Config
	ollamaReachable bool
)

func TestMain(m *testing.M) {
	testConfig = llm.LoadConfig()
	ollamaReachable = llm.OllamaReachable(testConfig, 2*time.Second)
	os.Exit(m.Run())
}

// runEcho drives the real buildRootAgent (from main.go) directly through
// runner, bypassing the launcher's console/web rendering entirely — that
// rendering concatenates a thinking model's reasoning with its answer
// (confirmed manually), which would break an exact-match assertion.
// firstAnswerText's Thought-filtering (from cmd/verify-setup's pattern) gets
// the real answer only. The instruction comes from the same prompts.Get call
// main() makes (registered by main.go's own init()), not a hardcoded copy —
// if the prompt file changes, this test exercises the new text automatically.
func runEcho(ctx context.Context, llmModel model.LLM, input string) (string, error) {
	instruction, err := prompts.Get(promptNamespace + "/echo_instruction")
	if err != nil {
		return "", err
	}

	echoAgent, err := buildRootAgent(llmModel, instruction)
	if err != nil {
		return "", err
	}

	r, err := runner.NewInMemory("echo_app", echoAgent)
	if err != nil {
		return "", err
	}

	msg := genai.NewContentFromText(input, genai.RoleUser)
	for event, err := range r.Run(ctx, "test_user", "test_session", msg, agent.RunConfig{}) {
		if err != nil {
			return "", err
		}
		if !event.IsFinalResponse() || event.Content == nil {
			continue
		}
		if text, ok := firstAnswerText(event.Content.Parts); ok {
			return text, nil
		}
	}
	return "", nil
}

// firstAnswerText returns the text of the first non-empty part that isn't a
// reasoning trace — same logic as cmd/verify-setup's helper of the same name.
func firstAnswerText(parts []*genai.Part) (string, bool) {
	i := slices.IndexFunc(parts, func(p *genai.Part) bool { return !p.Thought && p.Text != "" })
	if i < 0 {
		return "", false
	}
	return parts[i].Text, true
}

// TestEchoAgent_RepeatsInputExactly_Ollama exercises the lab's own "Expected
// Behavior" table against the local Ollama backend: the agent must repeat
// each input verbatim, never answer. It skips when TEST_BACKEND excludes
// Ollama (see llm.SelectedTestBackend), or when Ollama isn't reachable, so
// CI or another machine doesn't break the suite.
func TestEchoAgent_RepeatsInputExactly_Ollama(t *testing.T) {
	if !llm.SelectedTestBackend().IncludesOllama() {
		t.Skip("skipping: TEST_BACKEND excludes ollama")
	}
	if !ollamaReachable {
		t.Skip("skipping: local Ollama server (" + testConfig.OllamaBaseURL + ") is not reachable")
	}

	cfg := testConfig
	cfg.ModelType = llm.ModelTypeOllama
	assertEchoAgentRepeatsInputExactly(t, cfg)
}

// TestEchoAgent_RepeatsInputExactly_Gemini is the same behavior, against the
// real Gemini backend. It skips when TEST_BACKEND excludes Gemini, or when
// GOOGLE_AI_STUDIO_API_KEY isn't set, since a Gemini credential isn't
// required for this local-first course.
func TestEchoAgent_RepeatsInputExactly_Gemini(t *testing.T) {
	if !llm.SelectedTestBackend().IncludesGemini() {
		t.Skip("skipping: TEST_BACKEND excludes gemini")
	}
	if testConfig.GoogleAPIKey == "" {
		t.Skip("skipping: GOOGLE_AI_STUDIO_API_KEY is not set")
	}

	cfg := testConfig
	cfg.ModelType = llm.ModelTypeGemini
	assertEchoAgentRepeatsInputExactly(t, cfg)
}

// assertEchoAgentRepeatsInputExactly builds the model from cfg and runs the
// lab's three Expected Behavior cases against it — shared by both the Ollama
// and Gemini variants so the assertions can never drift between them.
func assertEchoAgentRepeatsInputExactly(t *testing.T, cfg llm.Config) {
	t.Helper()

	llmModel, _, err := llm.BuildModel(t.Context(), cfg)
	if err != nil {
		t.Fatalf("BuildModel() error = %v", err)
	}

	tests := []string{
		"Hello!",
		"What is the capital of France?",
		"12345",
	}

	for _, input := range tests {
		t.Run(input, func(t *testing.T) {
			got, err := runEcho(t.Context(), llmModel, input)
			if err != nil {
				t.Fatalf("runEcho(%q) error = %v", input, err)
			}
			if got != input {
				t.Fatalf("runEcho(%q) = %q, want the exact input echoed back", input, got)
			}
		})
	}
}
