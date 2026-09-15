// Package marketanalyst defines the Global Market Analyst agent: it answers
// currency-conversion questions using the real, free Frankfurter API,
// through a tool built declaratively rather than hand-written per endpoint.
//
// The Go SDK has no OpenAPIToolset equivalent — confirmed exhaustively
// (module-11): no package, no type, no dependency anywhere in
// google.golang.org/adk/v2. internal/infrastructure/openapitool fills that
// gap by hand, using the same underlying tool.Tool/tool.Toolset contract
// this SDK already exposes — see that package's own doc comment for the
// full finding. Like internal/agents/calculator and
// internal/agents/memory, this agent needs no cloud fallback: a plain
// function-declared tool (no built-in tool involved) works through the
// local Ollama backend just as well as Gemini.
package marketanalyst

import (
	"embed"
	"io/fs"
	"log"

	"github.com/v8tix/adk-training-go/internal/infrastructure/openapitool"
	"github.com/v8tix/adk-training-go/internal/infrastructure/prompts"
	"google.golang.org/adk/v2/agent"
	"google.golang.org/adk/v2/agent/llmagent"
	"google.golang.org/adk/v2/model"
	"google.golang.org/adk/v2/tool"
)

// PromptNamespace identifies this package's entries in the shared prompts
// cache (internal/infrastructure/prompts).
const PromptNamespace = "market-analyst"

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

// frankfurterSpec describes the one Frankfurter operation this agent needs —
// Go's declarative equivalent of the OpenAPI spec dict Python's lab builds
// inline for the same "/latest" endpoint.
var frankfurterSpec = openapitool.OperationSpec{
	OperationID: "get_latest_rates",
	Summary:     "Get latest currency exchange rates. Use this whenever the user asks to convert between currencies.",
	BaseURL:     "https://api.frankfurter.dev/v1",
	Path:        "/latest",
	Parameters: []openapitool.ParamSpec{
		{Name: "amount", Type: "number", Description: "The amount to convert", Required: true},
		{Name: "from", Type: "string", Description: "The 3-letter currency code to convert from, e.g. USD", Required: true},
		{Name: "to", Type: "string", Description: "The 3-letter currency code to convert to, e.g. EUR", Required: true},
	},
}

// BuildRootAgent constructs the Market Analyst agent around llmModel,
// attaching the Frankfurter toolset via Toolsets — the closer Go equivalent
// of Python's OpenAPIToolset object (one value producing multiple tools),
// rather than spreading a tool slice into the plain Tools field.
func BuildRootAgent(llmModel model.LLM) (agent.Agent, error) {
	instruction, err := prompts.Get(PromptNamespace + "/market_analyst_instruction")
	if err != nil {
		return nil, err
	}

	toolset, err := openapitool.NewToolset("frankfurter", frankfurterSpec)
	if err != nil {
		return nil, err
	}

	return llmagent.New(llmagent.Config{
		Name:        "market_analyst_agent",
		Model:       llmModel,
		Description: "Answers currency-conversion questions using live exchange rates.",
		Instruction: instruction,
		Toolsets:    []tool.Toolset{toolset},
	})
}
