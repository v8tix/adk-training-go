package calculator

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/v8tix/adk-training-go/internal/infrastructure/llm"
	"google.golang.org/adk/v2/agent"
	"google.golang.org/adk/v2/model"
	"google.golang.org/adk/v2/runner"
	"google.golang.org/genai"
)

// This file is module-24's own, wholly additive contribution — nothing in
// agent.go, tools.go, agent_test.go, or tools_test.go changes. Python's
// module-24 lab evaluates the ADK's declarative evaluation framework
// (EvalSet/.evalset.json files, adk eval, metrics like
// tool_trajectory_avg_score/response_match_score) against this same
// Calculator agent from Module 9. That framework has no Go implementation
// at all — confirmed directly in the pinned SDK's own
// server/adkrest/internal/routers/eval.go doc comment: "ADK Go has no
// evaluation implementation... Use adk-python for eval workflows." Every
// eval REST endpoint the Dev UI calls is wired up but deliberately returns
// 501.
//
// What follows is NOT a port of that framework — it's the Go-idiomatic
// alternative this repo has actually used since Module 9's own
// agent_test.go: hand-written, structural assertions on the real events an
// agent produces. agent_test.go's own askCalculator already checks that
// one named tool was called with the right result; this file extends that
// idea to the fuller concept Python's richer examples describe — an
// ordered, multi-step tool-call trajectory — which nothing in this repo
// checks yet.

// expectedToolCall and goldenPathCase are this module's own lightweight,
// hand-written analogue of Python's EvalCase concept — a recorded
// "golden path": the user's question, the expected ordered tool
// trajectory, and a substring the final answer must contain. This is a
// plain Go struct compared with plain Go equality checks, not a port of
// the ADK's EvalSet/.evalset.json file format or its scored metrics.
type expectedToolCall struct {
	tool string
	args map[string]any
}

type goldenPathCase struct {
	name                 string
	question             string
	wantTrajectory       []expectedToolCall
	wantResponseContains string
}

type observedToolCall struct {
	tool string
	args map[string]any
}

// goldenPathCases includes a single-tool-call case as a baseline sanity
// check, and a genuine multi-step trajectory: multiply's own arguments
// (15, the sum of 10+5) depend on add's own result, so an agent that
// called these two tools out of order, or with the wrong arguments, would
// be caught here — this is a real dependency, not two independent calls
// that happen to look ordered.
var goldenPathCases = []goldenPathCase{
	{
		name:                 "single tool call",
		question:             "What is 42 + 118?",
		wantTrajectory:       []expectedToolCall{{tool: "add", args: map[string]any{"a": 42, "b": 118}}},
		wantResponseContains: "160",
	},
	{
		name:     "multi-step trajectory: add then multiply, in order",
		question: "First add 10 and 5. Then multiply that result by 2. What is the final answer?",
		wantTrajectory: []expectedToolCall{
			{tool: "add", args: map[string]any{"a": 10, "b": 5}},
			{tool: "multiply", args: map[string]any{"a": 15, "b": 2}},
		},
		wantResponseContains: "30",
	},
}

// runGoldenPath drives BuildRootAgent through a single Run() call and
// records every FunctionCall event in the order it occurs — the real
// "trajectory" Python's tool_trajectory_avg_score metric would score,
// captured here as a plain ordered slice instead of a declarative,
// LLM-scored comparison.
func runGoldenPath(ctx context.Context, llmModel model.LLM, question string) (answer string, trajectory []observedToolCall, err error) {
	calcAgent, err := BuildRootAgent(llmModel)
	if err != nil {
		return "", nil, err
	}

	r, err := runner.NewInMemory("golden_path_test_app", calcAgent)
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
			if p.FunctionCall != nil {
				trajectory = append(trajectory, observedToolCall{tool: p.FunctionCall.Name, args: p.FunctionCall.Args})
			}
			if event.IsFinalResponse() && p.Text != "" && !p.Thought {
				answer = p.Text
			}
		}
	}
	return answer, trajectory, nil
}

