package researchassistant

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/v8tix/adk-training-go/internal/infrastructure/llm"
	"google.golang.org/adk/v2/agent"
	"google.golang.org/adk/v2/agent/llmagent"
	"google.golang.org/adk/v2/model"
	"google.golang.org/adk/v2/runner"
	"google.golang.org/adk/v2/tool"
	"google.golang.org/adk/v2/tool/geminitool"
	"google.golang.org/genai"
)

var testConfig llm.Config

func TestMain(m *testing.M) {
	testConfig = llm.LoadConfig()
	os.Exit(m.Run())
}

// askAgent drives builtAgent directly through runner, returning its final
// non-thought text answer, whether any FunctionResponse for wantToolCalled
// was observed (nil if not asked for or not called), and whether real
// grounding metadata (from google_search) was observed on any event.
func askAgent(ctx context.Context, builtAgent agent.Agent, appName, question, wantToolCalled string) (answer string, toolResponse map[string]any, sawGrounding bool, err error) {
	r, err := runner.NewInMemory(appName, builtAgent)
	if err != nil {
		return "", nil, false, err
	}

	msg := genai.NewContentFromText(question, genai.RoleUser)
	for event, runErr := range r.Run(ctx, "test_user", "test_session", msg, agent.RunConfig{}) {
		if runErr != nil {
			return "", nil, false, runErr
		}
		if event.GroundingMetadata != nil {
			sawGrounding = true
		}
		if event.Content == nil {
			continue
		}
		for _, p := range event.Content.Parts {
			if p.FunctionResponse != nil && wantToolCalled != "" && p.FunctionResponse.Name == wantToolCalled {
				toolResponse = p.FunctionResponse.Response
			}
			if event.IsFinalResponse() && p.Text != "" && !p.Thought {
				answer = p.Text
			}
		}
	}
	return answer, toolResponse, sawGrounding, nil
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

func skipOnQuota(t *testing.T, err error) {
	t.Helper()
	if err != nil && strings.Contains(err.Error(), "RESOURCE_EXHAUSTED") {
		t.Skipf("skipping: Gemini free-tier quota exhausted for today (%v) — external rate limit, not a code defect", err)
	}
}

func buildModel(t *testing.T, cfg llm.Config) model.LLM {
	t.Helper()
	llmModel, _, err := llm.BuildModel(t.Context(), cfg)
	if err != nil {
		t.Fatalf("BuildModel() error = %v", err)
	}
	return llmModel
}

// TestBuildAgent_Constructs confirms all three builders succeed without
// error, independent of any live call. agent.Agent exposes no way to
// inspect a built agent's actual tool composition from outside the
// package, so this deliberately claims only "constructs successfully" —
// the real structural proof that each agent carries the right tools (and
// only those) is the live behavioral tests below (e.g.
// TestResearchAgent_SearchesTheWeb_Gemini proves the search agent grounds
// but never calls a custom tool; TestFormatterAgent_NeverUsesSearch_Gemini
// proves the reverse; TestCombinedAgent_UsesSearchAndCustomTool_Gemini
// proves the combined agent does both).
func TestBuildAgent_Constructs(t *testing.T) {
	tests := []struct {
		name    string
		builder func(model.LLM) (agent.Agent, error)
	}{
		{"BuildResearchAgent", BuildResearchAgent},
		{"BuildFormatterAgent", BuildFormatterAgent},
		{"BuildCombinedAgent", BuildCombinedAgent},
	}

	cfg := testConfig
	cfg.ModelType = llm.ModelTypeGemini
	cfg.GoogleAPIKey = "unused-for-construction"
	llmModel := buildModel(t, cfg)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a, err := tt.builder(llmModel)
			if err != nil {
				t.Fatalf("%s() error = %v", tt.name, err)
			}
			if a == nil {
				t.Fatalf("%s() returned a nil agent", tt.name)
			}
		})
	}
}

// TestResearchAgent_SearchesTheWeb_Gemini confirms the search-only agent
// actually invokes google_search and returns grounded content — real
// GroundingMetadata, not the model's own training-data guess.
func TestResearchAgent_SearchesTheWeb_Gemini(t *testing.T) {
	cfg := skipIfNoGemini(t)
	llmModel := buildModel(t, cfg)

	a, err := BuildResearchAgent(llmModel)
	if err != nil {
		t.Fatalf("BuildResearchAgent() error = %v", err)
	}

	answer, _, sawGrounding, err := askAgent(t.Context(), a, "research_agent_test_app",
		"Research this topic: the most recent Nobel Prize in Physics winner.", "")
	skipOnQuota(t, err)
	if err != nil {
		t.Fatalf("askAgent() error = %v", err)
	}
	if answer == "" {
		t.Fatal("research agent returned no answer")
	}
	if !sawGrounding {
		t.Error("no GroundingMetadata observed — the agent likely answered from training data instead of searching")
	}
}

