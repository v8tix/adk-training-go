// Package travelplanner defines a collaborative team: a coordinator
// delegates to two specialists whose llmagent.Config.Mode determines
// whether control returns automatically. No workflow.Workflow wrapper is
// involved — plain SubAgents with Mode set is sufficient, matching
// module-15's own established finding for LLM-driven delegation.
//
// Confirmed live this module (temp/module-19/probe/main.go): unlike
// Python's lab, which requires rerun_on_resume=True on every agent in the
// dispatch chain (its own lab.md warns omitting it on any of the three
// agents raises "ValueError: A node must have rerun_on_resume=True."), Go's
// llmagent.Config has no such field at all — the runner wraps every
// llmagent as a workflow.NewDynamicNode and sets NodeConfig.RerunOnResume
// automatically for any LlmAgent (runner/agent_node.go's newAgentNode), so
// there's nothing to configure here. A real two-turn conversation —
// flight_booker asking a clarifying question, then finishing and
// automatically returning control to travel_planner on the next turn —
// worked correctly with zero resumability configuration anywhere.
//
// Also confirmed live, by reading internal/workflowinternal/
// task_agent_tool.go and single_turn_tool.go directly: the tool a
// ModeTask/ModeSingleTurn sub-agent is exposed as to its parent is named
// after the sub-agent itself (e.g. "flight_booker"), not a compound name
// like Python's own README describes ("request_task_flight_booker").
package travelplanner

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
const PromptNamespace = "travel-planner"

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

// BuildRootAgent constructs the travel planning team: a weather checker
// (single_turn), a flight booker (task, allowing back-and-forth about
// preferences), and a coordinator that delegates to both and synthesizes
// the final plan.
func BuildRootAgent(llmModel model.LLM) (agent.Agent, error) {
	weatherInstruction, err := prompts.Get(PromptNamespace + "/weather_checker_instruction")
	if err != nil {
		return nil, err
	}
	flightInstruction, err := prompts.Get(PromptNamespace + "/flight_booker_instruction")
	if err != nil {
		return nil, err
	}
	plannerInstruction, err := prompts.Get(PromptNamespace + "/travel_planner_instruction")
	if err != nil {
		return nil, err
	}

	weatherChecker, err := llmagent.New(llmagent.Config{
		Name:        "weather_checker",
		Model:       llmModel,
		Description: "Provides a brief 3-day forecast for a destination.",
		Mode:        llmagent.ModeSingleTurn,
		Instruction: weatherInstruction,
	})
	if err != nil {
		return nil, err
	}
	flightBooker, err := llmagent.New(llmagent.Config{
		Name:        "flight_booker",
		Model:       llmModel,
		Description: "Books a flight, asking about preferences if not provided.",
		Mode:        llmagent.ModeTask,
		Instruction: flightInstruction,
	})
	if err != nil {
		return nil, err
	}

	return llmagent.New(llmagent.Config{
		Name:        "travel_planner",
		Model:       llmModel,
		Description: "Coordinates weather and flight booking into one travel plan.",
		Instruction: plannerInstruction,
		SubAgents:   []agent.Agent{weatherChecker, flightBooker},
	})
}
