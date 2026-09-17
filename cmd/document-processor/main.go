// Command document-processor runs a Document Processor agent: a four-step
// pipeline (extract, summarize, chart, report) that reads and writes
// versioned artifacts at every step. Run with `web --port 8080 webui api`
// for the Dev UI, `web --port 8080 api` alone for just the REST API, or
// `console` for a no-browser CLI chat — same shape as cmd/personal-tutor.
//
// Unlike every prior module's cmd/ program, this one MUST set
// ArtifactService explicitly: runner.New never defaults a nil
// ArtifactService (only cmd/launcher/web's own internal wiring does, and
// only for web mode), so console mode with no ArtifactService set gets a
// nil agent.Artifacts interface — any tool calling ctx.Artifacts().Save(...)
// against it panics. Confirmed live by tracing runner.go and
// cmd/launcher/console/console.go during this module's own Phase 1.
//
// The agent's own definition lives in internal/agents/documentprocessor —
// this file is just the CLI/launcher entrypoint.
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"

	"github.com/v8tix/adk-training-go/internal/agents/documentprocessor"
	"github.com/v8tix/adk-training-go/internal/infrastructure/llm"
	"google.golang.org/adk/v2/agent"
	"google.golang.org/adk/v2/artifact"
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
	fmt.Printf("📄 document-processor using %s\n", modelName)
	rootAgent, err := documentprocessor.BuildRootAgent(llmModel)
	if err != nil {
		log.Fatalf("building agent: %v", err)
	}
	config := &launcher.Config{
		AgentLoader:     agent.NewSingleLoader(rootAgent),
		ArtifactService: artifact.InMemoryService(),
	}
	l := universal.NewLauncher(console.NewLauncher(), web.NewLauncher(webui.NewLauncher(), api.NewLauncher()))
	if err := l.Execute(ctx, config, os.Args[1:]); err != nil {
		log.Fatalf("run failed: %v\n\n%s", err, l.CommandLineSyntax())
	}
}
