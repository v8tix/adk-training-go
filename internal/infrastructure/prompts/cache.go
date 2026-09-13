// Package prompts is a shared cache for every cmd/ program's agent
// instructions. Go's //go:embed directive is directory-scoped — it can only
// reach files in or below the directory of the source file that declares
// it — so each cmd/ program still embeds its own local prompts/*.md files
// itself. What's shared here is the cache and lookup: every program
// registers its own embed.FS once at startup, and Get then serves any
// registered module's prompt by name. Prompts are static strings with no
// dynamic data to substitute, so this stays a plain cache — no template
// parsing.
package prompts

import (
	"errors"
	"fmt"
	"io/fs"
	"strings"
)

var (
	// ErrWalkingPrompts indicates walking a registered prompts directory failed.
	ErrWalkingPrompts = errors.New("error walking prompts directory")
	// ErrReadingPrompt indicates reading a prompt file's content failed.
	ErrReadingPrompt = errors.New("error reading prompt content")
	// ErrPromptNotFound indicates no prompt is registered under the given key.
	ErrPromptNotFound = errors.New("prompt not found")
)

// cache holds every registered module's prompts, keyed by "<namespace>/<name>"
// (name is the file's base name without its extension). Written once per
// namespace at startup via Register, read many times via Get.
var cache = make(map[string]string)

// Register loads every file with the given extension from fsys into the
// shared cache, keyed by "<namespace>/<name>". Call it once per cmd/
// program, passing that program's own //go:embed'd prompts directory — e.g.:
//
//	//go:embed prompts/*.md
//	var promptFS embed.FS
//
//	func init() {
//		// fs.Sub is required: //go:embed dir/*.ext keeps the "dir/" prefix
//		// inside the resulting embed.FS (unlike a single-file //go:embed
//		// path into a string, which has no such prefix) — without this,
//		// entries register as "<namespace>/prompts/<name>", not
//		// "<namespace>/<name>".
//		promptFiles, err := fs.Sub(promptFS, "prompts")
//		if err != nil { ... }
//		if err := prompts.Register("echo-agent", promptFiles, ".md"); err != nil { ... }
//	}
//
//	instruction, err := prompts.Get("echo-agent/echo_instruction")
func Register(namespace string, fsys fs.FS, ext string) error {
	return fs.WalkDir(fsys, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return fmt.Errorf("%w: %v", ErrWalkingPrompts, err)
		}
		if d.IsDir() || !strings.HasSuffix(path, ext) {
			return nil
		}

		data, err := fs.ReadFile(fsys, path)
		if err != nil {
			return fmt.Errorf("%w: %s: %v", ErrReadingPrompt, path, err)
		}

		name := strings.TrimSuffix(path, ext)
		cache[namespace+"/"+name] = strings.TrimSpace(string(data))
		return nil
	})
}

// Get returns the prompt registered under key ("<namespace>/<name>").
func Get(key string) (string, error) {
	p, ok := cache[key]
	if !ok {
		return "", fmt.Errorf("%w: %s", ErrPromptNotFound, key)
	}
	return p, nil
}
