package researcher

import (
	"context"
	"strings"
	"testing"

	"github.com/v8tix/adk-training-go/internal/infrastructure/llm"
	"google.golang.org/adk/v2/agent"
	"google.golang.org/adk/v2/runner"
	"google.golang.org/genai"
)

// TestResearcher_UsesGoogleSearch_Gemini is the only variant this agent can
// have — unlike internal/agents/supportanalyzer, there's no _Ollama
// counterpart, because the local backend can't run it at all: confirmed live
// (module-8) that model/openaimodel unconditionally rejects any non-function
// tool, including the built-in google_search, before ever reaching the
// network. It skips when TEST_BACKEND excludes Gemini, or when
// GOOGLE_AI_STUDIO_API_KEY isn't set, since a Gemini credential isn't
// required for this local-first course's other modules.
//
// The prompt asks for the current weather in a named real city rather than
// an open-ended "what happened recently" question — weather cannot exist in
// any model's training data, so a tool call is effectively mandatory rather
// than merely likely, keeping this test far less flake-prone than an
// open-ended current-events question the model might occasionally answer
// from partial memory.
func TestResearcher_UsesGoogleSearch_Gemini(t *testing.T) {
	if !llm.SelectedTestBackend().IncludesGemini() {
		t.Skip("skipping: TEST_BACKEND excludes gemini (this agent has no Ollama path — google_search requires Gemini)")
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

	researcherAgent, err := BuildRootAgent(llmModel)
	if err != nil {
		t.Fatalf("BuildRootAgent() error = %v", err)
	}

	r, err := runner.NewInMemory("researcher_test_app", researcherAgent)
	if err != nil {
		t.Fatalf("building runner: %v", err)
	}

	msg := genai.NewContentFromText("What is the current weather in Tokyo right now?", genai.RoleUser)

	got, searchQueries, err := askResearcher(t.Context(), r, msg)
	if err != nil && strings.Contains(err.Error(), "RESOURCE_EXHAUSTED") {
		t.Skipf("skipping: Gemini free-tier quota exhausted for today (%v) — this is an external rate limit, not a code defect; re-run once it resets", err)
	}
	if err != nil {
		t.Fatalf("askResearcher() error = %v", err)
	}
	if got == "" {
		t.Fatal("agent returned no answer")
	}
	// A real, structural check that the agent actually searched, not just
	// answered from memory: a non-empty WebSearchQueries is stronger than a
	// merely non-nil GroundingMetadata (which could theoretically be present
	// but empty) — confirmed live in this module's Phase 1 probe that a real
	// search populates real queries here.
	if len(searchQueries) == 0 {
		t.Errorf("answer = %q, want the final response event's GroundingMetadata to carry at least one WebSearchQuery (proof google_search fired)", got)
	}
}

// askResearcher returns the first final response event's answer text and its
// GroundingMetadata.WebSearchQueries (nil if ungrounded), matching
// visualcatalog/agent_test.go's describeImage in returning on the first final
// event rather than continuing to scan for later ones.
func askResearcher(ctx context.Context, r *runner.Runner, msg *genai.Content) (text string, searchQueries []string, err error) {
	for event, err := range r.Run(ctx, "test_user", "test_session", msg, agent.RunConfig{}) {
		if err != nil {
			return "", nil, err
		}
		if !event.IsFinalResponse() {
			continue
		}
		if event.GroundingMetadata != nil {
			searchQueries = event.GroundingMetadata.WebSearchQueries
		}
		if event.Content == nil {
			return "", searchQueries, nil
		}
		for _, p := range event.Content.Parts {
			if p.Text != "" && !p.Thought {
				return p.Text, searchQueries, nil
			}
		}
		return "", searchQueries, nil
	}
	return "", nil, nil
}
