package llm

import (
	"os"
	"strings"
)

// TestBackend identifies which model backend(s) a live-call integration test
// should exercise. Every cmd/ package with both an *_Ollama and a *_Gemini
// live-call test reads this once, via SelectedTestBackend, instead of always
// running both and relying only on availability-based skipping — so running
// `TEST_BACKEND=gemini go test ./...` exercises only the Gemini path, even
// when a local Ollama server happens to be reachable too.
type TestBackend string

const (
	TestBackendOllama TestBackend = "ollama"
	TestBackendGemini TestBackend = "gemini"
	TestBackendBoth   TestBackend = "both"

	testBackendEnvVar = "TEST_BACKEND"
)

// SelectedTestBackend reads TEST_BACKEND ("ollama", "gemini", or "both",
// case-insensitive), defaulting to TestBackendBoth when unset or set to
// anything else — the safest default, since it matches every test's
// existing availability/credential-based skip behavior.
func SelectedTestBackend() TestBackend {
	switch strings.ToLower(os.Getenv(testBackendEnvVar)) {
	case string(TestBackendOllama):
		return TestBackendOllama
	case string(TestBackendGemini):
		return TestBackendGemini
	default:
		return TestBackendBoth
	}
}

// IncludesOllama reports whether the Ollama backend should be exercised
// under this selection.
func (b TestBackend) IncludesOllama() bool {
	return b == TestBackendOllama || b == TestBackendBoth
}

// IncludesGemini reports whether the Gemini backend should be exercised
// under this selection.
func (b TestBackend) IncludesGemini() bool {
	return b == TestBackendGemini || b == TestBackendBoth
}
