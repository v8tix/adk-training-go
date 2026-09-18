// Package uiagent defines a deliberately trivial, tool-less agent — module-29's
// real lesson is the UI talking to this agent over the REST API's /run_sse
// endpoint, not the agent's own logic. Matches Python's own equally trivial
// "ui_agent" exactly.
package uiagent

import (
	"embed"
	"io/fs"
	"log"

	"github.com/v8tix/adk-training-go/internal/infrastructure/prompts"
	"google.golang.org/adk/v2/agent"
	"google.golang.org/adk/v2/agent/llmagent"
	"google.golang.org/adk/v2/model"
)

// PromptNamespace identifies this package's entries in the shared prompts
// cache (internal/infrastructure/prompts).
const PromptNamespace = "uiagent"

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

// BuildRootAgent constructs the UI Agent around llmModel — no tools, just a
// friendly instruction, since this module's own lesson is the client talking
// to it over HTTP/SSE, not what the agent itself can do.
func BuildRootAgent(llmModel model.LLM) (agent.Agent, error) {
	instruction, err := prompts.Get(PromptNamespace + "/ui_instruction")
	if err != nil {
		return nil, err
	}
	return llmagent.New(llmagent.Config{
		Name:        "ui_agent",
		Model:       llmModel,
		Description: "A helpful and friendly assistant, exercised through a custom HTML/JS chat UI over the REST API.",
		Instruction: instruction,
	})
}
