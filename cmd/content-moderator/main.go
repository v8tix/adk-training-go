// Command content-moderator runs the Content Moderator agent, with all six
// of llmagent.Config's callback slots wired directly in
// internal/agents/contentmoderator.BuildRootAgent. Run with `web --port
// 8080 webui api` for the Dev UI, `web --port 8080 api` alone for just the
// REST API, or `console` for a no-browser CLI chat — same shape as
// cmd/pii-guardrail.
//
// Unlike cmd/observability-agent and cmd/pii-guardrail, this program needs
// NO PluginConfig at all — every hook here is a callback, registered
// directly on the agent itself (a node-level concern), not a Runner-level
// Plugin (an app-level concern). See internal/agents/contentmoderator's
// own package doc comment for the real architectural distinction.
//
// The agent's own definition lives in internal/agents/contentmoderator —
// this file is just the CLI/launcher entrypoint.
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"

	"github.com/v8tix/adk-training-go/internal/agents/contentmoderator"
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
	fmt.Printf("🧯 content-moderator using %s\n", modelName)
	rootAgent, err := contentmoderator.BuildRootAgent(llmModel)
	if err != nil {
		log.Fatalf("building agent: %v", err)
	}

	config := &launcher.Config{AgentLoader: agent.NewSingleLoader(rootAgent)}
	l := universal.NewLauncher(console.NewLauncher(), web.NewLauncher(webui.NewLauncher(), api.NewLauncher()))
	if err := l.Execute(ctx, config, os.Args[1:]); err != nil {
		log.Fatalf("run failed: %v\n\n%s", err, l.CommandLineSyntax())
	}
}
