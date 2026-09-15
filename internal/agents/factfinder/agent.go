// Package factfinder defines an agent with one custom function tool backed
// by a real, independently-maintained third-party Go package
// (github.com/trietmn/go-wiki), demonstrating that Go needs no dedicated
// wrapper class for a third-party-backed tool — functiontool.New wraps it
// exactly like any hand-written function, since tool.Tool draws no
// distinction between the two. See docs/module-14/README.md for the full
// finding.
package factfinder

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
const PromptNamespace = "fact-finder"

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

// BuildRootAgent constructs the fact-finder agent around llmModel, wrapping
// lookupWikipedia as its one custom function tool.
func BuildRootAgent(llmModel model.LLM) (agent.Agent, error) {
	instruction, err := prompts.Get(PromptNamespace + "/factfinder_instruction")
	if err != nil {
		return nil, err
	}

	wikipediaTool, err := functiontool.New(functiontool.Config{
		Name:        "lookup_wikipedia",
		Description: "Looks up a topic on Wikipedia and returns a short summary.",
	}, lookupWikipedia)
	if err != nil {
		return nil, err
	}

	return llmagent.New(llmagent.Config{
		Name:        "fact_finder_agent",
		Model:       llmModel,
		Description: "An agent that can look up information on Wikipedia.",
		Instruction: instruction,
		Tools:       []tool.Tool{wikipediaTool},
	})
}
