package llm

import (
	"errors"
	"testing"
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

func TestKnownModelTypes(t *testing.T) {
	got := knownModelTypes()

	want := []string{ModelTypeGemini, ModelTypeOllama} // sorted
	if len(got) != len(want) {
		t.Fatalf("knownModelTypes() = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("knownModelTypes() = %v, want %v", got, want)
		}
	}
}
