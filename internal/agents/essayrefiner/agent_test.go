package essayrefiner

import (
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

// TestEssayRefiner_RefinesUntilApproved_Ollama confirms the loop genuinely
// iterates and terminates with a story that incorporates the critic's
// required element — proof the critic/refiner cycle actually ran, not just
// that the graph completed — through local Ollama.
func TestEssayRefiner_RefinesUntilApproved_Ollama(t *testing.T) {
	assertRefinesUntilApproved(t, skipIfNoOllama(t))
}

// TestEssayRefiner_RefinesUntilApproved_Gemini is the same behavior against
// real Gemini.
func TestEssayRefiner_RefinesUntilApproved_Gemini(t *testing.T) {
	assertRefinesUntilApproved(t, skipIfNoGemini(t))
}

func assertRefinesUntilApproved(t *testing.T, cfg llm.Config) {
	t.Helper()

	llmModel, _, err := llm.BuildModel(t.Context(), cfg)
	if err != nil {
		t.Fatalf("BuildModel() error = %v", err)
	}

	rootAgent, err := BuildRootAgent(llmModel)
	if err != nil {
		t.Fatalf("BuildRootAgent() error = %v", err)
	}
	r, err := runner.NewInMemory("essay_refiner_test_app", rootAgent)
	if err != nil {
		t.Fatalf("runner.NewInMemory() error = %v", err)
	}

	msg := genai.NewContentFromText("a lonely lighthouse keeper", genai.RoleUser)
	var finalStory string
	var sawWorkflowOutput bool
	for event, runErr := range r.Run(t.Context(), "test_user", "test_session", msg, agent.RunConfig{}) {
		skipOnQuota(t, runErr)
		if runErr != nil {
			t.Fatalf("run error: %v", runErr)
		}
		// The dynamic node's own terminal event is authored by the root
		// workflow agent's own name and carries the loop's real return
		// value in Output, not Content — confirmed live in this module's
		// probe. Any of the writer/critic/refiner's own chat-content events
		// (including the critic's final "APPROVED" reply) are NOT the
		// answer, even though they're the visible chat stream.
		if event.Author == "EssayRefiner" {
			sawWorkflowOutput = true
			s, ok := event.Output.(string)
			if !ok {
				t.Fatalf("workflow terminal event.Output type = %T, want string", event.Output)
			}
			finalStory = s
		}
	}

	if !sawWorkflowOutput {
		t.Fatal("never observed the workflow's own terminal event (Author == \"EssayRefiner\")")
	}
	if finalStory == "" {
		t.Fatal("workflow terminal Output was an empty string")
	}
	// The critic only approves once the story contains the literal word
	// "treasure" — the initial writer draft has no reason to include it on
	// its own, so this proves the critic/refiner loop genuinely ran and the
	// feedback was genuinely incorporated, not just that some story was
	// returned.
	if !strings.Contains(strings.ToLower(finalStory), "treasure") {
		t.Errorf("final story does not mention a treasure — the critic/refiner loop may not have genuinely run: %q", finalStory)
	}
}
