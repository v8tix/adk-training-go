package mcpgithub

import (
	"context"
	"net/http"
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

var testConfig llm.Config

func TestMain(m *testing.M) {
	testConfig = llm.LoadConfig()
	os.Exit(m.Run())
}

// fakeRoundTripper records the last request it saw and returns a canned
// response — a stub, not a real network call, so the header-setting logic
// can be proven without a token or network access.
type fakeRoundTripper struct {
	gotReq *http.Request
}

func (f *fakeRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	f.gotReq = req
	return &http.Response{StatusCode: http.StatusOK, Body: http.NoBody, Header: http.Header{}}, nil
}

// TestGithubAuthTransport_SetsBothHeaders proves the exact two headers
// GitHub's MCP server expects are set on every request, matching Python's
// own bonus headers dict exactly.
func TestGithubAuthTransport_SetsBothHeaders(t *testing.T) {
	fake := &fakeRoundTripper{}
	transport := &githubAuthTransport{token: "test-token-123", base: fake}

	req, err := http.NewRequest(http.MethodPost, githubEndpoint, nil)
	if err != nil {
		t.Fatalf("building request: %v", err)
	}

	if _, err := transport.RoundTrip(req); err != nil {
		t.Fatalf("RoundTrip() error = %v", err)
	}

	if fake.gotReq == nil {
		t.Fatal("RoundTrip() never called the base transport")
	}
	if got := fake.gotReq.Header.Get("Authorization"); got != "Bearer test-token-123" {
		t.Errorf("Authorization header = %q, want %q", got, "Bearer test-token-123")
	}
	if got := fake.gotReq.Header.Get("X-MCP-Readonly"); got != "true" {
		t.Errorf("X-MCP-Readonly header = %q, want %q", got, "true")
	}
	if fake.gotReq == req {
		t.Error("RoundTrip() passed the caller's own *http.Request to the base transport, want a clone")
	}
	if got := req.Header.Get("Authorization"); got != "" {
		t.Errorf("caller's original request Authorization header = %q, want it left unmodified (empty) — RoundTrip should clone before mutating", got)
	}
}

// TestGithubAuthTransport_DefaultsToDefaultTransport proves a nil base
// falls back to http.DefaultTransport rather than panicking.
func TestGithubAuthTransport_DefaultsToDefaultTransport(t *testing.T) {
	transport := &githubAuthTransport{token: "unused"}
	req, err := http.NewRequest(http.MethodGet, "https://example.invalid", nil)
	if err != nil {
		t.Fatalf("building request: %v", err)
	}
	// A real network call to a .invalid TLD always fails, but it must fail
	// with a network error, never a nil-pointer panic on t.base.
	if _, err := transport.RoundTrip(req); err == nil {
		t.Fatal("RoundTrip() to a .invalid host unexpectedly succeeded")
	}
}

// TestBuildRootAgent_Constructs is a structural regression guard,
// independent of any live call — mcptoolset.New's own doc comment confirms
// MCP session creation is lazy, so construction never dials the GitHub
// server, even for an empty token.
func TestBuildRootAgent_Constructs(t *testing.T) {
	cfg := testConfig
	cfg.ModelType = llm.ModelTypeGemini
	cfg.GoogleAPIKey = "unused-for-construction"
	llmModel, _, err := llm.BuildModel(t.Context(), cfg)
	if err != nil {
		t.Fatalf("BuildModel() error = %v", err)
	}

	a, err := BuildRootAgent(llmModel, "")
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

// TestGitHubMCP_ListsOpenIssues_Ollama exercises the real GitHub MCP server
// against the local Ollama backend. Skips cleanly (not fails) when
// GITHUB_TOKEN isn't set — a real Personal Access Token isn't a hard
// requirement to run this repo's own suite, the same discipline this
// repo's Gemini-optional tests already follow for GOOGLE_AI_STUDIO_API_KEY.
func TestGitHubMCP_ListsOpenIssues_Ollama(t *testing.T) {
	if !llm.SelectedTestBackend().IncludesOllama() {
		t.Skip("skipping: TEST_BACKEND excludes ollama")
	}
	token := os.Getenv("GITHUB_TOKEN")
	if token == "" {
		t.Skip("skipping: GITHUB_TOKEN is not set")
	}
	if !llm.OllamaReachable(testConfig, 2*time.Second) {
		t.Skip("skipping: local Ollama server (" + testConfig.OllamaBaseURL + ") is not reachable")
	}

	cfg := testConfig
	cfg.ModelType = llm.ModelTypeOllama
	assertGitHubMCPListsOpenIssues(t, cfg, token)
}

// assertGitHubMCPListsOpenIssues drives one real turn against a real
// runner.New, asking about a well-known public repository's open issues.
func assertGitHubMCPListsOpenIssues(t *testing.T, cfg llm.Config, token string) {
	t.Helper()

	llmModel, _, err := llm.BuildModel(t.Context(), cfg)
	if err != nil {
		t.Fatalf("BuildModel() error = %v", err)
	}

	a, err := BuildRootAgent(llmModel, token)
	if err != nil {
		t.Fatalf("BuildRootAgent() error = %v", err)
	}

	r, err := runner.New(runner.Config{
		AppName:           "mcpgithub_test_app",
		Agent:             a,
		SessionService:    session.InMemoryService(),
		AutoCreateSession: true,
	})
	if err != nil {
		t.Fatalf("building runner: %v", err)
	}

	answer, err := askAgent(t.Context(), r, "test_user", "test_session", "What are the open issues on google/adk-go?")
	if err != nil && strings.Contains(err.Error(), "RESOURCE_EXHAUSTED") {
		t.Skipf("skipping: Gemini free-tier quota exhausted for today (%v)", err)
	}
	if err != nil {
		t.Fatalf("askAgent() error = %v", err)
	}
	if answer == "" {
		t.Fatal("agent returned no answer")
	}
}
