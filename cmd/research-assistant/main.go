// Command research-assistant runs a two-agent research pipeline: a
// search-only agent grounds itself in live web results via google_search,
// then a formatter-only agent (no google_search) turns those findings into a
// structured report — the sequential-composition workaround for the Gemini
// API's default restriction against mixing a built-in tool with custom
// function tools in one agent (confirmed live, module-12; see
// docs/module-12/README.md for the full finding, including the
// BuildCombinedAgent alternative that lifts the restriction instead of
// working around it).
//
// Unlike every prior cmd/ program in this repo, this one has no natural fit
// for cmd/launcher's single-agent console/web shell — Python's own main.py
// is a plain standalone script running two agents in sequence, not something
// `adk web` drives, so this mirrors that shape directly: run with no
// arguments, or pass a topic as the first argument.
//
// Requires MODEL_TYPE=gemini (forced in code, not left to the shared .env
// default) — google_search is confirmed live (module-8) to be rejected by
// the local Ollama backend's Go client before ever reaching the network.
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/joho/godotenv"

	"github.com/v8tix/adk-training-go/internal/agents/researchassistant"
	"github.com/v8tix/adk-training-go/internal/infrastructure/llm"
	"google.golang.org/adk/v2/agent"
	"google.golang.org/adk/v2/model"
	"google.golang.org/adk/v2/runner"
	"google.golang.org/genai"
)

const defaultTopic = "the latest AI developments from Google"

// runAgent runs one turn of builtAgent against messageText, in its own
// fresh in-memory session, and returns the final non-thought text answer —
// the Go equivalent of Python's run_agent helper.
func runAgent(ctx context.Context, builtAgent agent.Agent, appName, messageText string) (string, error) {
	r, err := runner.NewInMemory(appName, builtAgent)
	if err != nil {
		return "", err
	}

	msg := genai.NewContentFromText(messageText, genai.RoleUser)
	var answer string
	for event, runErr := range r.Run(ctx, "student", "s1", msg, agent.RunConfig{}) {
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

// runResearchPipeline builds both agents and runs the search agent, then the
// formatter agent, feeding the first's output into the second's input —
// extracted so main() and the integration test call the same real logic.
func runResearchPipeline(ctx context.Context, llmModel model.LLM, topic string) (findings, report string, err error) {
	researchAgent, err := researchassistant.BuildResearchAgent(llmModel)
	if err != nil {
		return "", "", fmt.Errorf("building research agent: %w", err)
	}
	formatterAgent, err := researchassistant.BuildFormatterAgent(llmModel)
	if err != nil {
		return "", "", fmt.Errorf("building formatter agent: %w", err)
	}

	findings, err = runAgent(ctx, researchAgent, "research_app", "Research this topic: "+topic)
	if err != nil {
		return "", "", fmt.Errorf("running research agent: %w", err)
	}

	report, err = runAgent(ctx, formatterAgent, "formatter_app", "Topic: "+topic+"\n\nFindings: "+findings)
	if err != nil {
		return findings, "", fmt.Errorf("running formatter agent: %w", err)
	}

	return findings, report, nil
}

func main() {
	ctx := context.Background()
	if err := godotenv.Load(); err != nil {
		fmt.Printf("ℹ️  No .env file loaded (%v) — continuing with the current environment.\n", err)
	}
	cfg := llm.LoadConfig()
	cfg.ModelType = llm.ModelTypeGemini
	llmModel, modelName, err := llm.BuildModel(ctx, cfg)
	if err != nil {
		log.Fatalf("building model: %v", err)
	}
	fmt.Printf("🔎 research-assistant using %s\n", modelName)

	topic := defaultTopic
	if args := os.Args[1:]; len(args) > 0 {
		topic = strings.Join(args, " ")
	}

	findings, report, err := runResearchPipeline(ctx, llmModel, topic)
	if err != nil {
		log.Fatalf("run failed: %v", err)
	}

	fmt.Println("--- RESEARCH FINDINGS ---")
	fmt.Println(findings)
	fmt.Println("\n--- FINAL REPORT ---")
	fmt.Println(report)
}
