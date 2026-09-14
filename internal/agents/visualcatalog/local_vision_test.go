package visualcatalog

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/v8tix/adk-training-go/internal/infrastructure/llm"
)

// TestLocalImageMIMEType is table-driven per this repo's own testing
// convention, and covers every branch localImageMIMEType has, including the
// unsupported-extension error path that TestDescribeImageLocally_Ollama
// (a single real .jpg call) never exercises.
func TestLocalImageMIMEType(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		want    string
		wantErr bool
	}{
		{name: "jpg", path: "photo.jpg", want: "image/jpeg"},
		{name: "jpeg", path: "photo.jpeg", want: "image/jpeg"},
		{name: "uppercase extension", path: "photo.JPG", want: "image/jpeg"},
		{name: "png", path: "photo.png", want: "image/png"},
		{name: "webp", path: "photo.webp", want: "image/webp"},
		{name: "gif", path: "photo.gif", want: "image/gif"},
		{name: "unsupported extension", path: "photo.bmp", wantErr: true},
		{name: "no extension", path: "photo", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := localImageMIMEType(tt.path)
			if tt.wantErr {
				if !errors.Is(err, ErrUnsupportedImageType) {
					t.Fatalf("localImageMIMEType(%q) error = %v, want errors.Is(err, ErrUnsupportedImageType)", tt.path, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("localImageMIMEType(%q) error = %v, want nil", tt.path, err)
			}
			if got != tt.want {
				t.Errorf("localImageMIMEType(%q) = %q, want %q", tt.path, got, tt.want)
			}
		})
	}
}

// TestDescribeImageLocally_Ollama is the bonus path's own test — no
// credential needed (unlike TestVisualCatalog_AnalyzesImage_Gemini), since
// it hits the local Ollama server directly. Skips (rather than fails) if
// that server isn't reachable, matching every other Ollama-backed test in
// this repo, and if TEST_BACKEND excludes Ollama.
func TestDescribeImageLocally_Ollama(t *testing.T) {
	if !llm.SelectedTestBackend().IncludesOllama() {
		t.Skip("skipping: TEST_BACKEND excludes ollama")
	}
	cfg := llm.LoadConfig()
	if !llm.OllamaReachable(cfg, 2*time.Second) {
		t.Skip("skipping: local Ollama server (" + cfg.OllamaBaseURL + ") is not reachable")
	}

	got, err := DescribeImageLocally(t.Context(), cfg.OllamaBaseURL, cfg.OllamaModel, "testdata/headphones.jpg")
	if err != nil {
		t.Fatalf("DescribeImageLocally() error = %v", err)
	}
	if got == "" {
		t.Fatal("DescribeImageLocally() returned no description")
	}
	// Same real, structural check as the Gemini test: proves the model
	// actually looked at the image, not generic filler.
	if !strings.Contains(strings.ToLower(got), "headphone") {
		t.Errorf("description = %q, want it to mention headphones (the actual subject of testdata/headphones.jpg)", got)
	}
}
