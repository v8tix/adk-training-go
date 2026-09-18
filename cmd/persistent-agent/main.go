// Command persistent-agent demonstrates a custom session.Service backed by
// Redis (internal/infrastructure/redissession), injected into a real
// runner.Runner via runner.Config.SessionService — the direct equivalent of
// Python's Runner(app=app, session_service=custom_service).
//
// Run `persistent-agent set` (defaults to "blue"; pass a color to override,
// e.g. `persistent-agent set teal`) to store a favorite color, then stop
// the process; run `persistent-agent ask` (no arguments) in a *separate*
// process invocation to verify it survived. Each invocation builds its own
// fresh runner.Runner and session.Service, connected to the same Redis —
// proving persistence survives a process restart, matching Python's own
// "run once, stop, run again" lab exercise (module13_5).
//
// Requires a reachable Redis at REDIS_ADDR (default localhost:6379) — no
// cloud credentials, unlike Python's lab, which requires a real GCP
// Firestore project and `gcloud auth application-default login`.
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/joho/godotenv"
	"github.com/redis/go-redis/v9"

	"github.com/v8tix/adk-training-go/internal/agents/persistentagent"
	"github.com/v8tix/adk-training-go/internal/infrastructure/llm"
	"github.com/v8tix/adk-training-go/internal/infrastructure/redissession"
	"google.golang.org/adk/v2/agent"
	"google.golang.org/adk/v2/runner"
	"google.golang.org/genai"
)

const (
	appName   = "extensibility_demo"
	userID    = "student_1"
	sessionID = "demo-session"
)

func redisAddr() string {
	if addr := os.Getenv("REDIS_ADDR"); addr != "" {
		return addr
	}
	return "localhost:6379"
}

// runTurn creates a fresh runner (and a fresh session.Service instance,
// pointed at redisAddr) for every invocation, then runs one turn — the
// persistence proof is that state set by one invocation's runner is
// visible to a completely independent later invocation's runner, both
// talking to the same Redis. Takes redisAddr explicitly (rather than
// reading the environment itself) so tests can point it at a
// Testcontainers-started instance without touching the process environment.
func runTurn(ctx context.Context, redisAddr, message string) (string, error) {
	cfg := llm.LoadConfig()
	llmModel, _, err := llm.BuildModel(ctx, cfg)
	if err != nil {
		return "", fmt.Errorf("building model: %w", err)
	}

	rootAgent, err := persistentagent.BuildRootAgent(llmModel)
	if err != nil {
		return "", fmt.Errorf("building agent: %w", err)
	}

	client := redis.NewClient(&redis.Options{Addr: redisAddr})
	defer client.Close()
	sessionService := redissession.NewService(client)

	r, err := runner.New(runner.Config{
		AppName:           appName,
		Agent:             rootAgent,
		SessionService:    sessionService,
		AutoCreateSession: true,
	})
	if err != nil {
		return "", fmt.Errorf("creating runner: %w", err)
	}

	msg := genai.NewContentFromText(message, genai.RoleUser)
	var answer string
	for event, runErr := range r.Run(ctx, userID, sessionID, msg, agent.RunConfig{}) {
		if runErr != nil {
			return "", runErr
		}
		if event.Content == nil {
			continue
		}
		for _, p := range event.Content.Parts {
			if p.Text != "" && !p.Thought {
				answer = p.Text
			}
		}
	}
	return answer, nil
}

// usage is the exact error message any invalid invocation reports —
// checked directly by TestParseArgs's own error-case assertions, so the
// message and the behavior it describes can't silently drift apart.
const usage = "usage: %s set [<color>]|ask"

// parseArgs turns args (os.Args[1:]) into the message runTurn should send.
// Split out from main so this repo's own established "extract anything
// beyond pure wiring" rule applies here — this is genuinely testable logic
// (argument validation and message construction), not just CLI plumbing.
//
// "set" defaults to "blue" (this module's own documented demo fact,
// matching docs/module-13_5/lab.md's `persistent-agent set` invocation
// exactly) but now genuinely accepts a custom color instead of silently
// discarding one — confirmed live: passing a full sentence like "My
// favorite color is teal." here previously vanished with no error at all,
// since the original switch statement never looked at os.Args beyond
// index 1. "ask" takes no arguments — the recall question is fixed by
// design, so an unexpected extra argument is now a real error instead of
// something silently ignored.
func parseArgs(args []string) (message string, err error) {
	if len(args) == 0 {
		return "", fmt.Errorf(usage, "persistent-agent")
	}
	switch args[0] {
	case "set":
		color := "blue"
		if len(args) > 1 {
			color = strings.Join(args[1:], " ")
		}
		return fmt.Sprintf("My favorite color is %s.", color), nil
	case "ask":
		if len(args) > 1 {
			return "", fmt.Errorf("usage: %s ask (no arguments)", "persistent-agent")
		}
		return "What is my favorite color?", nil
	default:
		return "", fmt.Errorf(usage, "persistent-agent")
	}
}

func main() {
	ctx := context.Background()
	if err := godotenv.Load(); err != nil {
		fmt.Printf("ℹ️  No .env file loaded (%v) — continuing with the current environment.\n", err)
	}

	message, err := parseArgs(os.Args[1:])
	if err != nil {
		log.Fatal(err)
	}

	addr := redisAddr()
	fmt.Printf("🔥 persistent-agent using Redis at %s\n", addr)
	answer, err := runTurn(ctx, addr, message)
	if err != nil {
		log.Fatalf("run failed: %v", err)
	}
	fmt.Println(answer)
}
