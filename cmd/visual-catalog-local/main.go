// Command visual-catalog-local is a bonus, outside this course's ADK-SDK
// lesson: it analyzes the same two product photos as cmd/visual-catalog, but
// by calling the local Ollama server directly (internal/agents/visualcatalog's
// DescribeImageLocally) instead of through llmagent/runner.
//
// This exists to back up module-7's real finding, not to replace it:
// model/openaimodel (the ADK Go SDK's local-model client) has no code path
// for sending an image Part at all — confirmed live, cmd/visual-catalog
// needs Gemini because of that SDK limitation, not because Ollama or the
// underlying model can't handle images. This program proves the local model
// server side of that claim by sending the same real images directly,
// bypassing the SDK entirely.
//
// Uses the repo's shared OLLAMA_MODEL default (no override needed) — that
// default was already confirmed to handle vision correctly back when this
// module chose it, which is exactly the point: this would be free and fully
// local today, if only the SDK's client supported it.
//
// Run it as `go run ./cmd/visual-catalog-local` from the repo root — its
// image paths are relative to there, same as cmd/visual-catalog.
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/joho/godotenv"

	"github.com/v8tix/adk-training-go/internal/agents/visualcatalog"
	"github.com/v8tix/adk-training-go/internal/infrastructure/llm"
)

func main() {
	ctx := context.Background()
	if err := godotenv.Load(); err != nil {
		fmt.Printf("ℹ️  No .env file loaded (%v) — continuing with the current environment.\n", err)
	}

	cfg := llm.LoadConfig() // shared default (qwen3.8:27b) — no MODEL_TYPE override
	fmt.Printf("🎨 visual-catalog-local using %s directly (no ADK agent/runner)\n", cfg.OllamaModel)

	products := []struct {
		id        string
		imagePath string
	}{
		{"HEADPHONES-01", "cmd/visual-catalog/images/headphones.jpg"},
		{"LAPTOP-02", "cmd/visual-catalog/images/laptop.jpg"},
	}

	for _, p := range products {
		fmt.Printf("\n--- Analyzing Product: %s ---\n", p.id)
		description, err := visualcatalog.DescribeImageLocally(ctx, cfg.OllamaBaseURL, cfg.OllamaModel, p.imagePath)
		if err != nil {
			log.Fatalf("analyzing %s: %v", p.id, err)
		}
		fmt.Printf("✅ Description:\n%s\n", description)
	}
}
