// Package streamingagent defines a deliberately trivial, tool-less agent —
// module-30's real lesson is the hand-written voice client talking to this
// agent over the REST API's /run_live WebSocket endpoint (bidirectional
// audio streaming), not the agent's own logic. Matches Python's own equally
// trivial "streaming_agent" exactly.
package streamingagent

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
const PromptNamespace = "streamingagent"

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

// BuildRootAgent constructs the Streaming Agent around llmModel — no tools,
// just a friendly instruction, since this module's own lesson is the
// bidirectional audio protocol the client speaks to it over, not what the
// agent itself can do. llmModel must be built via
// llm.ModelTypeVertexAILive — runner.RunLive rejects any model that doesn't
// expose a *genai.Client (see internal/llminternal/base_flow.go), which
// rules out every other backend this project supports.
func BuildRootAgent(llmModel model.LLM) (agent.Agent, error) {
	instruction, err := prompts.Get(PromptNamespace + "/streaming_instruction")
	if err != nil {
		return nil, err
	}
	return llmagent.New(llmagent.Config{
		Name:        "streaming_agent",
		Model:       llmModel,
		Description: "A friendly and talkative assistant, exercised through a hand-written voice client over the REST API's Live WebSocket endpoint.",
		Instruction: instruction,
	})
}
