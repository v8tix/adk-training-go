// Package supportrouter defines a dynamic-orchestration graph: a single
// workflow.DynamicNode classifies a request's sentiment, then plain Go
// if/else control flow — not a declared edge — picks which specialist
// handles it.
//
// Confirmed live this module (temp/module-18/probe/main.go): unlike
// module-17's classify_and_route FunctionNode, a workflow.NewDynamicNode's
// body genuinely can call workflow.RunNode imperatively — its context
// carries the sub-scheduler RunNode requires, which a plain
// FunctionNode/AgentNode context never does. So the classifier here is
// invoked directly from inside supportRouterWorkflow's own body, the same
// way Python's ctx.run_node(classifier, node_input) is invoked from inside
// its own @node-decorated function — this is the one place in this course
// so far where Go's mechanism maps almost exactly onto Python's.
//
// A genuine, confirmed parity with Python's own lab.md warning: RunNode's
// output type is a plain Go type assertion (rawOut.(OUT)) with no
// schema-aware conversion fallback, unlike FunctionNode's input coercion
// (module-17's finding). So even though classifier is built with
// OutputSchema, workflow.RunNode[map[string]any] — not a typed struct — is
// the only call shape that actually works; the orchestrator reads
// classification["sentiment"] directly, exactly as Python's own lab.md
// tells students to do with the equivalent Pydantic-schema-but-dict-at-
// runtime result.
package supportrouter

import (
	"embed"
	"io/fs"
	"log"

	"github.com/v8tix/adk-training-go/internal/infrastructure/prompts"
	"google.golang.org/adk/v2/agent"
	"google.golang.org/adk/v2/agent/llmagent"
	"google.golang.org/adk/v2/agent/workflowagent"
	"google.golang.org/adk/v2/model"
	"google.golang.org/adk/v2/session"
	"google.golang.org/adk/v2/workflow"
	"google.golang.org/genai"
)

// PromptNamespace identifies this package's entries in the shared prompts
// cache (internal/infrastructure/prompts).
const PromptNamespace = "support-router"

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

// sentimentSchema constrains the classifier's final response to a JSON
// object with a single sentiment field. No matching Go struct is declared
// — workflow.RunNode has no schema-aware conversion (confirmed live), so
// its result is read as map[string]any directly, not unmarshaled into a
// typed value.
var sentimentSchema = &genai.Schema{
	Type: genai.TypeObject,
	Properties: map[string]*genai.Schema{
		"sentiment": {Type: genai.TypeString, Enum: []string{"angry", "neutral", "happy"}},
	},
	Required: []string{"sentiment"},
}

// BuildRootAgent constructs the classifier + two specialists, wraps them as
// graph nodes, and builds a single DynamicNode whose body imperatively
// classifies and routes — wrapped as a plain agent.Agent via
// workflowagent.New so it can be run by a runner.Runner exactly like any
// other agent.
func BuildRootAgent(llmModel model.LLM) (agent.Agent, error) {
	classifierInstruction, err := prompts.Get(PromptNamespace + "/classifier_instruction")
	if err != nil {
		return nil, err
	}
	aiSupportInstruction, err := prompts.Get(PromptNamespace + "/ai_support_instruction")
	if err != nil {
		return nil, err
	}
	humanEscalationInstruction, err := prompts.Get(PromptNamespace + "/human_escalation_instruction")
	if err != nil {
		return nil, err
	}

	classifier, err := llmagent.New(llmagent.Config{
		Name:         "classifier",
		Model:        llmModel,
		Description:  "Classifies the sentiment of a support request.",
		Instruction:  classifierInstruction,
		OutputSchema: sentimentSchema,
	})
	if err != nil {
		return nil, err
	}
	aiSupport, err := llmagent.New(llmagent.Config{
		Name:        "ai_support",
		Model:       llmModel,
		Description: "Handles routine technical support requests.",
		Instruction: aiSupportInstruction,
	})
	if err != nil {
		return nil, err
	}
	humanEscalation, err := llmagent.New(llmagent.Config{
		Name:        "human_escalation",
		Model:       llmModel,
		Description: "Handles upset customers needing human escalation.",
		Instruction: humanEscalationInstruction,
	})
	if err != nil {
		return nil, err
	}

	classifierNode, err := workflow.NewAgentNode(classifier, workflow.NodeConfig{})
	if err != nil {
		return nil, err
	}
	aiSupportNode, err := workflow.NewAgentNode(aiSupport, workflow.NodeConfig{})
	if err != nil {
		return nil, err
	}
	humanEscalationNode, err := workflow.NewAgentNode(humanEscalation, workflow.NodeConfig{})
	if err != nil {
		return nil, err
	}

	supportRouterWorkflow := workflow.NewDynamicNode("support_router_workflow",
		func(ctx agent.Context, input string, emit func(*session.Event) error) (string, error) {
			classification, err := workflow.RunNode[map[string]any](ctx, classifierNode, input)
			if err != nil {
				return "", err
			}

			// neutral and happy are deliberately handled the same way — only
			// angry needs escalation to a human.
			chosen := aiSupportNode
			if classification["sentiment"] == "angry" {
				chosen = humanEscalationNode
			}

			return workflow.RunNode[string](ctx, chosen, input)
		},
		workflow.NodeConfig{},
	)

	edges := []workflow.Edge{
		{From: workflow.Start, To: supportRouterWorkflow},
	}

	return workflowagent.New(workflowagent.Config{
		Name:        "SupportSystem",
		Description: "Classifies a support request's sentiment and routes it to AI support or human escalation.",
		Edges:       edges,
	})
}
