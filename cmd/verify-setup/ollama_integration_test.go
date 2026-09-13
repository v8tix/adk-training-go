package main

import (
	"os"
	"testing"
	"time"

	"github.com/v8tix/adk-training-go/internal/infrastructure/llm"
)

// testConfig and ollamaReachable are set once by TestMain, so individual
// tests don't each reload config or pay the probe-dial cost.
var (
	testConfig      llm.Config
	ollamaReachable bool
)

func TestMain(m *testing.M) {
	testConfig = llm.LoadConfig()
	ollamaReachable = llm.OllamaReachable(testConfig, 2*time.Second)
	os.Exit(m.Run())
}

// TestVerifyConnectivity_LocalOllama makes a real call through openaimodel to
// the local Ollama server. It skips rather than fails when that server isn't
// reachable, so CI or another machine without this box on the network doesn't
// break the suite.
func TestVerifyConnectivity_LocalOllama(t *testing.T) {
	if !ollamaReachable {
		t.Skip("skipping: local Ollama server (" + testConfig.OllamaBaseURL + ") is not reachable")
	}

	if err := verifyConnectivity(t.Context(), testConfig); err != nil {
		t.Fatalf("verifyConnectivity() error = %v, want a successful call to the local Ollama model", err)
	}
}
