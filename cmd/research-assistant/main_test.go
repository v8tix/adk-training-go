package main

import (
	"os"
	"strings"
	"testing"

	"github.com/v8tix/adk-training-go/internal/infrastructure/llm"
)

var testConfig llm.Config

func TestMain(m *testing.M) {
	testConfig = llm.LoadConfig()
	os.Exit(m.Run())
}

// TestRunResearchPipeline_Gemini drives the real runResearchPipeline (the
// same function main() calls) end-to-end against real Gemini: no Ollama
// variant exists for this test — google_search requires Gemini, confirmed
// live in module-8, and this pipeline's first stage depends on it entirely.
func TestRunResearchPipeline_Gemini(t *testing.T) {
	if !llm.SelectedTestBackend().IncludesGemini() {
		t.Skip("skipping: TEST_BACKEND excludes gemini")
	}
	if testConfig.GoogleAPIKey == "" {
		t.Skip("skipping: GOOGLE_AI_STUDIO_API_KEY is not set")
	}

	cfg := testConfig
	cfg.ModelType = llm.ModelTypeGemini
	llmModel, _, err := llm.BuildModel(t.Context(), cfg)
	if err != nil {
		t.Fatalf("BuildModel() error = %v", err)
	}

	findings, report, err := runResearchPipeline(t.Context(), llmModel, "the most recent Nobel Prize in Physics winner")
	if err != nil && strings.Contains(err.Error(), "RESOURCE_EXHAUSTED") {
		t.Skipf("skipping: Gemini free-tier quota exhausted for today (%v) — external rate limit, not a code defect", err)
	}
	if err != nil {
		t.Fatalf("runResearchPipeline() error = %v", err)
	}

	if findings == "" {
		t.Error("runResearchPipeline() returned empty findings — the research agent produced no output")
	}
	if report == "" {
		t.Fatal("runResearchPipeline() returned an empty report")
	}
	if !strings.Contains(report, "# Research Report:") {
		t.Errorf("report = %q, want it to contain the formatted document's title line", report)
	}
	if !strings.Contains(report, "Generated:") {
		t.Errorf("report = %q, want it to contain a generated timestamp", report)
	}
	if !strings.Contains(report, "## Findings") {
		t.Errorf("report = %q, want it to contain the findings section", report)
	}
}
