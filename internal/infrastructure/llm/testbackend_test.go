package llm

import "testing"

func TestSelectedTestBackend(t *testing.T) {
	tests := []struct {
		name    string
		envVal  string
		want    TestBackend
		wantOl  bool
		wantGem bool
	}{
		{name: "unset defaults to both", envVal: "", want: TestBackendBoth, wantOl: true, wantGem: true},
		{name: "ollama", envVal: "ollama", want: TestBackendOllama, wantOl: true, wantGem: false},
		{name: "gemini", envVal: "gemini", want: TestBackendGemini, wantOl: false, wantGem: true},
		{name: "both, explicit", envVal: "both", want: TestBackendBoth, wantOl: true, wantGem: true},
		{name: "case-insensitive", envVal: "OLLAMA", want: TestBackendOllama, wantOl: true, wantGem: false},
		{name: "unrecognized value falls back to both", envVal: "vertex", want: TestBackendBoth, wantOl: true, wantGem: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv(testBackendEnvVar, tt.envVal)

			got := SelectedTestBackend()
			if got != tt.want {
				t.Fatalf("SelectedTestBackend() = %q, want %q", got, tt.want)
			}
			if got.IncludesOllama() != tt.wantOl {
				t.Errorf("IncludesOllama() = %v, want %v", got.IncludesOllama(), tt.wantOl)
			}
			if got.IncludesGemini() != tt.wantGem {
				t.Errorf("IncludesGemini() = %v, want %v", got.IncludesGemini(), tt.wantGem)
			}
		})
	}
}
