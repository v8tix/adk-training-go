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
// the local Ollama server. It skips when TEST_BACKEND excludes Ollama (see
// llm.SelectedTestBackend), or when that server isn't reachable, so CI or
// another machine without this box on the network doesn't break the suite.
func TestVerifyConnectivity_LocalOllama(t *testing.T) {
	if !llm.SelectedTestBackend().IncludesOllama() {
		t.Skip("skipping: TEST_BACKEND excludes ollama")
	}
	if !ollamaReachable {
		t.Skip("skipping: local Ollama server (" + testConfig.OllamaBaseURL + ") is not reachable")
	}

	cfg := testConfig
	cfg.ModelType = llm.ModelTypeOllama
	if err := verifyConnectivity(t.Context(), cfg); err != nil {
		t.Fatalf("verifyConnectivity() error = %v, want a successful call to the local Ollama model", err)
	}
}

// TestVerifyConnectivity_Gemini makes a real call through the Gemini backend.
// It skips when TEST_BACKEND excludes Gemini, or when GOOGLE_AI_STUDIO_API_KEY
// isn't set, so nobody without a Gemini credential (the common case for this
// local-first course) has this test break their suite.
func TestVerifyConnectivity_Gemini(t *testing.T) {
	if !llm.SelectedTestBackend().IncludesGemini() {
		t.Skip("skipping: TEST_BACKEND excludes gemini")
	}
	if testConfig.GoogleAPIKey == "" {
		t.Skip("skipping: GOOGLE_AI_STUDIO_API_KEY is not set")
	}

	cfg := testConfig
	cfg.ModelType = llm.ModelTypeGemini
	if err := verifyConnectivity(t.Context(), cfg); err != nil {
		t.Fatalf("verifyConnectivity() error = %v, want a successful call to Gemini", err)
	}
}
