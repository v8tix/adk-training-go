// Command fact-finder runs an agent equipped with one custom function tool,
// lookup_wikipedia, wrapping a real third-party Go package
// (github.com/trietmn/go-wiki). Run with `web --port 9091 webui
// -api_server_address http://localhost:9091/api api` for the Dev UI, or
// `console` for a no-browser CLI chat — same shape as cmd/calculator.
//
// Confirmed live (module-14): a custom function tool backed by a
// third-party Go package needs no special wrapper — functiontool.New wraps
// it exactly like any hand-written function — and works through the local
// Ollama default just as well as Gemini, no cloud fallback needed, matching
// module-9's own calculator precedent.
//
// The agent's own definition lives in internal/agents/factfinder — this
// file is just the CLI/launcher entrypoint.
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"

	"github.com/v8tix/adk-training-go/internal/agents/factfinder"
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
	fmt.Printf("📚 fact-finder using %s\n", modelName)
	rootAgent, err := factfinder.BuildRootAgent(llmModel)
	if err != nil {
		log.Fatalf("building agent: %v", err)
	}
	config := &launcher.Config{AgentLoader: agent.NewSingleLoader(rootAgent)}
	l := universal.NewLauncher(console.NewLauncher(), web.NewLauncher(webui.NewLauncher(), api.NewLauncher()))
	if err := l.Execute(ctx, config, os.Args[1:]); err != nil {
		log.Fatalf("run failed: %v\n\n%s", err, l.CommandLineSyntax())
	}
}
