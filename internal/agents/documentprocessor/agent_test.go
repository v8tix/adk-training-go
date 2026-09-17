package documentprocessor

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/v8tix/adk-training-go/internal/infrastructure/llm"
	"google.golang.org/adk/v2/agent"
	"google.golang.org/adk/v2/artifact"
	"google.golang.org/adk/v2/runner"
	"google.golang.org/adk/v2/session"
	"google.golang.org/genai"
)

// testConfig and ollamaReachable are set once by TestMain, matching
// internal/agents/personaltutor's own pattern.
var (
	testConfig      llm.Config
	ollamaReachable bool
)

func TestMain(m *testing.M) {
	testConfig = llm.LoadConfig()
	ollamaReachable = llm.OllamaReachable(testConfig, 2*time.Second)
	os.Exit(m.Run())
}

// askDocumentProcessor drives the real BuildRootAgent through a single
// Run() call. It returns the final text answer, plus every tool's own
// FunctionResponse seen in that turn, keyed by tool name — since one
// natural-language request is expected to trigger all four pipeline
// tools in sequence (matching Python's own lab: a single "Process the
// document named 'X'" message). Keying by name is deliberately permissive
// about a model retrying a tool call (a second call just overwrites the
// first entry) — the real, non-negotiable proof this test relies on is the
// artifact-store read-back below, not the exact number of times each tool
// fired.
func askDocumentProcessor(ctx context.Context, r *runner.Runner, userID, sessionID, message string) (answer string, toolResponses map[string]map[string]any, err error) {
	toolResponses = make(map[string]map[string]any)
	msg := genai.NewContentFromText(message, genai.RoleUser)
	for event, err := range r.Run(ctx, userID, sessionID, msg, agent.RunConfig{}) {
		if err != nil {
			return "", nil, err
		}
		if event.Content == nil {
			continue
		}
		for _, p := range event.Content.Parts {
			if p.FunctionResponse != nil {
				toolResponses[p.FunctionResponse.Name] = p.FunctionResponse.Response
			}
			if event.IsFinalResponse() && p.Text != "" && !p.Thought {
				answer = p.Text
			}
		}
	}
	return answer, toolResponses, nil
}

// TestDocumentProcessor_BuildsVersionedPipeline_Ollama exercises this
// module's own finding against the local Ollama backend.
func TestDocumentProcessor_BuildsVersionedPipeline_Ollama(t *testing.T) {
	if !llm.SelectedTestBackend().IncludesOllama() {
		t.Skip("skipping: TEST_BACKEND excludes ollama")
	}
	if !ollamaReachable {
		t.Skip("skipping: local Ollama server (" + testConfig.OllamaBaseURL + ") is not reachable")
	}

	cfg := testConfig
	cfg.ModelType = llm.ModelTypeOllama
	assertDocumentProcessorBuildsPipeline(t, cfg)
}

// TestDocumentProcessor_BuildsVersionedPipeline_Gemini is the same behavior
// against the real Gemini backend.
func TestDocumentProcessor_BuildsVersionedPipeline_Gemini(t *testing.T) {
	if !llm.SelectedTestBackend().IncludesGemini() {
		t.Skip("skipping: TEST_BACKEND excludes gemini")
	}
	if testConfig.GoogleAPIKey == "" {
		t.Skip("skipping: GOOGLE_AI_STUDIO_API_KEY is not set")
	}

	cfg := testConfig
	cfg.ModelType = llm.ModelTypeGemini
	assertDocumentProcessorBuildsPipeline(t, cfg)
}

