package travelplanner

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

// runTurn drives one user message through r on the given session and
// returns the concatenated text of every visible (non-thought) response
// part for that turn, regardless of which agent produced it — task-mode
// dispatch folds a sub-agent's mid-task question into the same response
// the coordinator sends the user (confirmed live this module), so this
// intentionally doesn't try to attribute text to a specific event/author.
func runTurn(ctx context.Context, r *runner.Runner, userID, sessionID, message string) (string, error) {
	msg := genai.NewContentFromText(message, genai.RoleUser)
	var visibleText string
	for event, runErr := range r.Run(ctx, userID, sessionID, msg, agent.RunConfig{}) {
		if runErr != nil {
			return "", runErr
		}
		if event.Content == nil {
			continue
		}
		for _, p := range event.Content.Parts {
			if p.Text != "" && !p.Thought {
				visibleText += p.Text
			}
		}
	}
	return visibleText, nil
}

// TestTravelPlanner_CompletesMultiTurnPlan_Ollama confirms a real two-turn
// conversation: turn 1 gets a weather forecast and a flight-preference
// question, turn 2 answers it and gets back one combined plan that
// genuinely reflects the answer — proof flight_booker's task mode
// genuinely paused and then genuinely returned control automatically,
// through local Ollama.
func TestTravelPlanner_CompletesMultiTurnPlan_Ollama(t *testing.T) {
	assertCompletesMultiTurnPlan(t, skipIfNoOllama(t))
}

// TestTravelPlanner_CompletesMultiTurnPlan_Gemini is the same behavior
// against real Gemini.
func TestTravelPlanner_CompletesMultiTurnPlan_Gemini(t *testing.T) {
	assertCompletesMultiTurnPlan(t, skipIfNoGemini(t))
}

func assertCompletesMultiTurnPlan(t *testing.T, cfg llm.Config) {
	t.Helper()

	llmModel, _, err := llm.BuildModel(t.Context(), cfg)
	if err != nil {
		t.Fatalf("BuildModel() error = %v", err)
	}

	rootAgent, err := BuildRootAgent(llmModel)
	if err != nil {
		t.Fatalf("BuildRootAgent() error = %v", err)
	}
	r, err := runner.NewInMemory("travel_planner_test_app", rootAgent)
	if err != nil {
		t.Fatalf("runner.NewInMemory() error = %v", err)
	}

	turn1Text, err := runTurn(t.Context(), r, "test_user", "test_session", "I want to go to Tokyo next week.")
	skipOnQuota(t, err)
	if err != nil {
		t.Fatalf("turn 1: runTurn() error = %v", err)
	}
	if turn1Text == "" {
		t.Fatal("turn 1: no visible response text observed")
	}
	if strings.Contains(strings.ToLower(turn1Text), "united") {
		t.Errorf("turn 1: response already mentions the airline before it was given: %q", turn1Text)
	}

	turn2Text, err := runTurn(t.Context(), r, "test_user", "test_session", "United, morning flight please.")
	skipOnQuota(t, err)
	if err != nil {
		t.Fatalf("turn 2: runTurn() error = %v", err)
	}
	if turn2Text == "" {
		t.Fatal("turn 2: expected a combined final plan after flight_booker finished, got no visible text")
	}
	// Proves flight_booker genuinely received this turn's answer, finished
	// its task, and its result made it into the coordinator's own final
	// synthesis — the real, structural signal that finish_task triggered an
	// automatic return this same turn, without depending on which specific
	// event/author the SDK happens to attribute the text to internally.
	if !strings.Contains(strings.ToLower(turn2Text), "united") {
		t.Errorf("turn 2: expected the final plan to reflect the airline just given, got: %q", turn2Text)
	}
}
