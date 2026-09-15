// Command persistent-agent demonstrates a custom session.Service backed by
// Redis (internal/infrastructure/redissession), injected into a real
// runner.Runner via runner.Config.SessionService — the direct equivalent of
// Python's Runner(app=app, session_service=custom_service).
//
// Run `persistent-agent set` to store a favorite color, then stop the
// process; run `persistent-agent ask` in a *separate* process invocation to
// verify it survived. Each invocation builds its own fresh runner.Runner
// and session.Service, connected to the same Redis — proving persistence
// survives a process restart, matching Python's own "run once, stop, run
// again" lab exercise (module13_5).
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

func main() {
	ctx := context.Background()
	if err := godotenv.Load(); err != nil {
		fmt.Printf("ℹ️  No .env file loaded (%v) — continuing with the current environment.\n", err)
	}

	if len(os.Args) < 2 {
		log.Fatalf("usage: %s set|ask", os.Args[0])
	}

	var message string
	switch os.Args[1] {
	case "set":
		message = "My favorite color is blue."
	case "ask":
		message = "What is my favorite color?"
	default:
		log.Fatalf("usage: %s set|ask", os.Args[0])
	}

	addr := redisAddr()
	fmt.Printf("🔥 persistent-agent using Redis at %s\n", addr)
	answer, err := runTurn(ctx, addr, message)
	if err != nil {
		log.Fatalf("run failed: %v", err)
	}
	fmt.Println(answer)
}