// TestGoldenPath_Calculator_Ollama exercises every case in goldenPathCases
// against the local Ollama backend.
func TestGoldenPath_Calculator_Ollama(t *testing.T) {
	if !llm.SelectedTestBackend().IncludesOllama() {
		t.Skip("skipping: TEST_BACKEND excludes ollama")
	}
	if !ollamaReachable {
		t.Skip("skipping: local Ollama server (" + testConfig.OllamaBaseURL + ") is not reachable")
	}

	cfg := testConfig
	cfg.ModelType = llm.ModelTypeOllama
	for _, tc := range goldenPathCases {
		t.Run(tc.name, func(t *testing.T) {
			assertGoldenPath(t, cfg, tc)
		})
	}
}

// TestGoldenPath_Calculator_Gemini is the same behavior against the real
// Gemini backend.
func TestGoldenPath_Calculator_Gemini(t *testing.T) {
	if !llm.SelectedTestBackend().IncludesGemini() {
		t.Skip("skipping: TEST_BACKEND excludes gemini")
	}
	if testConfig.GoogleAPIKey == "" {
		t.Skip("skipping: GOOGLE_AI_STUDIO_API_KEY is not set")
	}

	cfg := testConfig
	cfg.ModelType = llm.ModelTypeGemini
	for _, tc := range goldenPathCases {
		t.Run(tc.name, func(t *testing.T) {
			assertGoldenPath(t, cfg, tc)
		})
	}
}

// assertGoldenPath builds the model from cfg, runs tc's question through
// runGoldenPath, and checks the observed trajectory against tc's expected
// one — exact tool names, in order, with the right arguments — plus a
// fuzzy (substring, not exact-equality) check on the final answer, mirroring
// what response_match_score's own purpose is without its LLM-judge
// machinery.
func assertGoldenPath(t *testing.T, cfg llm.Config, tc goldenPathCase) {
	t.Helper()

	llmModel, _, err := llm.BuildModel(t.Context(), cfg)
	if err != nil {
		t.Fatalf("BuildModel() error = %v", err)
	}

	answer, trajectory, err := runGoldenPath(t.Context(), llmModel, tc.question)
	if err != nil && strings.Contains(err.Error(), "RESOURCE_EXHAUSTED") {
		t.Skipf("skipping: Gemini free-tier quota exhausted for today (%v) — this is an external rate limit, not a code defect; re-run once it resets", err)
	}
	if err != nil {
		t.Fatalf("runGoldenPath() error = %v", err)
	}

	if len(trajectory) != len(tc.wantTrajectory) {
		t.Fatalf("trajectory = %+v (len %d), want %+v (len %d)", trajectory, len(trajectory), tc.wantTrajectory, len(tc.wantTrajectory))
	}
	for i, want := range tc.wantTrajectory {
		got := trajectory[i]
		if got.tool != want.tool {
			t.Errorf("trajectory[%d].tool = %q, want %q — the tool call sequence is out of order or wrong", i, got.tool, want.tool)
		}
		// Only the keys named in want.args are checked — a subset match, not
		// a full deep-equal, so an extra argument the model happened to pass
		// would not fail this check.
		for argName, wantVal := range want.args {
			gotVal, ok := got.args[argName]
			if !ok {
				t.Errorf("trajectory[%d] (%s) is missing argument %q", i, got.tool, argName)
				continue
			}
			// gotVal decodes from JSON as float64 (FunctionCall.Args is
			// map[string]any from the wire), while wantVal above is written
			// as a plain int literal — fmt.Sprint renders both the same way
			// for integral values (float64(15) and int(15) both print "15"),
			// so this comparison works today but would need a numeric-aware
			// comparison instead of string formatting if a case ever
			// compares a non-integral float.
			if fmt.Sprint(gotVal) != fmt.Sprint(wantVal) {
				t.Errorf("trajectory[%d] (%s) argument %q = %v, want %v", i, got.tool, argName, gotVal, wantVal)
			}
		}
	}

	if answer == "" {
		t.Fatal("agent returned no final answer")
	}
	if !strings.Contains(answer, tc.wantResponseContains) {
		t.Errorf("answer = %q, want it to contain %q", answer, tc.wantResponseContains)
	}
}
