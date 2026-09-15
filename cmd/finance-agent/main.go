// Command finance-agent runs a secure finance agent that requires human
// confirmation before every investment, and escalates large investments to
// a supervisor. Run with `web --port 9091 webui -api_server_address
// http://localhost:9091/api api` for the Dev UI, or `console` for a
// no-browser CLI chat — same shape as cmd/calculator and cmd/memory.
//
// Unlike modules 12's cmd/research-assistant, this program needs no custom
// confirmation-driving code: the SDK's own console launcher
// (cmd/launcher/console) already recognizes the adk_request_confirmation
// FunctionCall the ADK framework emits for a functiontool.Config.RequireConfirmation-wrapped
// tool, and prompts interactively for yes/no — confirmed by reading
// cmd/launcher/console/hitl.go directly.
//
// Confirmed live (module-13) that this module needs no forced backend —
// both Ollama and Gemini genuinely support the confirmation round-trip and
// dynamic agent transfer, unlike modules 7/8/12's built-in-tool-driven
// cloud-only requirements.
//
// The agent's own definition lives in internal/agents/financeagent — this
// file is just the CLI/launcher entrypoint.
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"

	"github.com/v8tix/adk-training-go/internal/agents/financeagent"
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
	fmt.Printf("💰 finance-agent using %s\n", modelName)
	rootAgent, err := financeagent.BuildRootAgent(llmModel)
	if err != nil {
		log.Fatalf("building agent: %v", err)
	}
	config := &launcher.Config{AgentLoader: agent.NewSingleLoader(rootAgent)}
	l := universal.NewLauncher(console.NewLauncher(), web.NewLauncher(webui.NewLauncher(), api.NewLauncher()))
	if err := l.Execute(ctx, config, os.Args[1:]); err != nil {
		log.Fatalf("run failed: %v\n\n%s", err, l.CommandLineSyntax())
	}
}
