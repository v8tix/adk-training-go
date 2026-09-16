// Package a2aorchestrator defines the local coordinator that delegates
// research requests to a remote research_specialist reachable over the A2A
// protocol.
//
// Confirmed live this module (temp/module-21/probe/main.go): registering a
// remoteagent/v2.NewA2A agent via plain SubAgents makes it a real
// transfer_to_agent target, exactly like a local ModeChat sub-agent — the
// coordinator's model calls transfer_to_agent, and the remote agent's own
// response comes back over a genuine network round trip, attributed to its
// own distinct event author. Unlike Python's RemoteA2aAgent, which exposes
// a narrower mode field (only "task" or None), Go's A2AConfig has no Mode
// field at all — AllowTransferToAgent is a different, unrelated setting
// (whether a transfer intent set by the remote agent's own model is
// honored locally), not a mode-selection mechanism.
package a2aorchestrator

import (
	"embed"
	"io/fs"
	"log"

	"github.com/v8tix/adk-training-go/internal/infrastructure/prompts"
	"google.golang.org/adk/v2/agent"
	"google.golang.org/adk/v2/agent/llmagent"
	remoteagentv2 "google.golang.org/adk/v2/agent/remoteagent/v2"
	"google.golang.org/adk/v2/model"
)

// PromptNamespace identifies this package's entries in the shared prompts
// cache (internal/infrastructure/prompts).
const PromptNamespace = "a2a-orchestrator"

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

// BuildRootAgent constructs the local coordinator, registering the remote
// research_specialist (reachable at specialistBaseURL) as a plain
// SubAgents entry.
func BuildRootAgent(llmModel model.LLM, specialistBaseURL string) (agent.Agent, error) {
	instruction, err := prompts.Get(PromptNamespace + "/coordinator_instruction")
	if err != nil {
		return nil, err
	}

	remoteResearcher, err := remoteagentv2.NewA2A(remoteagentv2.A2AConfig{
		Name:              "research_specialist",
		AgentCardProvider: remoteagentv2.NewAgentCardProvider(specialistBaseURL),
	})
	if err != nil {
		return nil, err
	}

	return llmagent.New(llmagent.Config{
		Name:        "a2a_orchestrator",
		Model:       llmModel,
		Description: "Delegates research requests to a remote specialist over A2A.",
		Instruction: instruction,
		SubAgents:   []agent.Agent{remoteResearcher},
	})
}
