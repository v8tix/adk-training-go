package factfinder

import (
	"encoding/json"
	"strings"
	"testing"

	"google.golang.org/adk/v2/agent"
)

// TestLookupWikipedia hits the real Wikipedia API — no mock. This
// package's entire value is being a thin, faithful wrapper around the real
// third-party client (github.com/trietmn/go-wiki), so a mocked test would
// verify nothing real, matching this repo's own "prefer real
// implementations over mocks" default. Skips (not fails) on a network or
// rate-limit error, matching this repo's external-dependency skip
// discipline.
func TestLookupWikipedia(t *testing.T) {
	mockCtx := agent.NewStrictContextMock(t.Context())

	t.Run("known page returns a real summary", func(t *testing.T) {
		got, err := lookupWikipedia(&mockCtx, LookupWikipediaArgs{Query: "Marie Curie"})
		if err != nil {
			t.Skipf("skipping: lookupWikipedia() error = %v — Wikipedia may be unreachable or rate-limited", err)
		}
		if got.Status != "success" {
			t.Skipf("skipping: Status = %q, Error = %q — Wikipedia may be unreachable or rate-limited", got.Status, got.Error)
		}
		if !strings.Contains(got.Summary, "Curie") {
			t.Errorf("Summary = %q, want it to mention %q", got.Summary, "Curie")
		}
	})

	t.Run("nonsense query returns a structured error, not a Go error", func(t *testing.T) {
		got, err := lookupWikipedia(&mockCtx, LookupWikipediaArgs{Query: "zzqxvbnm-not-a-real-topic-987654321"})
		if err != nil {
			t.Fatalf("lookupWikipedia() error = %v, want a structured error result instead", err)
		}
		if got.Status != "error" {
			t.Errorf("Status = %q, want %q for a nonexistent page", got.Status, "error")
		}
		if got.Error == "" {
			t.Error("Error is empty, want a real error message")
		}
	})
}

// TestLookupWikipediaResult_EmptySummaryIsMarshaled proves a real bug class
// the other tests can't catch: they assert on LookupWikipediaResult's Go
// struct field directly, never on what actually gets marshaled and handed
// to the model. An empty Summary is a legitimate "success" outcome —
// gowiki.Summary returns ("", nil) for a page with no extract (a stub,
// disambiguation, or file/category page) — not an absent one.
// LookupWikipediaResult.Summary must not have an "omitempty" tag, or a
// "success" result with a genuinely empty summary silently loses the key
// entirely, matching module-9's CalcResult.Result finding.
func TestLookupWikipediaResult_EmptySummaryIsMarshaled(t *testing.T) {
	result := LookupWikipediaResult{Status: "success", Summary: ""}

	raw, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("json.Marshal(%+v) error = %v", result, err)
	}

	var decoded map[string]any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("json.Unmarshal(%s) error = %v", raw, err)
	}
	if _, ok := decoded["summary"]; !ok {
		t.Errorf("marshaled JSON %s has no \"summary\" key for an empty summary — the model would see a success response with no summary field at all", raw)
	}
}
