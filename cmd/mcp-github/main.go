// Command mcp-github runs the GitHub MCP agent
// (internal/agents/mcpgithub), connected to GitHub's own hosted, remote MCP
// server over StreamableHTTP — module-27's "Bonus" transport, swapping
// cmd/mcp-filesystem's local stdio subprocess for a network call to a
// server GitHub runs and maintains. Run with `web --port 8080 webui api`
// for the Dev UI, `web --port 8080 api` alone for just the REST API, or
// `console` for a no-browser CLI chat — same shape as cmd/mcp-filesystem.
//
// Requires a real GitHub Personal Access Token in GITHUB_TOKEN (see
// .env.example) — create one at
// github.com/settings/personal-access-tokens/new with read-only repository
// access.
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"

	"github.com/v8tix/adk-training-go/internal/agents/mcpgithub"
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

	githubToken := os.Getenv("GITHUB_TOKEN")
	if githubToken == "" {
		log.Fatal("GITHUB_TOKEN is not set — create a read-only Personal Access Token at " +
			"github.com/settings/personal-access-tokens/new and set it in .env (see .env.example)")
	}

	cfg := llm.LoadConfig()
	llmModel, modelName, err := llm.BuildModel(ctx, cfg)
	if err != nil {
		log.Fatalf("building model: %v", err)
	}
	fmt.Printf("🐙 mcp-github using %s\n", modelName)

	rootAgent, err := mcpgithub.BuildRootAgent(llmModel, githubToken)
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
