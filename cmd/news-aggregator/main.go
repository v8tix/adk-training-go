// Command news-aggregator runs a hybrid static-orchestration graph: two
// research agents run in parallel, converge at a workflow.JoinNode, then a
// summarizer runs sequentially. Run with `web --port 9091 webui
// -api_server_address http://localhost:9091/api api` for the Dev UI, or
// `console` for a no-browser CLI chat — same shape as cmd/calculator.
//
// Confirmed live (module-16): a workflow-backed agent.Agent (built via
// agent/workflowagent.New) runs through the standard launcher exactly like
// any plain llmagent — no special wiring needed. No cloud fallback needed
// either, matching every plain-llmagent module's precedent since module-9.
//
// The graph's own definition lives in internal/agents/newsaggregator —
// this file is just the CLI/launcher entrypoint.
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"

	"github.com/v8tix/adk-training-go/internal/agents/newsaggregator"
	"github.com/v8tix/adk-training-go/internal/infrastructure/llm"
	"google.golang.org/adk/v2/agent"
	"google.golang.org/adk/v2/cmd/launcher"
	"google.golang.org/adk/v2/cmd/launcher/console"
	"google.golang.org/adk/v2/cmd/launcher/universal"
	"google.golang.org/adk/v2/cmd/launcher/web"
	"google.golang.org/adk/v2/cmd/launcher/web/api"
	"google.golang.org/adk/v2/cmd/launcher/web/webui"
)

func main() {
	ctx := context.Background()
	if err := godotenv.Load(); err != nil {
		fmt.Printf("ℹ️  No .env file loaded (%v) — continuing with the current environment.\n", err)
	}
	cfg := llm.LoadConfig()
	llmModel, modelName, err := llm.BuildModel(ctx, cfg)
	if err != nil {
		log.Fatalf("building model: %v", err)
	}
	fmt.Printf("📰 news-aggregator using %s\n", modelName)
	rootAgent, err := newsaggregator.BuildRootAgent(llmModel)
	if err != nil {
		log.Fatalf("building agent: %v", err)
	}
	config := &launcher.Config{AgentLoader: agent.NewSingleLoader(rootAgent)}
	l := universal.NewLauncher(console.NewLauncher(), web.NewLauncher(webui.NewLauncher(), api.NewLauncher()))
	if err := l.Execute(ctx, config, os.Args[1:]); err != nil {
		log.Fatalf("run failed: %v\n\n%s", err, l.CommandLineSyntax())
	}
}
