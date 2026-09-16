// Command essay-refiner runs a cyclic workflow: a single
// workflow.DynamicNode writes an initial draft, then loops a critic and a
// refiner (capped at 3 iterations) until the critic approves. Run with
// `web --port 9091 webui -api_server_address http://localhost:9091/api api`
// for the Dev UI, or `console` for a no-browser CLI chat — same shape as
// cmd/support-router.
//
// Confirmed live (module-20): the console launcher streams every
// intermediate chat-content event as it happens (the draft, each critic
// verdict, each refined rewrite), so a console session shows the essay
// evolving in real time — matching this course's own "transparency" framing
// for iterative workflows. The loop's own final return value only ever
// gets specially rendered as a fallback when no chat content streamed at
// all (confirmed by reading cmd/launcher/console/console.go directly),
// which never happens here since this loop always produces real chat
// content.
//
// The graph's own definition lives in internal/agents/essayrefiner — this
// file is just the CLI/launcher entrypoint.
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"

	"github.com/v8tix/adk-training-go/internal/agents/essayrefiner"
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
	fmt.Printf("✍️  essay-refiner using %s\n", modelName)
	rootAgent, err := essayrefiner.BuildRootAgent(llmModel)
	if err != nil {
		log.Fatalf("building agent: %v", err)
	}
	config := &launcher.Config{AgentLoader: agent.NewSingleLoader(rootAgent)}
	l := universal.NewLauncher(console.NewLauncher(), web.NewLauncher(webui.NewLauncher(), api.NewLauncher()))
	if err := l.Execute(ctx, config, os.Args[1:]); err != nil {
		log.Fatalf("run failed: %v\n\n%s", err, l.CommandLineSyntax())
	}
}
