// Command pii-guardrail runs the "leak_agent" demo agent with a real
// Fail-Closed safety Plugin (internal/agents/piiguardrail's own
// PIIGuardrailPlugin) registered via launcher.Config.PluginConfig. Run
// with `web --port 8080 webui api` for the Dev UI, `web --port 8080 api`
// alone for just the REST API, or `console` for a no-browser CLI chat —
// same shape as cmd/observability-agent.
//
// The agent's own definition lives in internal/agents/piiguardrail — this
// file is just the CLI/launcher entrypoint plus the plugin registration.
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"

	"github.com/v8tix/adk-training-go/internal/agents/piiguardrail"
	"github.com/v8tix/adk-training-go/internal/infrastructure/llm"
	"google.golang.org/adk/v2/agent"
	"google.golang.org/adk/v2/cmd/launcher"
	"google.golang.org/adk/v2/cmd/launcher/console"
	"google.golang.org/adk/v2/cmd/launcher/universal"
	"google.golang.org/adk/v2/cmd/launcher/web"
	"google.golang.org/adk/v2/cmd/launcher/web/api"
	"google.golang.org/adk/v2/cmd/launcher/web/webui"
	"google.golang.org/adk/v2/plugin"
	"google.golang.org/adk/v2/runner"
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
	fmt.Printf("🛡️  pii-guardrail using %s\n", modelName)
	rootAgent, err := piiguardrail.BuildRootAgent(llmModel)
	if err != nil {
		log.Fatalf("building agent: %v", err)
	}

	guardrailPlugin, err := piiguardrail.NewPIIGuardrailPlugin("pii_guardrail")
	if err != nil {
		log.Fatalf("building PII guardrail plugin: %v", err)
	}

	config := &launcher.Config{
		AgentLoader:  agent.NewSingleLoader(rootAgent),
		PluginConfig: runner.PluginConfig{Plugins: []*plugin.Plugin{guardrailPlugin}},
	}
	l := universal.NewLauncher(console.NewLauncher(), web.NewLauncher(webui.NewLauncher(), api.NewLauncher()))
	if err := l.Execute(ctx, config, os.Args[1:]); err != nil {
		log.Fatalf("run failed: %v\n\n%s", err, l.CommandLineSyntax())
	}
}
