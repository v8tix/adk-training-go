// Command support-analyzer runs an agent that reads a customer support
// ticket and returns a structured JSON analysis (category, sentiment,
// summary) instead of a plain-text reply. Run with `web --port 8080 webui`
// for the Dev UI (note: webui must be named after web's own flags) or
// `console` for a no-browser CLI chat.
//
// Structured output (llmagent.Config.OutputSchema) needs the repo's default
// OLLAMA_MODEL — a GGUF quantization, not this machine's faster MLX presets,
// which return 501 "structured output is unavailable" for it. See
// docs/module-4/README.md for the finding.
package main

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"log"
	"os"

	"github.com/joho/godotenv"

	"github.com/v8tix/adk-training-go/internal/infrastructure/llm"
	"github.com/v8tix/adk-training-go/internal/infrastructure/prompts"
	"google.golang.org/adk/v2/agent"
	"google.golang.org/adk/v2/agent/llmagent"
	"google.golang.org/adk/v2/cmd/launcher"
	"google.golang.org/adk/v2/cmd/launcher/console"
	"google.golang.org/adk/v2/cmd/launcher/universal"
	"google.golang.org/adk/v2/cmd/launcher/web"
	"google.golang.org/adk/v2/cmd/launcher/web/webui"
	"google.golang.org/adk/v2/model"
	"google.golang.org/genai"
)

const (
	promptNamespace = "support-analyzer"

	// outputKey is the session-state key the agent's JSON analysis is saved
	// under (event.Actions.StateDelta[outputKey]) — matches the Python lab's
	// literal "last_ticket_analysis".
	outputKey = "last_ticket_analysis"
)

//go:embed prompts/*.md
var promptFS embed.FS

func init() {
	promptFiles, err := fs.Sub(promptFS, "prompts")
	if err != nil {
		log.Fatalf("resolving prompts directory: %v", err)
	}
	if err := prompts.Register(promptNamespace, promptFiles, ".md"); err != nil {
		log.Fatalf("registering prompts: %v", err)
	}
}

// SupportAnalysis is the structured result the agent must return: a
// category, a sentiment, and a one-sentence summary of the customer's
// issue. It mirrors supportAnalysisSchema field-for-field — the ADK Go SDK
// has no Pydantic-style derivation of one from the other, so both are kept
// in sync by hand.
type SupportAnalysis struct {
	Category  string `json:"category"`
	Sentiment string `json:"sentiment"`
	Summary   string `json:"summary"`
}

// supportAnalysisSchema constrains the model's final response to a JSON
// object shaped like SupportAnalysis.
var supportAnalysisSchema = &genai.Schema{
	Type: genai.TypeObject,
	Properties: map[string]*genai.Schema{
		"category":  {Type: genai.TypeString},
		"sentiment": {Type: genai.TypeString},
		"summary":   {Type: genai.TypeString},
	},
	Required: []string{"category", "sentiment", "summary"},
}

func buildRootAgent(llmModel model.LLM, instruction string) (agent.Agent, error) {
	return llmagent.New(llmagent.Config{
		Name:         "support_analyzer_agent",
		Model:        llmModel,
		Description:  "Analyzes a customer support ticket into category, sentiment, and summary.",
		Instruction:  instruction,
		OutputSchema: supportAnalysisSchema,
		OutputKey:    outputKey,
	})
}

func main() {
	ctx := context.Background()
	if err := godotenv.Load(); err != nil {
		fmt.Printf("ℹ️  No .env file loaded (%v) — continuing with the current environment.\n", err)
	}
	cfg := llm.LoadConfig()
	llmModel, modelName, err := llm.BuildModel(ctx, cfg)
	if err != nil {
		log.Fatalf("building model: %v", err)
	}
	fmt.Printf("🎫 support-analyzer using %s\n", modelName)
	instruction, err := prompts.Get(promptNamespace + "/support_analyzer_instruction")
	if err != nil {
		log.Fatalf("loading prompt: %v", err)
	}
	rootAgent, err := buildRootAgent(llmModel, instruction)
	if err != nil {
		log.Fatalf("building agent: %v", err)
	}
	config := &launcher.Config{AgentLoader: agent.NewSingleLoader(rootAgent)}
	l := universal.NewLauncher(console.NewLauncher(), web.NewLauncher(webui.NewLauncher()))
	if err := l.Execute(ctx, config, os.Args[1:]); err != nil {
		log.Fatalf("run failed: %v\n\n%s", err, l.CommandLineSyntax())
	}
}
