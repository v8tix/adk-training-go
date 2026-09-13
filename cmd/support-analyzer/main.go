// Command support-analyzer runs an agent that reads a customer support
// ticket and returns a structured JSON analysis (category, sentiment,
// summary) instead of a plain-text reply. Run with `web --port 8080 webui
// api` for the Dev UI (both sub-launcher keywords must come after web's own
// flags, and `api` is required alongside `webui` — the Dev UI's frontend
// calls into the REST API for everything beyond serving its static page),
// `web --port 8080 api` alone for just the REST API server with no Dev UI
// (see docs/module-5/README.md for its routes), or `console` for a
// no-browser CLI chat.
//
// The agent's own definition lives in internal/agents/supportanalyzer,
// shared with cmd/support-analyzer-runner (module-6) — this file is just the
// CLI/launcher entrypoint.
//
// Structured output (llmagent.Config.OutputSchema) needs the repo's default
// OLLAMA_MODEL — a GGUF quantization. Some other quantizations of the same
// model family return 501 "structured output is unavailable" for it. See
// docs/module-4/README.md for the finding.
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"

	"github.com/v8tix/adk-training-go/internal/agents/supportanalyzer"
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
	fmt.Printf("🎫 support-analyzer using %s\n", modelName)
	rootAgent, err := supportanalyzer.BuildRootAgent(llmModel)
	if err != nil {
		log.Fatalf("building agent: %v", err)
	}
	config := &launcher.Config{AgentLoader: agent.NewSingleLoader(rootAgent)}
	l := universal.NewLauncher(console.NewLauncher(), web.NewLauncher(webui.NewLauncher(), api.NewLauncher()))
	if err := l.Execute(ctx, config, os.Args[1:]); err != nil {
		log.Fatalf("run failed: %v\n\n%s", err, l.CommandLineSyntax())
	}
}
