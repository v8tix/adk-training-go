// Command observability-agent runs the Observability agent with a real
// Runner-level Plugin (internal/agents/observability's own AlertingPlugin)
// registered via launcher.Config.PluginConfig. Run with `web --port 8080
// webui api` for the Dev UI, `web --port 8080 api` alone for just the REST
// API, or `console` for a no-browser CLI chat — same shape as
// cmd/personal-tutor.
//
// This module's real point needs no extra wiring here to demonstrate:
// both the console and web sub-launchers already expose a `-otel_to_cloud`
// flag wired straight into google.golang.org/adk/v2/telemetry (confirmed
// live in cmd/launcher/console/console.go and cmd/launcher/web/web.go).
// The default, `-otel_to_cloud=false`, needs no cloud credentials at all —
// exactly what this program's own tests and lab exercise. Passing
// `-otel_to_cloud=true` would additionally need real Google Application
// Default Credentials and a real GCP project to actually export anything
// to Cloud Trace/Monitoring; neither this program nor its tests ever do
// that, matching this repo's own "never require real cloud credentials for
// the shipped lab" discipline.
//
// The agent's own definition lives in internal/agents/observability — this
// file is just the CLI/launcher entrypoint plus the plugin registration.
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"

	"github.com/v8tix/adk-training-go/internal/agents/observability"
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
	fmt.Printf("🔭 observability-agent using %s\n", modelName)
	rootAgent, err := observability.BuildRootAgent(llmModel)
	if err != nil {
		log.Fatalf("building agent: %v", err)
	}

	alertingPlugin, err := observability.NewAlertingPlugin("alerting_plugin")
	if err != nil {
		log.Fatalf("building alerting plugin: %v", err)
	}

	config := &launcher.Config{
		AgentLoader:  agent.NewSingleLoader(rootAgent),
		PluginConfig: runner.PluginConfig{Plugins: []*plugin.Plugin{alertingPlugin}},
	}
	l := universal.NewLauncher(console.NewLauncher(), web.NewLauncher(webui.NewLauncher(), api.NewLauncher()))
	if err := l.Execute(ctx, config, os.Args[1:]); err != nil {
		log.Fatalf("run failed: %v\n\n%s", err, l.CommandLineSyntax())
	}
}
