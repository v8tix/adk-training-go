// Package persistentagent defines a minimal agent with no tools, mirroring
// Python's PersistentAgent — the lesson in module-13_5 is entirely about the
// Runner's storage layer (a custom session.Service), not the agent's own
// logic, so the agent itself is as simple as module-3's echo agent.
package persistentagent

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
const PromptNamespace = "persistent-agent"

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

// BuildRootAgent constructs the persistent agent around llmModel — no
// tools, matching Python's PersistentAgent exactly.
func BuildRootAgent(llmModel model.LLM) (agent.Agent, error) {
	instruction, err := prompts.Get(PromptNamespace + "/persistent_instruction")
	if err != nil {
		return nil, err
	}
	return llmagent.New(llmagent.Config{
		Name:        "PersistentAgent",
		Model:       llmModel,
		Description: "A helpful assistant that remembers the user's favorite color.",
		Instruction: instruction,
	})
}