// TestFormatterAgent_NeverUsesSearch_Gemini confirms the formatter agent
// processes given findings through its two custom tools without ever
// touching google_search (it has no such tool to call).
func TestFormatterAgent_NeverUsesSearch_Gemini(t *testing.T) {
	cfg := skipIfNoGemini(t)
	llmModel := buildModel(t, cfg)

	a, err := BuildFormatterAgent(llmModel)
	if err != nil {
		t.Fatalf("BuildFormatterAgent() error = %v", err)
	}

	question := "Topic: solar power\n\nFindings: Solar panel efficiency has improved. Battery storage costs have fallen sharply. Grid integration remains a challenge."
	answer, toolResponse, sawGrounding, err := askAgent(t.Context(), a, "formatter_agent_test_app", question, "format_research_notes")
	skipOnQuota(t, err)
	if err != nil {
		t.Fatalf("askAgent() error = %v", err)
	}
	if answer == "" {
		t.Fatal("formatter agent returned no answer")
	}
	if sawGrounding {
		t.Error("formatter agent produced GroundingMetadata — it has no google_search tool to have caused this")
	}
	if toolResponse == nil {
		t.Fatal("no FunctionResponse from format_research_notes was observed")
	}
}

// TestCombinedAgent_UsesSearchAndCustomTool_Gemini is the live, behavioral
// proof for this module's central new finding: one agent, with
// IncludeServerSideToolInvocations set, genuinely uses both google_search
// and a custom function tool in the same conversation.
func TestCombinedAgent_UsesSearchAndCustomTool_Gemini(t *testing.T) {
	cfg := skipIfNoGemini(t)
	llmModel := buildModel(t, cfg)

	a, err := BuildCombinedAgent(llmModel)
	if err != nil {
		t.Fatalf("BuildCombinedAgent() error = %v", err)
	}

	answer, toolResponse, sawGrounding, err := askAgent(t.Context(), a, "combined_agent_test_app",
		"Research the most recent Nobel Prize in Physics winner and give me the formatted report.", "format_research_notes")
	skipOnQuota(t, err)
	if err != nil {
		t.Fatalf("askAgent() error = %v", err)
	}
	if answer == "" {
		t.Fatal("combined agent returned no answer")
	}
	if !sawGrounding {
		t.Error("no GroundingMetadata observed — google_search was not genuinely used")
	}
	if toolResponse == nil {
		t.Fatal("no FunctionResponse from format_research_notes was observed — the custom tool was not genuinely used")
	}
}

// TestMixedTools_WithoutServerSideFlag_Fails_Gemini is the permanent
// regression guard for this module's confirmed Gemini API restriction: a
// deliberately-invalid agent combining google_search with a custom tool,
// with no ToolConfig set, must fail with a 400 — proving the restriction is
// a real Gemini API constraint, not something this repo's own code enforces
// or could silently work around by accident.
func TestMixedTools_WithoutServerSideFlag_Fails_Gemini(t *testing.T) {
	cfg := skipIfNoGemini(t)
	llmModel := buildModel(t, cfg)

	tools, err := customTools()
	if err != nil {
		t.Fatalf("customTools() error = %v", err)
	}
	tools = append([]tool.Tool{geminitool.GoogleSearch{}}, tools...)

	invalidAgent, err := llmagent.New(llmagent.Config{
		Name:        "invalid_mixed_tools_agent",
		Model:       llmModel,
		Description: "Deliberately invalid: mixes google_search with custom tools, no ToolConfig set.",
		Instruction: "Say hello.",
		Tools:       tools,
	})
	if err != nil {
		t.Fatalf("llmagent.New() error = %v", err)
	}

	_, _, _, err = askAgent(t.Context(), invalidAgent, "invalid_mixed_tools_test_app", "Say hello.", "")
	if err == nil {
		t.Fatal("expected the mixed-tools call to fail without IncludeServerSideToolInvocations, got no error")
	}
	if strings.Contains(err.Error(), "RESOURCE_EXHAUSTED") {
		t.Skipf("skipping: Gemini free-tier quota exhausted for today (%v) — external rate limit, not a code defect", err)
	}
	if !strings.Contains(err.Error(), "INVALID_ARGUMENT") {
		t.Errorf("err = %v, want it to contain INVALID_ARGUMENT (the confirmed Gemini API rejection for mixed tool types)", err)
	}
	if !strings.Contains(err.Error(), "include_server_side_tool_invocations") {
		t.Errorf("err = %v, want it to name include_server_side_tool_invocations as the fix, per the confirmed live error text", err)
	}
}
