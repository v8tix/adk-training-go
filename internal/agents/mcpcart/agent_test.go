package mcpcart

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/v8tix/adk-training-go/internal/infrastructure/llm"
	"google.golang.org/adk/v2/agent"
	"google.golang.org/adk/v2/runner"
	"google.golang.org/adk/v2/session"
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
// independent of any live call — mcptoolset.New's own doc comment confirms
// MCP session creation is lazy, so construction never launches the server
// subprocess, even for a repoRoot that doesn't exist.
func TestBuildRootAgent_Constructs(t *testing.T) {
	cfg := testConfig
	cfg.ModelType = llm.ModelTypeGemini
	cfg.GoogleAPIKey = "unused-for-construction"
	llmModel, _, err := llm.BuildModel(t.Context(), cfg)
	if err != nil {
		t.Fatalf("BuildModel() error = %v", err)
	}

	a, err := BuildRootAgent(llmModel, "/does/not/exist")
	if err != nil {
		t.Fatalf("BuildRootAgent() error = %v", err)
	}
	if a == nil {
		t.Fatal("BuildRootAgent() returned a nil agent")
	}
}

// TestBuildServerCmd_ForwardsStderr is the direct regression guard for a
// real bug found live during Build: mcp.CommandTransport never touches a
// launched subprocess's Stderr, so a nil Cmd.Stderr silently discards every
// log line cart-mcp-server's own handlers emit — confirmed against Go's own
// os/exec docs and cmd.go directly. Without this test, a future refactor
// could drop serverCmd.Stderr = os.Stderr with nothing turning red.
func TestBuildServerCmd_ForwardsStderr(t *testing.T) {
	got := buildServerCmd("/does/not/exist")

	if got.Stderr != os.Stderr {
		t.Error("buildServerCmd().Stderr is not forwarded to os.Stderr — the server's own log lines would be silently discarded")
	}
	if got.Dir != "/does/not/exist" {
		t.Errorf("buildServerCmd().Dir = %q, want %q", got.Dir, "/does/not/exist")
	}
}

// askAgent drives a single Run() call and returns the final text answer.
func askAgent(ctx context.Context, r *runner.Runner, userID, sessionID, message string) (answer string, err error) {
	msg := genai.NewContentFromText(message, genai.RoleUser)
	for event, err := range r.Run(ctx, userID, sessionID, msg, agent.RunConfig{}) {
		if err != nil {
			return "", err
		}
		if event.Content == nil {
			continue
		}
		for _, p := range event.Content.Parts {
			if event.IsFinalResponse() && p.Text != "" && !p.Thought {
				answer = p.Text
			}
		}
	}
	return answer, nil
}

// TestShoppingCart_AddsAndViewsItems_Ollama exercises the real, custom
// cart-mcp-server against the local Ollama backend — the whole point of
// this module: state genuinely carried across turns by a server this repo
// wrote itself, not a mocked handler.
func TestShoppingCart_AddsAndViewsItems_Ollama(t *testing.T) {
	if !llm.SelectedTestBackend().IncludesOllama() {
		t.Skip("skipping: TEST_BACKEND excludes ollama")
	}
	if !ollamaReachable {
		t.Skip("skipping: local Ollama server (" + testConfig.OllamaBaseURL + ") is not reachable")
	}

	cfg := testConfig
	cfg.ModelType = llm.ModelTypeOllama
	assertShoppingCartAddsAndViewsItems(t, cfg)
}

// TestShoppingCart_AddsAndViewsItems_Gemini is the same behavior against
// the real Gemini backend.
func TestShoppingCart_AddsAndViewsItems_Gemini(t *testing.T) {
	if !llm.SelectedTestBackend().IncludesGemini() {
		t.Skip("skipping: TEST_BACKEND excludes gemini")
	}
	if testConfig.GoogleAPIKey == "" {
		t.Skip("skipping: GOOGLE_AI_STUDIO_API_KEY is not set")
	}

	cfg := testConfig
	cfg.ModelType = llm.ModelTypeGemini
	assertShoppingCartAddsAndViewsItems(t, cfg)
}

func assertShoppingCartAddsAndViewsItems(t *testing.T, cfg llm.Config) {
	t.Helper()

	llmModel, _, err := llm.BuildModel(t.Context(), cfg)
	if err != nil {
		t.Fatalf("BuildModel() error = %v", err)
	}

	root, err := RepoRoot()
	if err != nil {
		t.Fatalf("RepoRoot() error = %v", err)
	}

	a, err := BuildRootAgent(llmModel, root)
	if err != nil {
		t.Fatalf("BuildRootAgent() error = %v", err)
	}

	r, err := runner.New(runner.Config{
		AppName:           "mcpcart_test_app",
		Agent:             a,
		SessionService:    session.InMemoryService(),
		AutoCreateSession: true,
	})
	if err != nil {
		t.Fatalf("building runner: %v", err)
	}

	const (
		userID    = "test_user"
		sessionID = "test_session"
	)
	skipIfExhausted := func(turn string, err error) {
		t.Helper()
		if err != nil && strings.Contains(err.Error(), "RESOURCE_EXHAUSTED") {
			t.Skipf("skipping: Gemini free-tier quota exhausted for today (%v) — this is an external rate limit, not a code defect; re-run once it resets (%s)", err, turn)
		}
	}

	if _, err := askAgent(t.Context(), r, userID, sessionID, "Please add milk to my cart."); err != nil {
		skipIfExhausted("add milk", err)
		t.Fatalf("askAgent(add milk) error = %v", err)
	}
	if _, err := askAgent(t.Context(), r, userID, sessionID, "Also add eggs."); err != nil {
		skipIfExhausted("add eggs", err)
		t.Fatalf("askAgent(add eggs) error = %v", err)
	}

	viewAnswer, err := askAgent(t.Context(), r, userID, sessionID, "What's in my shopping cart?")
	skipIfExhausted("view cart", err)
	if err != nil {
		t.Fatalf("askAgent(view cart) error = %v", err)
	}
	if !strings.Contains(strings.ToLower(viewAnswer), "milk") {
		t.Errorf("view cart answer = %q, want it to mention milk", viewAnswer)
	}
	if !strings.Contains(strings.ToLower(viewAnswer), "eggs") {
		t.Errorf("view cart answer = %q, want it to mention eggs", viewAnswer)
	}
}
