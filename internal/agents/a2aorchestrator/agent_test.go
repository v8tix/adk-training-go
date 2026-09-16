package a2aorchestrator

import (
	"net"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/a2aproject/a2a-go/v2/a2a"
	"github.com/a2aproject/a2a-go/v2/a2asrv"

	"github.com/v8tix/adk-training-go/internal/agents/researchspecialist"
	"github.com/v8tix/adk-training-go/internal/infrastructure/llm"
	"google.golang.org/adk/v2/agent"
	"google.golang.org/adk/v2/model"
	"google.golang.org/adk/v2/runner"
	"google.golang.org/adk/v2/server/adka2a/v2"
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
// independent of any live call or reachable specialist server.
func TestBuildRootAgent_Constructs(t *testing.T) {
	cfg := testConfig
	cfg.ModelType = llm.ModelTypeGemini
	cfg.GoogleAPIKey = "unused-for-construction"
	llmModel, _, err := llm.BuildModel(t.Context(), cfg)
	if err != nil {
		t.Fatalf("BuildModel() error = %v", err)
	}

	a, err := BuildRootAgent(llmModel, "http://unused.invalid")
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

// startSpecialistServer starts a real research_specialist A2A service on a
// genuine, OS-assigned TCP port — the same shape confirmed live in this
// module's own probe — and returns its base URL. The server is torn down
// automatically when the test completes.
func startSpecialistServer(t *testing.T, llmModel model.LLM) string {
	t.Helper()

	specialist, err := researchspecialist.BuildRootAgent(llmModel)
	if err != nil {
		t.Fatalf("researchspecialist.BuildRootAgent() error = %v", err)
	}

	executor := adka2a.NewExecutor(adka2a.ExecutorConfig{
		RunnerConfig: runner.Config{
			AppName:        "research_specialist_test",
			Agent:          specialist,
			SessionService: session.InMemoryService(),
		},
	})
	requestHandler := a2asrv.NewHandler(executor)

	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("net.Listen() error = %v", err)
	}
	baseURL := "http://" + lis.Addr().String()

	agentCard := &a2a.AgentCard{
		Name:                "research_specialist",
		SupportedInterfaces: []*a2a.AgentInterface{a2a.NewAgentInterface(baseURL+"/invoke", a2a.TransportProtocolJSONRPC)},
		Version:             "1.0.0",
		DefaultInputModes:   []string{"text/plain"},
		DefaultOutputModes:  []string{"text/plain"},
	}

	mux := http.NewServeMux()
	mux.Handle(a2asrv.WellKnownAgentCardPath, a2asrv.NewStaticAgentCardHandler(agentCard))
	mux.Handle("/invoke", a2asrv.NewJSONRPCHandler(requestHandler))

	server := &http.Server{Handler: mux, ReadHeaderTimeout: 5 * time.Second}
	go server.Serve(lis) //nolint:errcheck // server.Close() below always returns http.ErrServerClosed
	t.Cleanup(func() { server.Close() })

	return baseURL
}

// TestA2AOrchestrator_DelegatesToRemoteSpecialist_Ollama confirms a real,
// genuinely-networked A2A round trip: a real HTTP server (its own OS TCP
// port, not mocked) hosts the specialist, and the orchestrator's
// transfer_to_agent call reaches it over the network — asserted via the
// remote specialist's own distinct event author, the confirmed reliable
// signal — through local Ollama.
func TestA2AOrchestrator_DelegatesToRemoteSpecialist_Ollama(t *testing.T) {
	assertDelegatesToRemoteSpecialist(t, skipIfNoOllama(t))
}

// TestA2AOrchestrator_DelegatesToRemoteSpecialist_Gemini is the same
// behavior against real Gemini.
func TestA2AOrchestrator_DelegatesToRemoteSpecialist_Gemini(t *testing.T) {
	assertDelegatesToRemoteSpecialist(t, skipIfNoGemini(t))
}

func assertDelegatesToRemoteSpecialist(t *testing.T, cfg llm.Config) {
	t.Helper()

	llmModel, _, err := llm.BuildModel(t.Context(), cfg)
	if err != nil {
		t.Fatalf("BuildModel() error = %v", err)
	}

	specialistBaseURL := startSpecialistServer(t, llmModel)

	rootAgent, err := BuildRootAgent(llmModel, specialistBaseURL)
	if err != nil {
		t.Fatalf("BuildRootAgent() error = %v", err)
	}
	r, err := runner.NewInMemory("a2a_orchestrator_test_app", rootAgent)
	if err != nil {
		t.Fatalf("runner.NewInMemory() error = %v", err)
	}

	msg := genai.NewContentFromText("Please research the latest advancements in quantum computing.", genai.RoleUser)
	var sawRemoteSpecialistText string
	for event, runErr := range r.Run(t.Context(), "test_user", "test_session", msg, agent.RunConfig{}) {
		skipOnQuota(t, runErr)
		if runErr != nil {
			t.Fatalf("run error: %v", runErr)
		}
		if event.Author != "research_specialist" {
			continue
		}
		// event.Author alone is NOT a reliable signal: every failure path in
		// remoteagent/v2 (agent-card resolution failure, RPC failure, etc.)
		// also stamps Author with this same local wrapper agent's name on its
		// synthesized error event (confirmed by reading
		// agent/remoteagent/v2/a2a_agent.go's toErrorEvent and
		// server/adka2a/v2/events.go's NewRemoteAgentEvent) — a test checking
		// only the author would pass even if the network call failed
		// outright. Checking for a real, non-empty, error-free response is
		// what actually proves the round trip succeeded.
		if event.ErrorMessage != "" {
			t.Fatalf("research_specialist event carries an error, not a real response: %s", event.ErrorMessage)
		}
		if event.Content != nil {
			for _, p := range event.Content.Parts {
				if p.Text != "" && !p.Thought {
					sawRemoteSpecialistText += p.Text
				}
			}
		}
	}

	if sawRemoteSpecialistText == "" {
		t.Fatal("never observed real, non-empty response text from the remote research_specialist — the A2A delegation may not have genuinely happened")
	}
}

// TestA2AOrchestrator_UnreachableSpecialist_Gemini is the regression guard
// for the exact bug found in this test's own history: an earlier version
// asserted only on event.Author, which a failed remote call also stamps
// with the local wrapper agent's name — so that assertion would have passed
// here too. Pointing at a specialist URL nothing is listening on proves the
// current, content-based assertion genuinely distinguishes success from
// failure.
func TestA2AOrchestrator_UnreachableSpecialist_Gemini(t *testing.T) {
	cfg := skipIfNoGemini(t)

	llmModel, _, err := llm.BuildModel(t.Context(), cfg)
	if err != nil {
		t.Fatalf("BuildModel() error = %v", err)
	}

	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("net.Listen() error = %v", err)
	}
	unreachableURL := "http://" + lis.Addr().String()
	lis.Close() // nothing will ever answer on this address again

	rootAgent, err := BuildRootAgent(llmModel, unreachableURL)
	if err != nil {
		t.Fatalf("BuildRootAgent() error = %v", err)
	}
	r, err := runner.NewInMemory("a2a_orchestrator_negative_test_app", rootAgent)
	if err != nil {
		t.Fatalf("runner.NewInMemory() error = %v", err)
	}

	msg := genai.NewContentFromText("Please research the latest advancements in quantum computing.", genai.RoleUser)
	var sawErrorFromSpecialist bool
	for event, runErr := range r.Run(t.Context(), "test_user", "test_session", msg, agent.RunConfig{}) {
		skipOnQuota(t, runErr)
		if runErr != nil {
			t.Fatalf("run error: %v", runErr)
		}
		if event.Author == "research_specialist" && event.ErrorMessage != "" {
			sawErrorFromSpecialist = true
		}
	}

	if !sawErrorFromSpecialist {
		t.Fatal("expected an error event from research_specialist when its address is unreachable, got none — either the negative-path setup is wrong, or the positive test's assertion wouldn't actually catch this failure mode")
	}
}
