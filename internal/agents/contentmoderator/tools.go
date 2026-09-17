package contentmoderator

import (
	"fmt"

	"google.golang.org/adk/v2/agent"
)

// GenerateTextArgs holds the topic and length for a generated essay.
type GenerateTextArgs struct {
	Topic     string `json:"topic" jsonschema:"the topic to write about"`
	WordCount int    `json:"word_count" jsonschema:"the target word count"`
}

// GenerateTextResult is generate_text's result.
type GenerateTextResult struct {
	Status string `json:"status"`
	Text   string `json:"text"`
}

// generateText is a trivial demo tool — matching Python's own lab exactly,
// it doesn't actually generate real text. This module's own point is the
// callbacks that surround the call (argument validation, output audit),
// not the tool's own logic.
func generateText(_ agent.Context, args GenerateTextArgs) (GenerateTextResult, error) {
	return GenerateTextResult{
		Status: "success",
		Text:   fmt.Sprintf("A %d-word essay on %s...", args.WordCount, args.Topic),
	}, nil
}
