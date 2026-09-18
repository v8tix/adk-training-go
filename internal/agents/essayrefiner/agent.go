// Package essayrefiner defines a cyclic workflow: a single
// workflow.DynamicNode runs a writer once, then loops a critic/refiner pair
// with a plain Go for loop, capped at maxIterations, breaking early once
// the critic approves.
//
// This is a direct extension of module-18's own confirmed
// workflow.NewDynamicNode/workflow.RunNode mechanism — no new SDK construct
// is needed for iteration; a Go for loop inside a DynamicFn body, calling
// RunNode repeatedly, is the exact equivalent of Python's for/while loop
// calling ctx.run_node() repeatedly.
//
// Confirmed live this module (temp/module-20/probe/main.go): the loop's own
// final result arrives on the terminal event authored by the root workflow
// agent's own name ("EssayRefiner", this package's own workflowagent.Config
// Name), with Content nil and Output set to the final story — not on any of
// the writer/critic/refiner's own chat-content events, whose last one in a
// real run can just as easily be the critic's own "APPROVED" reply. A
// caller wanting the clean final result reads Output from that specific
// event.
package essayrefiner

import (
	"embed"
	"fmt"
	"io/fs"
	"log"
	"strings"

	"github.com/v8tix/adk-training-go/internal/infrastructure/prompts"
	"google.golang.org/adk/v2/agent"
	"google.golang.org/adk/v2/agent/llmagent"
	"google.golang.org/adk/v2/agent/workflowagent"
	"google.golang.org/adk/v2/model"
	"google.golang.org/adk/v2/session"
	"google.golang.org/adk/v2/workflow"
)

// PromptNamespace identifies this package's entries in the shared prompts
// cache (internal/infrastructure/prompts).
const PromptNamespace = "essay-refiner"

// maxIterations caps the critic/refiner loop — a required safety limit for
// any iterative workflow whose exit condition depends on a model's own
// judgment, matching this course's own "Safety First" framing.
const maxIterations = 3

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

// BuildRootAgent constructs the writer/critic/refiner team and the dynamic
// node that orchestrates the refinement loop, wrapped as a plain
// agent.Agent via workflowagent.New so it can be run by a runner.Runner
// exactly like any other agent.
func BuildRootAgent(llmModel model.LLM) (agent.Agent, error) {
	writerInstruction, err := prompts.Get(PromptNamespace + "/writer_instruction")
	if err != nil {
		return nil, err
	}
	criticInstruction, err := prompts.Get(PromptNamespace + "/critic_instruction")
	if err != nil {
		return nil, err
	}
	refinerInstruction, err := prompts.Get(PromptNamespace + "/refiner_instruction")
	if err != nil {
		return nil, err
	}

	writer, err := llmagent.New(llmagent.Config{
		Name:        "writer",
		Model:       llmModel,
		Description: "Writes an initial short story draft.",
		Instruction: writerInstruction,
	})
	if err != nil {
		return nil, err
	}
	critic, err := llmagent.New(llmagent.Config{
		Name:        "critic",
		Model:       llmModel,
		Description: "Reviews a story and approves it or gives feedback.",
		Instruction: criticInstruction,
	})
	if err != nil {
		return nil, err
	}
	refiner, err := llmagent.New(llmagent.Config{
		Name:        "refiner",
		Model:       llmModel,
		Description: "Rewrites a story to address the critic's feedback.",
		Instruction: refinerInstruction,
	})
	if err != nil {
		return nil, err
	}

	writerNode, err := workflow.NewAgentNode(writer, workflow.NodeConfig{})
	if err != nil {
		return nil, err
	}
	criticNode, err := workflow.NewAgentNode(critic, workflow.NodeConfig{})
	if err != nil {
		return nil, err
	}
	refinerNode, err := workflow.NewAgentNode(refiner, workflow.NodeConfig{})
	if err != nil {
		return nil, err
	}

	refinementWorkflow := workflow.NewDynamicNode("refinement_workflow",
		func(ctx agent.Context, topic string, emit func(*session.Event) error) (string, error) {
			currentStory, err := workflow.RunNode[string](ctx, writerNode, fmt.Sprintf("Topic: %s", topic))
			if err != nil {
				return "", err
			}

			for range maxIterations {
				feedback, err := workflow.RunNode[string](ctx, criticNode, currentStory)
				if err != nil {
					return "", err
				}
				if strings.Contains(feedback, "APPROVED") {
					break
				}

				currentStory, err = workflow.RunNode[string](ctx, refinerNode,
					fmt.Sprintf("WORK:\n%s\n\nFEEDBACK:\n%s", currentStory, feedback))
				if err != nil {
					return "", err
				}
			}

			return currentStory, nil
		},
		workflow.NodeConfig{},
	)

	edges := []workflow.Edge{
		{From: workflow.Start, To: refinementWorkflow},
	}

	return workflowagent.New(workflowagent.Config{
		Name:        "EssayRefiner",
		Description: "Iteratively refines a short story with a critic/refiner loop.",
		Edges:       edges,
	})
}
