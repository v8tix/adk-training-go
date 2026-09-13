// Package supportanalyzer defines the Support Analyzer agent: given a
// customer support ticket, it returns a structured JSON analysis (category,
// sentiment, summary) instead of a plain-text reply. Importable by more than
// one cmd/ program — the CLI/launcher entrypoint in cmd/support-analyzer and
// the programmatic runner in cmd/support-analyzer-runner — the same way
// Python's lab separates agent.py (definition) from main.py (how you invoke
// it), expressed here as a package boundary instead of two files in one
// directory (Go can't have two func main()s in the same package).
package supportanalyzer

import (
	"embed"
	"io/fs"
	"log"

	"github.com/v8tix/adk-training-go/internal/infrastructure/prompts"
	"google.golang.org/adk/v2/agent"
	"google.golang.org/adk/v2/agent/llmagent"
	"google.golang.org/adk/v2/model"
	"google.golang.org/genai"
)

const (
	// PromptNamespace identifies this package's entries in the shared
	// prompts cache (internal/infrastructure/prompts).
	PromptNamespace = "support-analyzer"

	// OutputKey is the session-state key the agent's JSON analysis is saved
	// under (event.Actions.StateDelta[OutputKey]) — matches the Python lab's
	// literal "last_ticket_analysis".
	OutputKey = "last_ticket_analysis"
)

//go:embed prompts/*.md
var promptFS embed.FS

func init() {
	promptFiles, err := fs.Sub(promptFS, "prompts")
	if err != nil {
		log.Fatalf("resolving prompts directory: %v", err)
	}
	if err := prompts.Register(PromptNamespace, promptFiles, ".md"); err != nil {
		log.Fatalf("registering prompts: %v", err)
	}
}

// SupportAnalysis is the structured result the agent must return: a
// category, a sentiment, and a one-sentence summary of the customer's
// issue. It mirrors Schema field-for-field — the ADK Go SDK has no
// Pydantic-style derivation of one from the other, so both are kept in sync
// by hand.
type SupportAnalysis struct {
	Category  string `json:"category"`
	Sentiment string `json:"sentiment"`
	Summary   string `json:"summary"`
}

// Schema constrains the model's final response to a JSON object shaped like
// SupportAnalysis.
var Schema = &genai.Schema{
	Type: genai.TypeObject,
	Properties: map[string]*genai.Schema{
		"category":  {Type: genai.TypeString},
		"sentiment": {Type: genai.TypeString},
		"summary":   {Type: genai.TypeString},
	},
	Required: []string{"category", "sentiment", "summary"},
}

// BuildRootAgent constructs the Support Analyzer agent around llmModel,
// resolving its own instruction from the shared prompts cache so callers
// don't need to know the prompt's namespace or key.
func BuildRootAgent(llmModel model.LLM) (agent.Agent, error) {
	instruction, err := prompts.Get(PromptNamespace + "/support_analyzer_instruction")
	if err != nil {
		return nil, err
	}
	return llmagent.New(llmagent.Config{
		Name:         "support_analyzer_agent",
		Model:        llmModel,
		Description:  "Analyzes a customer support ticket into category, sentiment, and summary.",
		Instruction:  instruction,
		OutputSchema: Schema,
		OutputKey:    OutputKey,
	})
}
