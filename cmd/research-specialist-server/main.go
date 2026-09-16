// Command research-specialist-server exposes the research specialist agent
// as a real A2A HTTP service — the Go equivalent of Python's
// `uvicorn agent:a2a_app`.
//
// Confirmed live (module-21): unlike an earlier draft of this file, a plain
// net/http server was NOT the only option here — google.golang.org/adk/v2's
// own cmd/launcher/web/a2a package is a real web.Sublauncher composable into
// the exact same universal.NewLauncher shape every other cmd/ program in
// this repo uses, and it's what the SDK's own prod/full launcher
// configurations actually use for A2A. It derives the agent card from the
// root agent automatically (name, description, skills) and serves both the
// current (1.0) and legacy-compat (0.3) A2A protocols — more complete than
// a hand-rolled net/http version, not just more idiomatic.
//
// Run with `web --port 8001 a2a -a2a_agent_url http://localhost:8001`, then
// point cmd/a2a-orchestrator at the same address
// (RESEARCH_SPECIALIST_URL, default http://localhost:8001).
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"

	"github.com/v8tix/adk-training-go/internal/agents/researchspecialist"
	"github.com/v8tix/adk-training-go/internal/infrastructure/llm"
	"google.golang.org/adk/v2/agent"
	"google.golang.org/adk/v2/cmd/launcher"
	"google.golang.org/adk/v2/cmd/launcher/universal"
	"google.golang.org/adk/v2/cmd/launcher/web"
	"google.golang.org/adk/v2/cmd/launcher/web/a2a"
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
	fmt.Printf("🔬 research-specialist-server using %s\n", modelName)

	specialist, err := researchspecialist.BuildRootAgent(llmModel)
	if err != nil {
		log.Fatalf("building agent: %v", err)
	}

	config := &launcher.Config{AgentLoader: agent.NewSingleLoader(specialist)}
	l := universal.NewLauncher(web.NewLauncher(a2a.NewLauncher()))
	if err := l.Execute(ctx, config, os.Args[1:]); err != nil {
		log.Fatalf("run failed: %v\n\n%s", err, l.CommandLineSyntax())
	}
}
