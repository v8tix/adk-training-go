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

// TestEchoAgent_RepeatsInputExactly exercises the lab's own "Expected
// Behavior" table: the agent must repeat each input verbatim, never answer.
func TestEchoAgent_RepeatsInputExactly(t *testing.T) {
	if !ollamaReachable {
		t.Skip("skipping: local Ollama server (" + testConfig.OllamaBaseURL + ") is not reachable")
	}

	llmModel, _, err := llm.BuildModel(t.Context(), testConfig)
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
