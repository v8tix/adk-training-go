package main

import (
	"errors"
	"fmt"
	"go/version"
	"runtime/debug"
	"slices"

	"google.golang.org/genai"
)

// ErrMalformedGoVersion indicates a string isn't a valid Go toolchain
// version (go/version.IsValid), e.g. missing the "go" prefix entirely.
var ErrMalformedGoVersion = errors.New("malformed go version")

// goVersionMeetsMinimum reports whether v (as returned by runtime.Version())
// is at least min (e.g. "go1.27"). Uses the standard library's go/version
// package rather than hand-parsing the version string.
func goVersionMeetsMinimum(v, min string) (bool, error) {
	if !version.IsValid(v) {
		return false, fmt.Errorf("%w: %q", ErrMalformedGoVersion, v)
	}
	return version.Compare(v, min) >= 0, nil
}

// resolvedModuleVersion returns the version of the dependency at path as recorded
// in bi, and whether it was found.
func resolvedModuleVersion(bi *debug.BuildInfo, path string) (string, bool) {
	i := slices.IndexFunc(bi.Deps, func(m *debug.Module) bool { return m.Path == path })
	if i < 0 {
		return "", false
	}
	return bi.Deps[i].Version, true
}

// firstAnswerText returns the text of the first non-empty part that isn't a
// reasoning trace (thinking-capable models, e.g. Qwen3, emit their
// chain-of-thought as a separate Part with Thought=true).
func firstAnswerText(parts []*genai.Part) (string, bool) {
	i := slices.IndexFunc(parts, func(p *genai.Part) bool { return !p.Thought && p.Text != "" })
	if i < 0 {
		return "", false
	}
	return parts[i].Text, true
}
