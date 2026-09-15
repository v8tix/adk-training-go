package persistentagent

import (
	"testing"

	"github.com/v8tix/adk-training-go/internal/infrastructure/llm"
)

// TestBuildRootAgent_Constructs is a structural-only test — this package's
// real, meaningful behavior (persisting state across separate process
// instances) is proven by internal/infrastructure/redissession's
// conformance suite and cmd/persistent-agent's own integration test, not
// here.
func TestBuildRootAgent_Constructs(t *testing.T) {
	cfg := llm.LoadConfig()
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
