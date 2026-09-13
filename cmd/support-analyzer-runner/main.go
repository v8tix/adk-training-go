// Command support-analyzer-runner drives the Support Analyzer agent
// programmatically — no CLI launcher, no web server — the Go mirror of
// Python's module-6 lab: wrap the agent in a runner, then drive two
// independent users (Alice, Bob) through that single shared runner
// instance, proving session isolation. There's no Go equivalent of Python's
// separate App type (google.golang.org/adk/v2/runner.Config already carries
// what Python's App holds — plugins, compaction/caching — directly) or of
// run_debug() (Run's own iterator is the only execution method; this file's
// runOnce is the same "range over events, find the final response" idiom
// every cmd/ program in this repo already uses).
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/joho/godotenv"

	"github.com/v8tix/adk-training-go/internal/agents/supportanalyzer"
	"github.com/v8tix/adk-training-go/internal/infrastructure/llm"
	"google.golang.org/adk/v2/agent"
	"google.golang.org/adk/v2/runner"
	"google.golang.org/genai"
)

// runOnce drives one message through r for the given user/session, waits
// for the final response event, and returns the structured analysis the SDK
// wrote to event.Actions.StateDelta[supportanalyzer.OutputKey].
func runOnce(ctx context.Context, r *runner.Runner, userID, sessionID, message string) (string, error) {
	msg := genai.NewContentFromText(message, genai.RoleUser)
	for event, err := range r.Run(ctx, userID, sessionID, msg, agent.RunConfig{}) {
		if err != nil {
			return "", err
		}
		if !event.IsFinalResponse() {
			continue
		}
		if s, ok := event.Actions.StateDelta[supportanalyzer.OutputKey].(string); ok {
			return s, nil
		}
	}
	return "", nil
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
	fmt.Printf("🎫 support-analyzer-runner using %s\n", modelName)

	rootAgent, err := supportanalyzer.BuildRootAgent(llmModel)
	if err != nil {
		log.Fatalf("building agent: %v", err)
	}

	// One Runner instance, shared by every user — the point of this module.
	r, err := runner.NewInMemory("support_analyzer_runner_app", rootAgent)
	if err != nil {
		log.Fatalf("building runner: %v", err)
	}

	fmt.Println("--- User A (Alice) ---")
	aliceResult, err := runOnce(ctx, r, "alice", "alice_session", "I was overcharged $50")
	if err != nil {
		log.Fatalf("running Alice's message: %v", err)
	}
	fmt.Printf("Agent Response: %s\n", aliceResult)

	fmt.Println("\n--- User B (Bob) ---")
	bobResult, err := runOnce(ctx, r, "bob", "bob_session", "My wifi is slow")
	if err != nil {
		log.Fatalf("running Bob's message: %v", err)
	}
	fmt.Printf("Agent Response: %s\n", bobResult)
}
