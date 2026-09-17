// Package visualcatalog defines the Visual Product Catalog agent: given a
// product photo, it writes a marketing description. Unlike
// internal/agents/supportanalyzer, this agent returns plain text, not a
// JSON-schema-constrained structure — Python's lab version doesn't set an
// output_schema either.
//
// This is a vision agent — it needs a model that can actually process an
// image Part. Confirmed live (module-7): the local Ollama backend's Go
// client (model/openaimodel) has no code path for an image Part at all — it
// errors before ever reaching the network, regardless of which model is
// loaded, even though the underlying model server handles the same image
// correctly when sent directly (confirmed via a raw HTTP call, bypassing
// this SDK's client). So this agent requires MODEL_TYPE=gemini — see
// cmd/visual-catalog and docs/module-07/README.md for the full finding.
package visualcatalog

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
const PromptNamespace = "visual-catalog"

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

// BuildRootAgent constructs the Visual Catalog agent around llmModel,
// resolving its own instruction from the shared prompts cache. llmModel must
// be able to process an image Part — see the package doc comment.
func BuildRootAgent(llmModel model.LLM) (agent.Agent, error) {
	instruction, err := prompts.Get(PromptNamespace + "/catalog_writer_instruction")
	if err != nil {
		return nil, err
	}
	return llmagent.New(llmagent.Config{
		Name:        "catalog_writer",
		Model:       llmModel,
		Description: "Analyzes a product image and writes a marketing description.",
		Instruction: instruction,
	})
}
