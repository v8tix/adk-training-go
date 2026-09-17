// Package observability defines the Observability agent: a single tool
// that can genuinely fail on demand, paired with a Runner-level Plugin
// (alerting_plugin.go) that intercepts real tool errors without touching
// this file's own logic at all — the real separation of concerns Python's
// own Plugin System exists for.
//
// The plugin is deliberately NOT attached here. Plugins are a
// Runner/launcher-level concern (runner.Config.PluginConfig,
// launcher.Config.PluginConfig), not an agent-level one — wiring one in
// would blur exactly the boundary this module is about. See
// cmd/observability-agent/main.go for where the plugin actually gets
// registered.
package observability

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
const PromptNamespace = "observability"

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

// BuildRootAgent constructs the Observability agent around llmModel,
// wrapping risky_operation as a custom function tool.
func BuildRootAgent(llmModel model.LLM) (agent.Agent, error) {
	instruction, err := prompts.Get(PromptNamespace + "/observability_instruction")
	if err != nil {
		return nil, err
	}

	riskyOperationTool, err := functiontool.New(functiontool.Config{
		Name:        "risky_operation",
		Description: "Performs an operation that can be made to fail, for testing error handling.",
	}, riskyOperation)
	if err != nil {
		return nil, err
	}

	return llmagent.New(llmagent.Config{
		Name:        "monitored_agent",
		Model:       llmModel,
		Description: "An agent whose risky_operation tool can be made to fail on demand, for observability testing.",
		Instruction: instruction,
		Tools:       []tool.Tool{riskyOperationTool},
	})
}
