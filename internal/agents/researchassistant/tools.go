package researchassistant

import (
	"fmt"
	"strings"
	"time"

	"google.golang.org/adk/v2/agent"
)

const (
	minFactLength   = 10
	defaultNumFacts = 5
)

// ExtractKeyFactsArgs is the input to extractKeyFacts.
type ExtractKeyFactsArgs struct {
	Text     string `json:"text"`
	NumFacts int    `json:"num_facts"`
}

// ExtractKeyFactsResult is the output of extractKeyFacts.
type ExtractKeyFactsResult struct {
	Status string   `json:"status"`
	Facts  []string `json:"facts"`
}

// extractKeyFacts splits text into sentences on '.' and returns the first
// numFacts sentences longer than minFactLength characters, trimmed of
// surrounding whitespace. Ported from Python's extract_key_facts.
func extractKeyFacts(_ agent.Context, args ExtractKeyFactsArgs) (ExtractKeyFactsResult, error) {
	numFacts := args.NumFacts
	if numFacts <= 0 {
		numFacts = defaultNumFacts
	}

	facts := make([]string, 0, numFacts)
	for _, sentence := range strings.Split(args.Text, ".") {
		trimmed := strings.TrimSpace(sentence)
		if len(trimmed) > minFactLength {
			facts = append(facts, trimmed)
		}
		if len(facts) == numFacts {
			break
		}
	}

	return ExtractKeyFactsResult{Status: "success", Facts: facts}, nil
}

// FormatResearchNotesArgs is the input to formatResearchNotes.
type FormatResearchNotesArgs struct {
	Topic    string `json:"topic"`
	Findings string `json:"findings"`
}

// FormatResearchNotesResult is the output of formatResearchNotes.
type FormatResearchNotesResult struct {
	Status   string `json:"status"`
	Document string `json:"document"`
}

// formatResearchNotes builds a structured Markdown-style report from a topic
// and its findings text, stamped with the current time. Ported from
// Python's format_research_notes.
func formatResearchNotes(_ agent.Context, args FormatResearchNotesArgs) (FormatResearchNotesResult, error) {
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	document := fmt.Sprintf(
		"# Research Report: %s\nGenerated: %s\n\n## Findings\n%s",
		args.Topic, timestamp, args.Findings,
	)
	return FormatResearchNotesResult{Status: "success", Document: document}, nil
}
