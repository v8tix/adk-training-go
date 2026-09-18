// Command ui-agent runs the UI Agent (internal/agents/uiagent) as an ADK
// REST API server — this module's own custom HTML/JS client
// (cmd/ui-client-server) is its real "UI," so the Dev UI isn't the focus
// here, but `web --port=9093 webui api` and `console` still work, same
// shape as every other cmd/ program in this repo.
//
// This module's own lab invokes it as:
//
//	go run ./cmd/ui-agent web --port=9093 api -webui_address http://localhost:9094
//
// `-webui_address` is api's own flag (confirmed in
// cmd/launcher/web/api/api.go) — despite the name, it controls CORS's
// Access-Control-Allow-Origin for any origin hitting the REST API, not just
// the ADK Dev UI. Set it to cmd/ui-client-server's own origin so the
// browser's cross-origin fetch calls succeed — the Go-named equivalent of
// Python's `adk api_server --allow_origins`.
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"

	"github.com/v8tix/adk-training-go/internal/agents/uiagent"
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
	fmt.Printf("💬 ui-agent using %s\n", modelName)

	rootAgent, err := uiagent.BuildRootAgent(llmModel)
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
