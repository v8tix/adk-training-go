package newsaggregator

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/v8tix/adk-training-go/internal/infrastructure/llm"
	"google.golang.org/adk/v2/agent"
	"google.golang.org/adk/v2/model"
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

// newsOutcome is what askForNews observes: whether each parallel branch's
// OutputKey was genuinely populated (proving both really ran, not just
// that the graph didn't error), and the summarizer's final text.
type newsOutcome struct {
	sawTechNews   bool
	sawMarketNews bool
	finalText     string
}

// askForNews drives the real BuildRootAgent through runner, inspecting
// each event's Actions.StateDelta directly for the two OutputKeys — a
// structural proof both parallel branches actually executed, not an
// assumption from the graph merely completing without error.
func askForNews(ctx context.Context, llmModel model.LLM, appName string) (newsOutcome, error) {
	rootAgent, err := BuildRootAgent(llmModel)
	if err != nil {
		return newsOutcome{}, err
	}

	r, err := runner.NewInMemory(appName, rootAgent)
	if err != nil {
		return newsOutcome{}, err
	}

	msg := genai.NewContentFromText("Give me today's update.", genai.RoleUser)
	var out newsOutcome
	for event, runErr := range r.Run(ctx, "test_user", "test_session", msg, agent.RunConfig{}) {
		if runErr != nil {
			return newsOutcome{}, runErr
		}
		if event.Actions.StateDelta != nil {
			if _, ok := event.Actions.StateDelta["tech_news"]; ok {
				out.sawTechNews = true
			}
			if _, ok := event.Actions.StateDelta["market_news"]; ok {
				out.sawMarketNews = true
			}
		}
		if event.Content == nil {
			continue
		}
		for _, p := range event.Content.Parts {
			if event.IsFinalResponse() && p.Text != "" && !p.Thought {
				out.finalText = p.Text
			}
		}
	}
	return out, nil
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

// TestNewsAggregator_ProducesNewsletter_Ollama confirms the full
// fan-out/fan-in/sequential graph works through local Ollama — no cloud
// fallback needed, matching every plain-llmagent module's precedent since
// module-9.
func TestNewsAggregator_ProducesNewsletter_Ollama(t *testing.T) {
	assertProducesNewsletter(t, skipIfNoOllama(t))
}

// TestNewsAggregator_ProducesNewsletter_Gemini is the same behavior against
// real Gemini.
func TestNewsAggregator_ProducesNewsletter_Gemini(t *testing.T) {
	assertProducesNewsletter(t, skipIfNoGemini(t))
}

// assertProducesNewsletter confirms both parallel branches genuinely ran
// (both OutputKeys populated) and the summarizer's final output reflects
// both — not an exact pinned string, since real headlines and model
// wording vary.
func assertProducesNewsletter(t *testing.T, cfg llm.Config) {
	t.Helper()

	llmModel, _, err := llm.BuildModel(t.Context(), cfg)
	if err != nil {
		t.Fatalf("BuildModel() error = %v", err)
	}

	out, err := askForNews(t.Context(), llmModel, "news_aggregator_test_app")
	skipOnQuota(t, err)
	if err != nil {
		t.Fatalf("askForNews() error = %v", err)
	}

	if !out.sawTechNews {
		t.Error("no tech_news OutputKey observed — the tech researcher branch may not have run")
	}
	if !out.sawMarketNews {
		t.Error("no market_news OutputKey observed — the market researcher branch may not have run")
	}
	if out.finalText == "" {
		t.Fatal("summarizer produced no final text")
	}
	if strings.Contains(out.finalText, "{tech_news}") || strings.Contains(out.finalText, "{market_news}") {
		t.Errorf("summarizer output contains an un-interpolated placeholder: %q — the {key} substitution silently failed", out.finalText)
	}
}
