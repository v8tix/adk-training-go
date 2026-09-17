// Package piiguardrail defines a small demo agent — no tools at all — whose
// only job is to reproduce a fixed, obviously-fake credit card number on a
// trigger phrase, giving the package's own PIIGuardrailPlugin
// (pii_guardrail_plugin.go) something real to intercept and block. Matches
// Python's own purpose-built "leak_agent" role exactly.
package piiguardrail

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
const PromptNamespace = "piiguardrail"

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

// BuildRootAgent constructs the demo "leak_agent" around llmModel. It has
// no tools — its instruction alone is enough to reproduce the trigger
// response the guardrail plugin is meant to catch.
func BuildRootAgent(llmModel model.LLM) (agent.Agent, error) {
	instruction, err := prompts.Get(PromptNamespace + "/leak_agent_instruction")
	if err != nil {
		return nil, err
	}

	return llmagent.New(llmagent.Config{
		Name:        "leak_agent",
		Model:       llmModel,
		Description: "A testing assistant that reproduces a fixed test string on request, for exercising a safety guardrail.",
		Instruction: instruction,
	})
}
