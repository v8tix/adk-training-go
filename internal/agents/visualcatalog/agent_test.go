package visualcatalog

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/v8tix/adk-training-go/internal/infrastructure/llm"
	"google.golang.org/adk/v2/agent"
	"google.golang.org/adk/v2/runner"
	"google.golang.org/genai"
)

// TestVisualCatalog_AnalyzesImage_Gemini is the only variant this agent can
// have — unlike internal/agents/supportanalyzer, there's no _Ollama
// counterpart, because the local backend can't run it at all: confirmed live
// (module-7) that model/openaimodel's request-builder has no code path for
// an image Part and errors before ever reaching the network, regardless of
// which model is loaded. It skips when TEST_BACKEND excludes Gemini, or when
// GOOGLE_AI_STUDIO_API_KEY isn't set, since a Gemini credential isn't
// required for this local-first course's other modules.
func TestVisualCatalog_AnalyzesImage_Gemini(t *testing.T) {
	if !llm.SelectedTestBackend().IncludesGemini() {
		t.Skip("skipping: TEST_BACKEND excludes gemini (this agent has no Ollama path — vision requires Gemini)")
	}
	cfg := llm.LoadConfig()
	if cfg.GoogleAPIKey == "" {
		t.Skip("skipping: GOOGLE_AI_STUDIO_API_KEY is not set")
	}
	cfg.ModelType = llm.ModelTypeGemini

	llmModel, _, err := llm.BuildModel(t.Context(), cfg)
	if err != nil {
		t.Fatalf("BuildModel() error = %v", err)
	}

	catalogAgent, err := BuildRootAgent(llmModel)
	if err != nil {
		t.Fatalf("BuildRootAgent() error = %v", err)
	}

	imageBytes, err := os.ReadFile("testdata/headphones.jpg")
	if err != nil {
		t.Fatalf("reading test image: %v", err)
	}

	r, err := runner.NewInMemory("visual_catalog_test_app", catalogAgent)
	if err != nil {
		t.Fatalf("building runner: %v", err)
	}

	msg := genai.NewContentFromParts([]*genai.Part{
		genai.NewPartFromText("What is in this image? One short sentence."),
		genai.NewPartFromBytes(imageBytes, "image/jpeg"),
	}, genai.RoleUser)

	got, err := describeImage(t.Context(), r, msg)
	if err != nil && strings.Contains(err.Error(), "RESOURCE_EXHAUSTED") {
		t.Skipf("skipping: Gemini free-tier quota exhausted for today (%v) — this is an external rate limit, not a code defect; re-run once it resets", err)
	}
	if err != nil {
		t.Fatalf("describeImage() error = %v", err)
	}
	if got == "" {
		t.Fatal("agent returned no description")
	}
	// A real, structural check that the model actually looked at the image,
	// not just returned generic filler: this specific photo is of
	// headphones, so a correct description must mention them.
	if !strings.Contains(strings.ToLower(got), "headphone") {
		t.Errorf("description = %q, want it to mention headphones (the actual subject of testdata/headphones.jpg)", got)
	}
}

func describeImage(ctx context.Context, r *runner.Runner, msg *genai.Content) (string, error) {
	for event, err := range r.Run(ctx, "test_user", "test_session", msg, agent.RunConfig{}) {
		if err != nil {
			return "", err
		}
		if !event.IsFinalResponse() || event.Content == nil {
			continue
		}
		for _, p := range event.Content.Parts {
			if p.Text != "" && !p.Thought {
				return p.Text, nil
			}
		}
	}
	return "", nil
}
