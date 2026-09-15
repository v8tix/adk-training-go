// Command market-analyst runs an agent that answers currency-conversion
// questions using the real, free Frankfurter API, through a tool built
// declaratively via internal/infrastructure/openapitool rather than
// hand-written per endpoint. Run with `web --port 9091 webui
// -api_server_address http://localhost:9091/api api` for the Dev UI (see
// module-3's README for why the api_server_address flag is required
// whenever --port isn't 8080), or `console` for a no-browser CLI chat —
// same shape as cmd/calculator and cmd/memory.
//
// Like those two, this program does NOT force a backend: a plain
// function-declared tool (no built-in tool involved) is confirmed live
// (module-11) to work through the local Ollama default just as well as
// Gemini.
//
// The agent's own definition lives in internal/agents/marketanalyst — this
// file is just the CLI/launcher entrypoint.
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"

	"github.com/v8tix/adk-training-go/internal/agents/marketanalyst"
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
	fmt.Printf("💱 market-analyst using %s\n", modelName)
	rootAgent, err := marketanalyst.BuildRootAgent(llmModel)
	if err != nil {
		log.Fatalf("building agent: %v", err)
	}
	config := &launcher.Config{AgentLoader: agent.NewSingleLoader(rootAgent)}
	l := universal.NewLauncher(console.NewLauncher(), web.NewLauncher(webui.NewLauncher(), api.NewLauncher()))
	if err := l.Execute(ctx, config, os.Args[1:]); err != nil {
		log.Fatalf("run failed: %v\n\n%s", err, l.CommandLineSyntax())
	}
}
