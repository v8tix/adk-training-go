// Package memory defines the Memory agent: it remembers a user's name
// across turns using two custom function tools, store_name and recall_name,
// both backed by agent.Context.State() — Go's direct equivalent of Python's
// ToolContext/tool_context.state. Unlike Python's opt-in tool_context
// parameter, agent.Context is unconditionally the first parameter of every
// custom function tool in this SDK (established in module-9) — this module
// is the first to actually use it for state instead of ignoring it.
//
// Confirmed live (module-10): state written via ctx.State().Set in one
// Run() call is genuinely visible via ctx.State().Get in a later, separate
// Run() call against the same session — real cross-turn memory, not just
// within-call state. Like internal/agents/calculator, this needs no cloud
// fallback; custom function tools (state access included) work through the
// local Ollama backend.
package memory

import (
	"embed"
	"io/fs"
	"log"

	"github.com/v8tix/adk-training-go/internal/infrastructure/prompts"
	"google.golang.org/adk/v2/agent"
	"google.golang.org/adk/v2/agent/llmagent"
	"google.golang.org/adk/v2/model"
	"google.golang.org/adk/v2/tool"
	"google.golang.org/adk/v2/tool/functiontool"
)

// PromptNamespace identifies this package's entries in the shared prompts
// cache (internal/infrastructure/prompts).
const PromptNamespace = "memory"

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

// BuildRootAgent constructs the Memory agent around llmModel, wrapping
// store_name and recall_name as custom function tools.
func BuildRootAgent(llmModel model.LLM) (agent.Agent, error) {
	instruction, err := prompts.Get(PromptNamespace + "/memory_instruction")
	if err != nil {
		return nil, err
	}

	storeTool, err := functiontool.New(functiontool.Config{
		Name:        "store_name",
		Description: "Saves the user's name to memory. Use this tool when the user tells you their name.",
	}, storeName)
	if err != nil {
		return nil, err
	}
	recallTool, err := functiontool.New(functiontool.Config{
		Name:        "recall_name",
		Description: "Retrieves the user's name from memory. Use this tool if the user asks who they are or what their name is.",
	}, recallName)
	if err != nil {
		return nil, err
	}

	return llmagent.New(llmagent.Config{
		Name:        "memory_agent",
		Model:       llmModel,
		Description: "A friendly assistant that remembers the user's name across turns.",
		Instruction: instruction,
		Tools:       []tool.Tool{storeTool, recallTool},
	})
}
