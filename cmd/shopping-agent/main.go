// Command shopping-agent runs the Shopping Agent (internal/agents/mcpcart),
// which launches this repo's own cmd/cart-mcp-server as its MCP server
// subprocess. Run with `web --port 8080 webui api` for the Dev UI, `web
// --port 8080 api` alone for just the REST API, or `console` for a
// no-browser CLI chat — same shape as cmd/mcp-filesystem.
//
// The agent's own definition lives in internal/agents/mcpcart — this file
// is just the CLI/launcher entrypoint plus repo-root resolution.
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"

	"github.com/v8tix/adk-training-go/internal/agents/mcpcart"
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
	fmt.Printf("🛒 shopping-agent using %s\n", modelName)

	root, err := mcpcart.RepoRoot()
	if err != nil {
		log.Fatalf("resolving repo root: %v", err)
	}

	rootAgent, err := mcpcart.BuildRootAgent(llmModel, root)
	if err != nil {
		log.Fatalf("building agent: %v", err)
	}

	config := &launcher.Config{
		AgentLoader: agent.NewSingleLoader(rootAgent),
	}
	l := universal.NewLauncher(console.NewLauncher(), web.NewLauncher(webui.NewLauncher(), api.NewLauncher()))
	if err := l.Execute(ctx, config, os.Args[1:]); err != nil {
		log.Fatalf("run failed: %v\n\n%s", err, l.CommandLineSyntax())
	}
}
