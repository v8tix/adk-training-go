// Command travel-planner runs a collaborative team: a coordinator delegates
// to a weather checker (single_turn) and a flight booker (task, allowing
// back-and-forth about preferences across turns), then synthesizes both
// into one plan. Run with `web --port 9091 webui
// -api_server_address http://localhost:9091/api api` for the Dev UI, or
// `console` for a no-browser CLI chat — same shape as cmd/support-router.
//
// Confirmed live (module-19): no Workflow/workflowagent wrapper is
// involved — plain SubAgents with llmagent.Config.Mode set is sufficient
// for mode-driven automatic call-and-return, matching module-15's own
// finding that a plain llmagent with SubAgents needs no workflow wrapper
// for LLM-driven delegation.
//
// The team's own definition lives in internal/agents/travelplanner — this
// file is just the CLI/launcher entrypoint.
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"

	"github.com/v8tix/adk-training-go/internal/agents/travelplanner"
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
	fmt.Printf("🧳 travel-planner using %s\n", modelName)
	rootAgent, err := travelplanner.BuildRootAgent(llmModel)
	if err != nil {
		log.Fatalf("building agent: %v", err)
	}
	config := &launcher.Config{AgentLoader: agent.NewSingleLoader(rootAgent)}
	l := universal.NewLauncher(console.NewLauncher(), web.NewLauncher(webui.NewLauncher(), api.NewLauncher()))
	if err := l.Execute(ctx, config, os.Args[1:]); err != nil {
		log.Fatalf("run failed: %v\n\n%s", err, l.CommandLineSyntax())
	}
}
