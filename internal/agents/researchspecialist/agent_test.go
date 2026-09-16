package researchspecialist

import (
	"testing"

	"github.com/v8tix/adk-training-go/internal/infrastructure/llm"
)

// TestBuildRootAgent_Constructs is a structural regression guard,
// independent of any live call.
func TestBuildRootAgent_Constructs(t *testing.T) {
	cfg := llm.LoadConfig()
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
