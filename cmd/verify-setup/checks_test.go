package main

import (
	"errors"
	"runtime/debug"
	"testing"

	"google.golang.org/genai"
)

func TestGoVersionMeetsMinimum(t *testing.T) {
	tests := []struct {
		name    string
		version string
		want    bool
		wantErr bool
	}{
		{name: "exact minimum", version: "go1.25.0", want: true},
		{name: "above minimum patch", version: "go1.25.3", want: true},
		{name: "above minimum minor", version: "go1.27.1", want: true},
		{name: "above minimum major", version: "go2.0.0", want: true},
		{name: "below minimum minor", version: "go1.24.9", want: false},
		{name: "below minimum major", version: "go0.9.0", want: false},
		{name: "no patch component", version: "go1.25", want: true},
		{name: "valid but below minimum - no minor", version: "go1", want: false},
		{name: "malformed - no go prefix", version: "1.25.0", wantErr: true},
		{name: "malformed - non-numeric", version: "go1.x.0", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := goVersionMeetsMinimum(tt.version, "go1.25")

			if tt.wantErr {
				if !errors.Is(err, ErrMalformedGoVersion) {
					t.Fatalf("goVersionMeetsMinimum(%q) error = %v, want errors.Is(err, ErrMalformedGoVersion)", tt.version, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("goVersionMeetsMinimum(%q) unexpected error: %v", tt.version, err)
			}
			if got != tt.want {
				t.Fatalf("goVersionMeetsMinimum(%q) = %v, want %v", tt.version, got, tt.want)
			}
		})
	}
}

func TestResolvedModuleVersion(t *testing.T) {
	tests := []struct {
		name   string
		deps   []*debug.Module
		path   string
		want   string
		wantOK bool
	}{
		{
			name: "dependency present",
			deps: []*debug.Module{
				{Path: "github.com/joho/godotenv", Version: "v1.5.1"},
				{Path: "google.golang.org/adk/v2", Version: "v2.4.0"},
			},
			path:   "google.golang.org/adk/v2",
			want:   "v2.4.0",
			wantOK: true,
		},
		{
			name: "dependency absent",
			deps: []*debug.Module{
				{Path: "github.com/joho/godotenv", Version: "v1.5.1"},
			},
			path:   "google.golang.org/adk/v2",
			want:   "",
			wantOK: false,
		},
		{
			name:   "empty deps list",
			deps:   nil,
			path:   "google.golang.org/adk/v2",
			want:   "",
			wantOK: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bi := &debug.BuildInfo{Deps: tt.deps}

			got, ok := resolvedModuleVersion(bi, tt.path)

			if ok != tt.wantOK {
				t.Fatalf("resolvedModuleVersion() ok = %v, want %v", ok, tt.wantOK)
			}
			if got != tt.want {
				t.Fatalf("resolvedModuleVersion() = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestFirstAnswerText is the regression test for a real bug found while
// verifying this tool manually: qwen38-standard (thinking-capable) splits its
// response into a reasoning Part (Thought=true) and the real answer
// (Thought=false). Reading Parts[0] blindly returned the reasoning trace
// instead of the answer.
func TestFirstAnswerText(t *testing.T) {
	tests := []struct {
		name   string
		parts  []*genai.Part
		want   string
		wantOK bool
	}{
		{
			name: "thought part before the real answer",
			parts: []*genai.Part{
				{Thought: true, Text: "The user is greeting me, I should reply as instructed."},
				{Thought: false, Text: "ADK 2.0 is Ready!"},
			},
			want:   "ADK 2.0 is Ready!",
			wantOK: true,
		},
		{
			name: "no thought part",
			parts: []*genai.Part{
				{Thought: false, Text: "ADK 2.0 is Ready!"},
			},
			want:   "ADK 2.0 is Ready!",
			wantOK: true,
		},
		{
			name: "only a thought part, no answer yet",
			parts: []*genai.Part{
				{Thought: true, Text: "Still thinking..."},
			},
			want:   "",
			wantOK: false,
		},
		{
			name:   "no parts",
			parts:  nil,
			want:   "",
			wantOK: false,
		},
		{
			name: "non-thought part with empty text is skipped",
			parts: []*genai.Part{
				{Thought: false, Text: ""},
				{Thought: false, Text: "ADK 2.0 is Ready!"},
			},
			want:   "ADK 2.0 is Ready!",
			wantOK: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := firstAnswerText(tt.parts)

			if ok != tt.wantOK {
				t.Fatalf("firstAnswerText() ok = %v, want %v", ok, tt.wantOK)
			}
			if got != tt.want {
				t.Fatalf("firstAnswerText() = %q, want %q", got, tt.want)
			}
		})
	}
}
