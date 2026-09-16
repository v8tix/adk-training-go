// Package marketrouter defines a structured-routing graph: a classifier
// node decides which currency a request is about, then a small routing
// function directs the request to the matching specialist.
//
// Named marketrouter rather than marketanalyst — internal/agents/marketanalyst
// (module-11) already owns that name for an unrelated currency-conversion
// agent, even though this module's own Python lab is also titled "Market
// Analyst."
//
// Confirmed live this module (temp/module-17/probe/main.go): Go has no
// working equivalent of Python's ctx.run_node inside a plain node —
// workflow.RunNode requires a dynamic node's sub-scheduler
// (google.golang.org/adk/v2/workflow/run_node.go returns
// ErrInvalidRunNodeContext otherwise), which a plain FunctionNode/AgentNode
// never has. So the classifier here is its own graph node, connected via a
// plain edge, and classifyAndRoute receives its structured result as an
// ordinary typed function input — proven live to arrive pre-converted from
// the classifier's OutputSchema-constrained JSON into a MarketRoute value,
// no manual json.Unmarshal needed. The route itself is set by returning a
// *session.Event with Routes populated — confirmed in
// google.golang.org/adk/v2/workflow/function_node.go's own Run method — not
// by assigning a context field the way Python's ctx.route does.
package marketrouter

import (
	"embed"
	"fmt"
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
const PromptNamespace = "market-router"

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

// MarketRoute is the classifier's structured result: which currency the
// user's request is about. Mirrors routeSchema field-for-field — the ADK Go
// SDK has no Pydantic-style derivation of one from the other, so both are
// kept in sync by hand (matching internal/agents/supportanalyzer's
// precedent).
type MarketRoute struct {
	Currency string `json:"currency"`
}

// routeSchema constrains the classifier's final response to a JSON object
// shaped like MarketRoute, with Currency limited to the three routed
// currencies.
var routeSchema = &genai.Schema{
	Type: genai.TypeObject,
	Properties: map[string]*genai.Schema{
		"currency": {Type: genai.TypeString, Enum: []string{"USD", "EUR", "GBP"}},
	},
	Required: []string{"currency"},
}

// BuildRootAgent constructs the classifier → route → specialist graph,
// wrapped as a plain agent.Agent via workflowagent.New so it can be run by
// a runner.Runner exactly like any other agent.
func BuildRootAgent(llmModel model.LLM) (agent.Agent, error) {
	classifierInstruction, err := prompts.Get(PromptNamespace + "/classifier_instruction")
	if err != nil {
		return nil, err
	}
	usdInstruction, err := prompts.Get(PromptNamespace + "/usd_analyst_instruction")
	if err != nil {
		return nil, err
	}
	eurInstruction, err := prompts.Get(PromptNamespace + "/eur_analyst_instruction")
	if err != nil {
		return nil, err
	}
	gbpInstruction, err := prompts.Get(PromptNamespace + "/gbp_analyst_instruction")
	if err != nil {
		return nil, err
	}

	classifier, err := llmagent.New(llmagent.Config{
		Name:         "classifier",
		Model:        llmModel,
		Description:  "Classifies which currency a request is about.",
		Instruction:  classifierInstruction,
		OutputSchema: routeSchema,
	})
	if err != nil {
		return nil, err
	}
	usdAnalyst, err := llmagent.New(llmagent.Config{
		Name:        "usd_analyst",
		Model:       llmModel,
		Description: "Analyzes USD market conditions.",
		Instruction: usdInstruction,
	})
	if err != nil {
		return nil, err
	}
	eurAnalyst, err := llmagent.New(llmagent.Config{
		Name:        "eur_analyst",
		Model:       llmModel,
		Description: "Analyzes EUR market conditions.",
		Instruction: eurInstruction,
	})
	if err != nil {
		return nil, err
	}
	gbpAnalyst, err := llmagent.New(llmagent.Config{
		Name:        "gbp_analyst",
		Model:       llmModel,
		Description: "Analyzes GBP market conditions.",
		Instruction: gbpInstruction,
	})
	if err != nil {
		return nil, err
	}

	classifierNode, err := workflow.NewAgentNode(classifier, workflow.NodeConfig{})
	if err != nil {
		return nil, err
	}
	usdNode, err := workflow.NewAgentNode(usdAnalyst, workflow.NodeConfig{})
	if err != nil {
		return nil, err
	}
	eurNode, err := workflow.NewAgentNode(eurAnalyst, workflow.NodeConfig{})
	if err != nil {
		return nil, err
	}
	gbpNode, err := workflow.NewAgentNode(gbpAnalyst, workflow.NodeConfig{})
	if err != nil {
		return nil, err
	}

	classifyAndRouteNode := workflow.NewFunctionNode("classify_and_route", classifyAndRoute, workflow.NodeConfig{})

	edges := workflow.NewEdgeBuilder().
		Add(workflow.Start, classifierNode).
		Add(classifierNode, classifyAndRouteNode).
		AddRoutes(classifyAndRouteNode, map[string]workflow.Node{
			"USD": usdNode,
			"EUR": eurNode,
			"GBP": gbpNode,
		}).
		Build()

	return workflowagent.New(workflowagent.Config{
		Name:        "MarketRouter",
		Description: "Classifies a currency request and routes it to the matching specialist.",
		Edges:       edges,
	})
}

// classifyAndRoute reads the classifier's structured decision — already
// converted from JSON into route by the SDK's own schema-aware input
// coercion (workflow.NewFunctionNode's fallback path), confirmed live in
// this module's probe — and sets the routing decision by returning a
// *session.Event with Routes populated. This is the Go mechanism
// workflow.NewFunctionNode's own Run method documents for explicit routing:
// a function node whose return type is already *session.Event is yielded
// directly, so its Routes field selects the matching successor edge.
func classifyAndRoute(ctx agent.Context, route MarketRoute) (*session.Event, error) {
	if route.Currency == "" {
		return nil, fmt.Errorf("classify_and_route: classifier returned no currency")
	}
	ev := session.NewEvent(ctx, ctx.InvocationID())
	ev.Routes = []string{route.Currency}
	ev.Output = route.Currency
	return ev, nil
}
