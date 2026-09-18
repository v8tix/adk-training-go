package llm

import (
	"errors"
	"testing"

	"google.golang.org/genai"
)

func TestBuildModel(t *testing.T) {
	tests := []struct {
		name      string
		modelType string
		wantName  string
		wantErr   bool
	}{
		{
			name:      "ollama",
			modelType: ModelTypeOllama,
			wantName:  "qwen3.8:27b",
		},
		{
			// gemini.NewModel validates that an API key is present (non-empty)
			// at construction time, but doesn't call out to Google to check
			// it's valid — a placeholder is enough to prove the factory wires
			// the client together correctly.
			name:      "gemini",
			modelType: ModelTypeGemini,
			wantName:  "gemini-3.5-flash",
		},
		{
			// Vertex AI Express Mode: a non-empty APIKey alone is sufficient
			// auth at construction time, confirmed against genai's own
			// client.go (no GCP project/location required when APIKey is set).
			name:      "vertexai",
			modelType: ModelTypeVertexAI,
			wantName:  "gemini-2.5-flash",
		},
		{
			// Live API model: same Express Mode auth path as "vertexai"
			// above, just a different model identifier and catalog.
			name:      "vertexai-live",
			modelType: ModelTypeVertexAILive,
			wantName:  "gemini-live-2.5-flash-native-audio",
		},
		{
			name:      "unknown model type",
			modelType: "not-a-real-backend",
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := LoadConfig()
			cfg.ModelType = tt.modelType
			cfg.GoogleAPIKey = "test-placeholder-key"
			cfg.VertexAIAPIKey = "test-placeholder-key"

			m, name, err := BuildModel(t.Context(), cfg)

			if tt.wantErr {
				if !errors.Is(err, ErrUnknownModelType) {
					t.Fatalf("BuildModel(%q) error = %v, want errors.Is(err, ErrUnknownModelType)", tt.modelType, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("BuildModel(%q) unexpected error: %v", tt.modelType, err)
			}
			if m == nil {
				t.Fatalf("BuildModel(%q) returned a nil model.LLM", tt.modelType)
			}
			if name != tt.wantName {
				t.Fatalf("BuildModel(%q) name = %q, want %q", tt.modelType, name, tt.wantName)
			}
		})
	}
}

func TestOllamaClientConfig_WiresAPIKeyAndBaseURL(t *testing.T) {
	tests := []struct {
		name    string
		baseURL string
	}{
		{name: "default base URL", baseURL: "http://localhost:11434/v1"},
		{name: "overridden base URL", baseURL: "http://example.invalid:9999/v1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := LoadConfig()
			cfg.OllamaBaseURL = tt.baseURL

			got := ollamaClientConfig(cfg)

			if got.APIKey != "ollama" {
				t.Errorf("APIKey = %q, want %q", got.APIKey, "ollama")
			}
			if got.BaseURL != tt.baseURL {
				t.Errorf("BaseURL = %q, want %q", got.BaseURL, tt.baseURL)
			}
			// option.RequestOption values are opaque closures with no public
			// way to inspect what they set (openai-go's RequestConfig lives in
			// an internal/ package) — asserting the count is the strongest
			// check available from outside that module. productionOllamaRetryOptions'
			// own values are covered directly by TestProductionOllamaRetryOptions.
			if len(got.Options) != len(productionOllamaRetryOptions()) {
				t.Errorf("len(Options) = %d, want %d (WithMaxRetries + WithMaxRetryDelay)", len(got.Options), len(productionOllamaRetryOptions()))
			}
		})
	}
}

func TestGeminiClientConfig_UsesProductionRetryOptions(t *testing.T) {
	cfg := LoadConfig()
	cfg.GoogleAPIKey = "test-placeholder-key"

	got := geminiClientConfig(cfg).HTTPOptions.RetryOptions
	want := productionRetryOptions()

	if got.MaxDelay == nil || want.MaxDelay == nil || *got.MaxDelay != *want.MaxDelay {
		t.Errorf("MaxDelay = %v, want %v", got.MaxDelay, want.MaxDelay)
	}
	if got.ExpBase == nil || want.ExpBase == nil || *got.ExpBase != *want.ExpBase {
		t.Errorf("ExpBase = %v, want %v", got.ExpBase, want.ExpBase)
	}
	if got.Jitter == nil || want.Jitter == nil || *got.Jitter != *want.Jitter {
		t.Errorf("Jitter = %v, want %v", got.Jitter, want.Jitter)
	}
}

func TestVertexAIClientConfig_UsesBackendAndProductionRetryOptions(t *testing.T) {
	cfg := LoadConfig()
	cfg.VertexAIAPIKey = "test-placeholder-key"

	got := vertexAIClientConfig(cfg)

	if got.Backend != genai.BackendVertexAI {
		t.Errorf("Backend = %v, want %v", got.Backend, genai.BackendVertexAI)
	}
	if got.APIKey != cfg.VertexAIAPIKey {
		t.Errorf("APIKey = %q, want %q", got.APIKey, cfg.VertexAIAPIKey)
	}

	want := productionRetryOptions()
	gotRetry := got.HTTPOptions.RetryOptions
	if gotRetry.MaxDelay == nil || want.MaxDelay == nil || *gotRetry.MaxDelay != *want.MaxDelay {
		t.Errorf("MaxDelay = %v, want %v", gotRetry.MaxDelay, want.MaxDelay)
	}
}

// TestVertexAIClientConfig_FallsBackToProjectLocationWithoutAPIKey proves the
// two auth paths are mutually exclusive and chosen correctly: when
// VertexAIAPIKey is empty, Project/Location pass through for genai.NewClient
// to resolve Application Default Credentials itself — this is a pure
// field-selection test, no live call or real ADC setup needed.
func TestVertexAIClientConfig_FallsBackToProjectLocationWithoutAPIKey(t *testing.T) {
	cfg := LoadConfig()
	cfg.VertexAIAPIKey = ""
	cfg.VertexAIProject = "test-project"
	cfg.VertexAILocation = "us-east1"

	got := vertexAIClientConfig(cfg)

	if got.APIKey != "" {
		t.Errorf("APIKey = %q, want empty when VertexAIAPIKey is unset", got.APIKey)
	}
	if got.Project != "test-project" {
		t.Errorf("Project = %q, want %q", got.Project, "test-project")
	}
	if got.Location != "us-east1" {
		t.Errorf("Location = %q, want %q", got.Location, "us-east1")
	}
}

func TestKnownModelTypes(t *testing.T) {
	got := knownModelTypes()

	want := []string{ModelTypeGemini, ModelTypeOllama, ModelTypeVertexAI, ModelTypeVertexAILive} // sorted
	if len(got) != len(want) {
		t.Fatalf("knownModelTypes() = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("knownModelTypes() = %v, want %v", got, want)
		}
	}
}
