package calculator

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/v8tix/adk-training-go/internal/infrastructure/llm"
	"google.golang.org/adk/v2/agent"
	"google.golang.org/adk/v2/model"
	"google.golang.org/adk/v2/runner"
	"google.golang.org/genai"
)

// testConfig and ollamaReachable are set once by TestMain, matching
// internal/agents/supportanalyzer's pattern: skip (not fail) when the local
// model backend isn't reachable.
var (
	testConfig      llm.Config
	ollamaReachable bool
)

func TestMain(m *testing.M) {
	testConfig = llm.LoadConfig()
	ollamaReachable = llm.OllamaReachable(testConfig, 2*time.Second)
	os.Exit(m.Run())
}

// askCalculator drives the real BuildRootAgent directly through runner,
// bypassing the launcher's console/web rendering. It returns the agent's
// final text answer, plus the named tool's actual FunctionResponse.Response
// map, if that tool was really called during the conversation (nil
// otherwise) — a structural proof the tool fired, not just that the final
// text happens to contain the right number, which an LLM could produce from
// its own arithmetic on an easy sum without ever calling the tool.
func askCalculator(ctx context.Context, llmModel model.LLM, question, wantToolCalled string) (answer string, toolResponse map[string]any, err error) {
	calcAgent, err := BuildRootAgent(llmModel)
	if err != nil {
		return "", nil, err
	}

	r, err := runner.NewInMemory("calculator_test_app", calcAgent)
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

// TestCalculator_Adds_Ollama exercises this module's own finding against the
// local Ollama backend: custom function tools work here, confirmed live
// (module-9) — unlike internal/agents/visualcatalog and
// internal/agents/researcher, this agent needs no cloud fallback. It skips
// when TEST_BACKEND excludes Ollama, or when Ollama isn't reachable.
func TestCalculator_Adds_Ollama(t *testing.T) {
	if !llm.SelectedTestBackend().IncludesOllama() {
		t.Skip("skipping: TEST_BACKEND excludes ollama")
	}
	if !ollamaReachable {
		t.Skip("skipping: local Ollama server (" + testConfig.OllamaBaseURL + ") is not reachable")
	}

	cfg := testConfig
	cfg.ModelType = llm.ModelTypeOllama
	assertCalculatorAdds(t, cfg)
}

// TestCalculator_Adds_Gemini is the same behavior against the real Gemini
// backend — the first tools module (after modules 7 and 8's cloud-only
// agents) where both backends genuinely work. It skips when TEST_BACKEND
// excludes Gemini, or when GOOGLE_AI_STUDIO_API_KEY isn't set.
func TestCalculator_Adds_Gemini(t *testing.T) {
	if !llm.SelectedTestBackend().IncludesGemini() {
		t.Skip("skipping: TEST_BACKEND excludes gemini")
	}
	if testConfig.GoogleAPIKey == "" {
		t.Skip("skipping: GOOGLE_AI_STUDIO_API_KEY is not set")
	}

	cfg := testConfig
	cfg.ModelType = llm.ModelTypeGemini
	assertCalculatorAdds(t, cfg)
}

// assertCalculatorAdds builds the model from cfg and confirms the agent
// actually invokes the add tool rather than guessing. 42 + 118 is easy
// enough arithmetic that a "thinking" model could produce the right final
// text without ever calling the tool, so the real, structural proof is the
// add tool's own FunctionResponse — a real function call actually happened,
// with the real computed result inside it — checked in addition to (not
// instead of) the final text containing the right number.
func assertCalculatorAdds(t *testing.T, cfg llm.Config) {
	t.Helper()

	const opA, opB = 42, 118
	sum := opA + opB

	llmModel, _, err := llm.BuildModel(t.Context(), cfg)
	if err != nil {
		t.Fatalf("BuildModel() error = %v", err)
	}

	question := fmt.Sprintf("What is %d + %d?", opA, opB)
	answer, toolResponse, err := askCalculator(t.Context(), llmModel, question, "add")
	if err != nil && strings.Contains(err.Error(), "RESOURCE_EXHAUSTED") {
		t.Skipf("skipping: Gemini free-tier quota exhausted for today (%v) — this is an external rate limit, not a code defect; re-run once it resets", err)
	}
	if err != nil {
		t.Fatalf("askCalculator() error = %v", err)
	}
	if answer == "" {
		t.Fatal("agent returned no answer")
	}

	if toolResponse == nil {
		t.Fatalf("no FunctionResponse from the add tool was observed — the agent likely computed %q itself instead of calling the tool", answer)
	}
	if got, _ := toolResponse["result"].(float64); got != float64(sum) {
		t.Errorf("add tool's FunctionResponse[\"result\"] = %v, want %v", toolResponse["result"], sum)
	}
	if !strings.Contains(answer, strconv.Itoa(sum)) {
		t.Errorf("answer = %q, want it to contain the real computed sum %d", answer, sum)
	}
}
