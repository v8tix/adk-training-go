// Command echo-agent runs a strict "echo only" agent: it repeats the user's
// input verbatim and never answers questions. Run with `web --port 8080
// webui` for the Dev UI (note: webui must be named after web's own flags) or
// `console` for a no-browser CLI chat.
package main

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"log"
	"os"

	"github.com/joho/godotenv"

	"github.com/v8tix/adk-training-go/internal/infrastructure/llm"
	"github.com/v8tix/adk-training-go/internal/infrastructure/prompts"
	"google.golang.org/adk/v2/agent"
	"google.golang.org/adk/v2/agent/llmagent"
	"google.golang.org/adk/v2/cmd/launcher"
	"google.golang.org/adk/v2/cmd/launcher/console"
	"google.golang.org/adk/v2/cmd/launcher/universal"
	"google.golang.org/adk/v2/cmd/launcher/web"
	"google.golang.org/adk/v2/cmd/launcher/web/webui"
	"google.golang.org/adk/v2/model"
)

// promptNamespace identifies this program's entries in the shared prompts
// cache (internal/infrastructure/prompts), so "echo_instruction" here can't
// collide with another program's prompt of the same name.
const promptNamespace = "echo-agent"

//go:embed prompts/*.md
var promptFS embed.FS

// init (not a call inside main) registers this program's prompts into the
// shared cache before anything — including tests, which never call main() —
// needs to read them via prompts.Get. fs.Sub roots the FS at "prompts" first:
// //go:embed prompts/*.md keeps the "prompts/" path prefix inside promptFS
// (it's not stripped the way a single-file `//go:embed path` into a string
// is), so without this, entries would register as "<namespace>/prompts/name"
// instead of "<namespace>/name".
func init() {
	promptFiles, err := fs.Sub(promptFS, "prompts")
	if err != nil {
		log.Fatalf("resolving prompts directory: %v", err)
	}
	if err := prompts.Register(promptNamespace, promptFiles, ".md"); err != nil {
		log.Fatalf("registering prompts: %v", err)
	}
}

// buildRootAgent constructs the echo agent's definition around llmModel and
// instruction. A separate function (not inlined in main) so tests can build
// the exact same agent main() would, instead of re-declaring it and testing
// a copy.
func buildRootAgent(llmModel model.LLM, instruction string) (agent.Agent, error) {
	return llmagent.New(llmagent.Config{
		Name:        "echo_agent",
		Model:       llmModel,
		Description: "An agent that repeats the user's input.",
		Instruction: instruction,
	})
}

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
	fmt.Printf("🔊 echo-agent using %s\n", modelName)

	instruction, err := prompts.Get(promptNamespace + "/echo_instruction")
	if err != nil {
		log.Fatalf("loading prompt: %v", err)
	}

	rootAgent, err := buildRootAgent(llmModel, instruction)
	if err != nil {
		log.Fatalf("building agent: %v", err)
	}

	config := &launcher.Config{AgentLoader: agent.NewSingleLoader(rootAgent)}

	// Only console (CLI chat) and web+webui (Dev UI) — not the full bundle's
	// A2A/pub-sub/Eventarc/REST-API sub-launchers, which this echo agent has
	// no use for and which pull in a much larger dependency graph.
	l := universal.NewLauncher(console.NewLauncher(), web.NewLauncher(webui.NewLauncher()))
	if err := l.Execute(ctx, config, os.Args[1:]); err != nil {
		log.Fatalf("run failed: %v\n\n%s", err, l.CommandLineSyntax())
	}
}
