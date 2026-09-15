package factfinder

import (
	gowiki "github.com/trietmn/go-wiki"
	"google.golang.org/adk/v2/agent"
)

// summarySentences bounds how much of a Wikipedia article lookupWikipedia
// returns — enough for a model to summarize, not the full article.
const summarySentences = 5

func init() {
	// Wikimedia rate-limits go-wiki's own generic default User-Agent (shared
	// by every user of the package, so Wikimedia throttles it collectively)
	// — confirmed live: an unset User-Agent produces a real fetch failure
	// ("unable to fetch the results"). Set once, before any request, the
	// same fix Python's wikipedia package needs (wikipedia.set_user_agent).
	//
	// gowiki.UserAgent is a package-level variable, not per-instance — this
	// is only safe because this package is the only importer of go-wiki in
	// this repo, and every cmd/ binary here builds exactly one agent
	// package. If a second package in the same binary ever imports go-wiki
	// with a different User-Agent, whichever init() runs last wins; Go does
	// not guarantee an order between unrelated packages' init() functions.
	gowiki.SetUserAgent("adk-training-go-fact-finder/1.0 (https://github.com/v8tix/adk-training-go)")
}

// LookupWikipediaArgs is the input to lookupWikipedia.
type LookupWikipediaArgs struct {
	Query string `json:"query"`
}

// LookupWikipediaResult is the output of lookupWikipedia. Summary
// deliberately has no "omitempty" — a page with no extract (a stub,
// disambiguation, or file/category page) is a genuine, reachable
// gowiki.Summary success case with an empty string, confirmed by reading
// gowiki.Summary's own source; "omitempty" on that zero value would
// silently drop it from the JSON handed to the model, leaving a "success"
// result with no text at all — the same class of bug found and fixed in
// module-9's CalcResult.
type LookupWikipediaResult struct {
	Status  string `json:"status"`
	Summary string `json:"summary"`
	Error   string `json:"error,omitempty"`
}

// lookupWikipedia looks up query on Wikipedia via github.com/trietmn/go-wiki,
// a real, independently-maintained third-party Go package — not part of
// this repo or the ADK SDK. Wrapped identically to any hand-written
// function tool (see agent.go): Go's tool.Tool interface draws no
// distinction between a function you wrote and one that calls into a
// third-party package.
//
// A page-not-found or network failure returns a structured error result
// (not a bare Go error), matching module-9's divide-by-zero precedent —
// this is an ordinary, expected outcome for a lookup tool, not a plumbing
// failure the model can't reason about. go-wiki itself doesn't distinguish
// the two cases with a typed or sentinel error (confirmed by reading its
// source: a "page not exist" error and an underlying HTTP/network failure
// both surface as a plain error value), so this tool can't either — a
// known, accepted limitation of the upstream package, not a distinction
// this code chose not to make.
func lookupWikipedia(_ agent.Context, args LookupWikipediaArgs) (LookupWikipediaResult, error) {
	summary, err := gowiki.Summary(args.Query, summarySentences, -1, false, true)
	if err != nil {
		return LookupWikipediaResult{Status: "error", Error: err.Error()}, nil
	}
	return LookupWikipediaResult{Status: "success", Summary: summary}, nil
}
