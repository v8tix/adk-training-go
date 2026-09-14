// Command researcher runs an agent equipped with the ADK's built-in
// google_search tool, so it can answer questions about current events by
// actually searching the web. Run with `web --port 8080 webui api` for the
// Dev UI (both sub-launcher keywords must come after web's own flags, and
// `api` is required alongside `webui`), `web --port 8080 api` alone for just
// the REST API, or `console` for a no-browser CLI chat — same shape as
// cmd/support-analyzer.
//
// google_search is confirmed live (module-8) to require Gemini:
// model/openaimodel (the local Ollama backend's Go client) unconditionally
// rejects any non-function tool, including this built-in one, before ever
// reaching the network. cfg.ModelType is forced to gemini in code, not left
// to the shared .env default, so this can never silently hit that confirmed
// error path. Also confirmed live: the plain GOOGLE_AI_STUDIO_API_KEY path
// (no Vertex AI) is sufficient — see docs/module-8/README.md.
//
// The agent's own definition lives in internal/agents/researcher — this
// file is just the CLI/launcher entrypoint.
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"

	"github.com/v8tix/adk-training-go/internal/agents/researcher"
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
	cfg.ModelType = llm.ModelTypeGemini
	llmModel, modelName, err := llm.BuildModel(ctx, cfg)
	if err != nil {
		log.Fatalf("building model: %v", err)
	}
	fmt.Printf("🔎 researcher using %s\n", modelName)
	rootAgent, err := researcher.BuildRootAgent(llmModel)
	if err != nil {
		log.Fatalf("building agent: %v", err)
	}
	config := &launcher.Config{AgentLoader: agent.NewSingleLoader(rootAgent)}
	l := universal.NewLauncher(console.NewLauncher(), web.NewLauncher(webui.NewLauncher(), api.NewLauncher()))
	if err := l.Execute(ctx, config, os.Args[1:]); err != nil {
		log.Fatalf("run failed: %v\n\n%s", err, l.CommandLineSyntax())
	}
}
