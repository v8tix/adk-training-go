package main

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
			modelType: modelTypeOllama,
			wantName:  "qwen38-standard",
		},
		{
			// gemini.NewModel validates that an API key is present (non-empty)
			// at construction time, but doesn't call out to Google to check
			// it's valid — a placeholder is enough to prove the factory wires
			// the client together correctly.
			name:      "gemini",
			modelType: modelTypeGemini,
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
			cfg := loadConfig()
			cfg.ModelType = tt.modelType
			cfg.GoogleAPIKey = "test-placeholder-key"

			m, name, err := buildModel(t.Context(), cfg)

			if tt.wantErr {
				if !errors.Is(err, ErrUnknownModelType) {
					t.Fatalf("buildModel(%q) error = %v, want errors.Is(err, ErrUnknownModelType)", tt.modelType, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("buildModel(%q) unexpected error: %v", tt.modelType, err)
			}
			if m == nil {
				t.Fatalf("buildModel(%q) returned a nil model.LLM", tt.modelType)
			}
			if name != tt.wantName {
				t.Fatalf("buildModel(%q) name = %q, want %q", tt.modelType, name, tt.wantName)
			}
		})
	}
}

func TestKnownModelTypes(t *testing.T) {
	got := knownModelTypes()

	want := []string{modelTypeGemini, modelTypeOllama} // sorted
	if len(got) != len(want) {
		t.Fatalf("knownModelTypes() = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("knownModelTypes() = %v, want %v", got, want)
		}
	}
}
