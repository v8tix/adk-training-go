package mcpfilesystem

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
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

// npxReachable reports whether npx is on PATH — the filesystem MCP server is
// a third-party Node.js package this repo doesn't control, so a live test
// against it must skip cleanly (not fail) when Node.js isn't installed,
// matching llm.OllamaReachable's own skip-not-fail discipline for a
// different kind of external dependency.
func npxReachable() bool {
	_, err := exec.LookPath("npx")
	return err == nil
}

// TestBuildRootAgent_Constructs is a structural regression guard,
// independent of any live call — mcptoolset.New's own doc comment confirms
// MCP session creation is lazy, so construction never dials npx or anything
// else, even for a sandbox directory that doesn't exist yet.
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

// TestFilesystemMCP_ListsAndReadsFile_Ollama exercises the real filesystem
// MCP server against the local Ollama backend.
func TestFilesystemMCP_ListsAndReadsFile_Ollama(t *testing.T) {
	if !llm.SelectedTestBackend().IncludesOllama() {
		t.Skip("skipping: TEST_BACKEND excludes ollama")
	}
	if !ollamaReachable {
		t.Skip("skipping: local Ollama server (" + testConfig.OllamaBaseURL + ") is not reachable")
	}
	if !npxReachable() {
		t.Skip("skipping: npx is not on PATH — install Node.js to run this test")
	}

	cfg := testConfig
	cfg.ModelType = llm.ModelTypeOllama
	assertFilesystemMCPListsAndReadsFile(t, cfg)
}

// TestFilesystemMCP_ListsAndReadsFile_Gemini is the same behavior against
// the real Gemini backend.
func TestFilesystemMCP_ListsAndReadsFile_Gemini(t *testing.T) {
	if !llm.SelectedTestBackend().IncludesGemini() {
		t.Skip("skipping: TEST_BACKEND excludes gemini")
	}
	if testConfig.GoogleAPIKey == "" {
		t.Skip("skipping: GOOGLE_AI_STUDIO_API_KEY is not set")
	}
	if !npxReachable() {
		t.Skip("skipping: npx is not on PATH — install Node.js to run this test")
	}

	cfg := testConfig
	cfg.ModelType = llm.ModelTypeGemini
	assertFilesystemMCPListsAndReadsFile(t, cfg)
}

// assertFilesystemMCPListsAndReadsFile builds its own isolated sandbox
// directory with a known fixture file — proving the general mechanism
// without depending on any checked-in fixture — then drives two real turns
// against a real runner.New: list the directory, then read the file back.
func assertFilesystemMCPListsAndReadsFile(t *testing.T, cfg llm.Config) {
	t.Helper()

	llmModel, _, err := llm.BuildModel(t.Context(), cfg)
	if err != nil {
		t.Fatalf("BuildModel() error = %v", err)
	}

	sandboxDir := t.TempDir()
	const fileName = "hello.txt"
	const fileContent = "Hello from the MCP world!"
	if err := os.WriteFile(filepath.Join(sandboxDir, fileName), []byte(fileContent), 0o644); err != nil {
		t.Fatalf("writing fixture file: %v", err)
	}

	a, err := BuildRootAgent(llmModel, sandboxDir)
	if err != nil {
		t.Fatalf("BuildRootAgent() error = %v", err)
	}

	r, err := runner.New(runner.Config{
		AppName:           "mcpfilesystem_test_app",
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

	listAnswer, err := askAgent(t.Context(), r, userID, sessionID, "What files are in my directory?")
	skipIfExhausted("list turn", err)
	if err != nil {
		t.Fatalf("askAgent(list turn) error = %v", err)
	}
	if !strings.Contains(listAnswer, fileName) {
		t.Errorf("list turn answer = %q, want it to mention the real file name %q", listAnswer, fileName)
	}

	readAnswer, err := askAgent(t.Context(), r, userID, sessionID, fmt.Sprintf("Great, can you read the content of %s for me?", fileName))
	skipIfExhausted("read turn", err)
	if err != nil {
		t.Fatalf("askAgent(read turn) error = %v", err)
	}
	if !strings.Contains(readAnswer, fileContent) {
		t.Errorf("read turn answer = %q, want it to contain the real file content %q", readAnswer, fileContent)
	}
}
