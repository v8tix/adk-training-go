// Package researchspecialist defines the research specialist agent exposed
// as a standalone A2A service — the node cmd/research-specialist-server
// serves over real HTTP, and internal/agents/a2aorchestrator calls as a
// remote sub-agent.
package researchspecialist

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
const PromptNamespace = "research-specialist"

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

// BuildRootAgent constructs the research specialist. cmd/research-
// specialist-server exposes it over real A2A HTTP via the standard launcher
// composed with the SDK's own a2a sublauncher, rather than the
// console/webui/api combination every chat-agent cmd/ program in this repo
// uses.
func BuildRootAgent(llmModel model.LLM) (agent.Agent, error) {
	instruction, err := prompts.Get(PromptNamespace + "/specialist_instruction")
	if err != nil {
		return nil, err
	}

	return llmagent.New(llmagent.Config{
		Name:        "research_specialist",
		Model:       llmModel,
		Description: "A remote research specialist reachable over A2A.",
		Instruction: instruction,
	})
}
