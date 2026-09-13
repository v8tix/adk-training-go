package prompts

import (
	"errors"
	"io/fs"
	"testing"
	"testing/fstest"
)

func TestRegisterAndGet(t *testing.T) {
	fsys := fstest.MapFS{
		"greeting.txt": &fstest.MapFile{Data: []byte("  Hello, world!  \n")},
		"farewell.txt": &fstest.MapFile{Data: []byte("Goodbye.")},
		"notes.md":     &fstest.MapFile{Data: []byte("not a prompt, wrong extension")},
	}

	if err := Register("test-register-and-get", fsys, ".txt"); err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	tests := []struct {
		name    string
		key     string
		want    string
		wantErr bool
	}{
		{
			name: "registered prompt, whitespace trimmed",
			key:  "test-register-and-get/greeting",
			want: "Hello, world!",
		},
		{
			name: "second registered prompt in the same namespace",
			key:  "test-register-and-get/farewell",
			want: "Goodbye.",
		},
		{
			name:    "wrong extension was not registered",
			key:     "test-register-and-get/notes",
			wantErr: true,
		},
		{
			name:    "unknown key",
			key:     "test-register-and-get/does-not-exist",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Get(tt.key)

			if tt.wantErr {
				if !errors.Is(err, ErrPromptNotFound) {
					t.Fatalf("Get(%q) error = %v, want errors.Is(err, ErrPromptNotFound)", tt.key, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("Get(%q) unexpected error: %v", tt.key, err)
			}
			if got != tt.want {
				t.Fatalf("Get(%q) = %q, want %q", tt.key, got, tt.want)
			}
		})
	}
}

// TestRegister_EmbedDirPrefixNeedsFsSub is a regression test for a real bug
// hit while wiring cmd/echo-agent: `//go:embed prompts/*.md` into an
// embed.FS keeps the "prompts/" path prefix (unlike a single-file
// `//go:embed path` into a string, which strips it) — Registering that FS
// directly produces "<namespace>/prompts/<name>", not "<namespace>/<name>",
// silently breaking every Get call written against the shorter key. fs.Sub
// is the fix; this proves both the failure mode and the fix, so a future
// regression here fails loudly instead of silently.
func TestRegister_EmbedDirPrefixNeedsFsSub(t *testing.T) {
	// Simulates the shape `//go:embed prompts/*.md` produces: "prompts/" is
	// a real directory entry inside the FS, not just a lookup prefix.
	embedShaped := fstest.MapFS{
		"prompts/echo_instruction.md": &fstest.MapFile{Data: []byte("hello")},
	}

	t.Run("registering the raw embed-shaped FS uses the wrong key", func(t *testing.T) {
		if err := Register("wrong-key-case", embedShaped, ".md"); err != nil {
			t.Fatalf("Register() error = %v", err)
		}
		if _, err := Get("wrong-key-case/echo_instruction"); !errors.Is(err, ErrPromptNotFound) {
			t.Fatalf("Get(short key) error = %v, want ErrPromptNotFound (proves the naive registration doesn't produce this key)", err)
		}
		got, err := Get("wrong-key-case/prompts/echo_instruction")
		if err != nil {
			t.Fatalf("Get(nested key) unexpected error: %v", err)
		}
		if got != "hello" {
			t.Fatalf("Get(nested key) = %q, want %q", got, "hello")
		}
	})

	t.Run("fs.Sub rooted at the embedded directory uses the intended key", func(t *testing.T) {
		sub, err := fs.Sub(embedShaped, "prompts")
		if err != nil {
			t.Fatalf("fs.Sub() error = %v", err)
		}
		if err := Register("correct-key-case", sub, ".md"); err != nil {
			t.Fatalf("Register() error = %v", err)
		}
		got, err := Get("correct-key-case/echo_instruction")
		if err != nil {
			t.Fatalf("Get(%q) unexpected error: %v", "correct-key-case/echo_instruction", err)
		}
		if got != "hello" {
			t.Fatalf("Get() = %q, want %q", got, "hello")
		}
	})
}

func TestRegister_SeparateNamespacesDoNotCollide(t *testing.T) {
	fsysA := fstest.MapFS{"instruction.txt": &fstest.MapFile{Data: []byte("agent A's instruction")}}
	fsysB := fstest.MapFS{"instruction.txt": &fstest.MapFile{Data: []byte("agent B's instruction")}}

	if err := Register("agent-a", fsysA, ".txt"); err != nil {
		t.Fatalf("Register(agent-a) error = %v", err)
	}
	if err := Register("agent-b", fsysB, ".txt"); err != nil {
		t.Fatalf("Register(agent-b) error = %v", err)
	}

	gotA, err := Get("agent-a/instruction")
	if err != nil {
		t.Fatalf("Get(agent-a/instruction) error = %v", err)
	}
	if gotA != "agent A's instruction" {
		t.Fatalf("Get(agent-a/instruction) = %q, want %q", gotA, "agent A's instruction")
	}

	gotB, err := Get("agent-b/instruction")
	if err != nil {
		t.Fatalf("Get(agent-b/instruction) error = %v", err)
	}
	if gotB != "agent B's instruction" {
		t.Fatalf("Get(agent-b/instruction) = %q, want %q", gotB, "agent B's instruction")
	}
}
