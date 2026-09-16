// Package newsaggregator defines a hybrid static-orchestration graph: two
// research agents run in parallel (fan-out from workflow.Start), converge
// at a workflow.JoinNode (fan-in), then a summarizer runs sequentially
// after both finish.
//
// Confirmed live this module: OutputKey + {key} instruction-template
// interpolation alone carries data from the parallel branches to the
// summarizer — the JoinNode's own aggregated map[string]any output (keyed
// by predecessor name) is never consumed directly by any node here. The
// JoinNode's real job is purely synchronization: it activates exactly once,
// only after every declared predecessor has completed (confirmed by
// reading google.golang.org/adk/v2/workflow/join_node.go's own doc
// comment), which is what makes it safe for the summarizer's instruction to
// assume both {tech_news} and {market_news} are already populated by the
// time it runs.
package newsaggregator

import (
	"embed"
	"io/fs"
	"log"

	"github.com/v8tix/adk-training-go/internal/infrastructure/prompts"
	"google.golang.org/adk/v2/agent"
	"google.golang.org/adk/v2/agent/llmagent"
	"google.golang.org/adk/v2/agent/workflowagent"
	"google.golang.org/adk/v2/model"
	"google.golang.org/adk/v2/workflow"
)

// PromptNamespace identifies this package's entries in the shared prompts
// cache (internal/infrastructure/prompts).
const PromptNamespace = "news-aggregator"

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

// BuildRootAgent constructs the full graph: two parallel researchers, a
// join barrier, and a sequential summarizer, wrapped as a plain agent.Agent
// via workflowagent.New so it can be run by a runner.Runner exactly like
// any other agent.
func BuildRootAgent(llmModel model.LLM) (agent.Agent, error) {
	techInstruction, err := prompts.Get(PromptNamespace + "/tech_researcher_instruction")
	if err != nil {
		return nil, err
	}
	marketInstruction, err := prompts.Get(PromptNamespace + "/market_researcher_instruction")
	if err != nil {
		return nil, err
	}
	summarizerInstruction, err := prompts.Get(PromptNamespace + "/summarizer_instruction")
	if err != nil {
		return nil, err
	}

	techResearcher, err := llmagent.New(llmagent.Config{
		Name:        "tech_researcher",
		Model:       llmModel,
		Description: "Researches recent AI and robotics headlines.",
		Instruction: techInstruction,
		OutputKey:   "tech_news",
	})
	if err != nil {
		return nil, err
	}
	marketResearcher, err := llmagent.New(llmagent.Config{
		Name:        "market_researcher",
		Model:       llmModel,
		Description: "Researches recent stock market headlines.",
		Instruction: marketInstruction,
		OutputKey:   "market_news",
	})
	if err != nil {
		return nil, err
	}
	summarizer, err := llmagent.New(llmagent.Config{
		Name:        "summarizer",
		Model:       llmModel,
		Description: "Combines tech and market research into one newsletter.",
		Instruction: summarizerInstruction,
	})
	if err != nil {
		return nil, err
	}

	techNode, err := workflow.NewAgentNode(techResearcher, workflow.NodeConfig{})
	if err != nil {
		return nil, err
	}
	marketNode, err := workflow.NewAgentNode(marketResearcher, workflow.NodeConfig{})
	if err != nil {
		return nil, err
	}
	summarizerNode, err := workflow.NewAgentNode(summarizer, workflow.NodeConfig{})
	if err != nil {
		return nil, err
	}
	syncer := workflow.NewJoinNode("news_sync")

	edges := []workflow.Edge{
		{From: workflow.Start, To: techNode},
		{From: workflow.Start, To: marketNode},
		{From: techNode, To: syncer},
		{From: marketNode, To: syncer},
		{From: syncer, To: summarizerNode},
	}

	return workflowagent.New(workflowagent.Config{
		Name:        "NewsSystem",
		Description: "A hybrid fan-out/fan-in/sequential news aggregator.",
		Edges:       edges,
	})
}
