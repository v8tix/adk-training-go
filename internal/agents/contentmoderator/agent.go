// Package contentmoderator defines the Content Moderator agent: a single
// generate_text tool wrapped by all six of llmagent.Config's callback
// slots — before/after agent (per-question response caching), before/after
// model (an input guardrail and output email redaction), and before/after
// tool (argument validation and output audit). Unlike modules 25/25.5's
// Plugins, every hook here is wired directly into BuildRootAgent itself —
// callbacks are node-scoped and part of an agent's own logic, not a
// Runner-level, cross-cutting concern.
package contentmoderator

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
const PromptNamespace = "contentmoderator"

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

// buildRootAgent is BuildRootAgent's own implementation, parameterized by
// an explicit *responseCache so a caller — this package's own live test
// included — can keep a handle on the cache's real hit counter. Production
// callers use BuildRootAgent, which supplies its own fresh cache.
func buildRootAgent(llmModel model.LLM, cache *responseCache) (agent.Agent, error) {
	instruction, err := prompts.Get(PromptNamespace + "/content_moderator_instruction")
	if err != nil {
		return nil, err
	}

	generateTextTool, err := functiontool.New(functiontool.Config{
		Name:        generateTextToolName,
		Description: "Generates short text (an essay) on a topic, for the given target word count.",
	}, generateText)
	if err != nil {
		return nil, err
	}

	return llmagent.New(llmagent.Config{
		Name:                 "secure_moderator",
		Model:                llmModel,
		Description:          "A content moderation assistant that caches answers, blocks unsafe requests, redacts sensitive output, and audits its own tool calls.",
		Instruction:          instruction,
		Tools:                []tool.Tool{generateTextTool},
		OutputKey:            outputKey,
		BeforeAgentCallbacks: []agent.BeforeAgentCallback{cache.beforeAgentCallback},
		AfterAgentCallbacks:  []agent.AfterAgentCallback{cache.afterAgentCallback},
		BeforeModelCallbacks: []llmagent.BeforeModelCallback{beforeModelCallback},
		AfterModelCallbacks:  []llmagent.AfterModelCallback{afterModelCallback},
		BeforeToolCallbacks:  []llmagent.BeforeToolCallback{beforeToolCallback},
		AfterToolCallbacks:   []llmagent.AfterToolCallback{afterToolCallback},
	})
}

// BuildRootAgent constructs the Content Moderator agent around llmModel,
// wrapping generate_text and registering all six callbacks with a fresh,
// process-local response cache.
func BuildRootAgent(llmModel model.LLM) (agent.Agent, error) {
	return buildRootAgent(llmModel, &responseCache{})
}