// assertDocumentProcessorBuildsPipeline drives the full four-step pipeline
// through a single Run() call, then reads the artifact store directly
// (sessionService/artifactService, not the model's own text) to prove the
// real, structural output: every step's artifact exists, the first save is
// version 1, the chart is genuinely binary with the right MIME type, and
// the final report's own content references the chart — the real proof
// the pipeline chains correctly, not just that each tool call individually
// reported success.
func assertDocumentProcessorBuildsPipeline(t *testing.T, cfg llm.Config) {
	t.Helper()

	llmModel, _, err := llm.BuildModel(t.Context(), cfg)
	if err != nil {
		t.Fatalf("BuildModel() error = %v", err)
	}

	docAgent, err := BuildRootAgent(llmModel)
	if err != nil {
		t.Fatalf("BuildRootAgent() error = %v", err)
	}

	const appName = "document_processor_test_app"
	artifactService := artifact.InMemoryService()
	r, err := runner.New(runner.Config{
		AppName:           appName,
		Agent:             docAgent,
		SessionService:    session.InMemoryService(),
		ArtifactService:   artifactService,
		AutoCreateSession: true,
	})
	if err != nil {
		t.Fatalf("building runner: %v", err)
	}

	const (
		userID       = "test_user"
		sessionID    = "test_session"
		documentName = "AnnualReport"
	)

	answer, toolResponses, err := askDocumentProcessor(t.Context(), r, userID, sessionID,
		"Process the document named 'AnnualReport'.")
	if err != nil {
		if strings.Contains(err.Error(), "RESOURCE_EXHAUSTED") {
			t.Skipf("skipping: Gemini free-tier quota exhausted for today (%v) — this is an external rate limit, not a code defect; re-run once it resets", err)
		}
		t.Fatalf("askDocumentProcessor() error = %v", err)
	}
	if answer == "" {
		t.Error("agent returned no final answer")
	}

	for _, toolName := range []string{"extract_text", "summarize_document", "generate_chart", "create_report"} {
		if toolResponses[toolName] == nil {
			t.Errorf("tool %q was not called", toolName)
		}
	}

	// Structural proof, read directly from the artifact service rather than
	// parsed from any model reply.
	listResp, err := artifactService.List(t.Context(), &artifact.ListRequest{
		AppName: appName, UserID: userID, SessionID: sessionID,
	})
	if err != nil {
		t.Fatalf("artifactService.List() error = %v", err)
	}
	wantNames := map[string]bool{
		extractedArtifactName(documentName): false,
		summaryArtifactName(documentName):   false,
		chartArtifactName(documentName):     false,
		reportArtifactName(documentName):    false,
	}
	for _, name := range listResp.FileNames {
		if _, ok := wantNames[name]; ok {
			wantNames[name] = true
		}
	}
	for name, found := range wantNames {
		if !found {
			t.Errorf("artifact %q was not saved; List() returned %v", name, listResp.FileNames)
		}
	}

	extractedResp, err := artifactService.Load(t.Context(), &artifact.LoadRequest{
		AppName: appName, UserID: userID, SessionID: sessionID,
		FileName: extractedArtifactName(documentName), Version: 1,
	})
	if err != nil {
		t.Errorf("artifactService.Load(%q, version=1) error = %v — the first save must be version 1", extractedArtifactName(documentName), err)
	} else if extractedResp.Part.Text == "" {
		t.Error("extracted-text artifact has no text content")
	}

	summaryResp, err := artifactService.Load(t.Context(), &artifact.LoadRequest{
		AppName: appName, UserID: userID, SessionID: sessionID,
		FileName: summaryArtifactName(documentName),
	})
	if err != nil {
		t.Errorf("artifactService.Load(summary) error = %v", err)
	} else if summaryResp.Part.Text == "" {
		t.Error("summary artifact has no text content")
	}

	chartResp, err := artifactService.Load(t.Context(), &artifact.LoadRequest{
		AppName: appName, UserID: userID, SessionID: sessionID,
		FileName: chartArtifactName(documentName),
	})
	if err != nil {
		t.Fatalf("artifactService.Load(chart) error = %v", err)
	}
	if chartResp.Part.InlineData == nil || chartResp.Part.InlineData.MIMEType != "image/png" {
		t.Errorf("chart artifact InlineData.MIMEType = %+v, want %q", chartResp.Part.InlineData, "image/png")
	}

	reportResp, err := artifactService.Load(t.Context(), &artifact.LoadRequest{
		AppName: appName, UserID: userID, SessionID: sessionID,
		FileName: reportArtifactName(documentName),
	})
	if err != nil {
		t.Fatalf("artifactService.Load(report) error = %v", err)
	}
	if !strings.Contains(reportResp.Part.Text, chartArtifactName(documentName)) {
		t.Errorf("report content does not reference the chart artifact — pipeline steps did not genuinely chain: %q", reportResp.Part.Text)
	}
}
