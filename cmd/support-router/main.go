// Command support-router runs a dynamic-orchestration graph: a single
// workflow.DynamicNode classifies a request's sentiment, then plain Go
// if/else control flow — not a declared edge — routes it to AI support or
// human escalation. Run with `web --port 9091 webui
// -api_server_address http://localhost:9091/api api` for the Dev UI, or
// `console` for a no-browser CLI chat — same shape as cmd/market-router.
//
// Confirmed live (module-18): a workflow-backed agent.Agent (built via
// agent/workflowagent.New) runs through the standard launcher exactly like
// any plain llmagent, whether its graph is static (module-16), dictionary-
// routed (module-17), or dynamic (this module) — no special wiring needed.
//
// The graph's own definition lives in internal/agents/supportrouter — this
// file is just the CLI/launcher entrypoint.
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"

	"github.com/v8tix/adk-training-go/internal/agents/supportrouter"
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
	fmt.Printf("🎧 support-router using %s\n", modelName)
	rootAgent, err := supportrouter.BuildRootAgent(llmModel)
	if err != nil {
		log.Fatalf("building agent: %v", err)
	}
	config := &launcher.Config{AgentLoader: agent.NewSingleLoader(rootAgent)}
	l := universal.NewLauncher(console.NewLauncher(), web.NewLauncher(webui.NewLauncher(), api.NewLauncher()))
	if err := l.Execute(ctx, config, os.Args[1:]); err != nil {
		log.Fatalf("run failed: %v\n\n%s", err, l.CommandLineSyntax())
	}
}
