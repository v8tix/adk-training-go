// Command visual-catalog analyzes local product photos with a vision-capable
// agent and prints a marketing description for each — the Go mirror of
// Python's module-7 lab. No launcher: this is a plain programmatic script,
// same shape as cmd/support-analyzer-runner (module-6). Run it as
// `go run ./cmd/visual-catalog` from the repo root — its image paths are
// relative to there, matching every other cmd/ program's documented
// invocation in this repo.
//
// Requires Gemini — confirmed live that the local Ollama backend's Go client
// (model/openaimodel) has no code path for sending an image Part at all, so
// this always forces MODEL_TYPE=gemini in code rather than trusting the
// shared .env default. See docs/module-07/README.md for the full finding.
//
// Also demonstrates this module's own lesson: unlike runner.NewInMemory
// (auto-creates sessions), this program builds its Runner with
// AutoCreateSession left false and explicitly creates each product's session
// before running it.
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"

	"github.com/v8tix/adk-training-go/internal/agents/visualcatalog"
	"github.com/v8tix/adk-training-go/internal/infrastructure/llm"
	"google.golang.org/adk/v2/agent"
	"google.golang.org/adk/v2/runner"
	"google.golang.org/adk/v2/session"
	"google.golang.org/genai"
)

const appName = "visual_catalog_app"

// analyzeProduct explicitly creates productID's session (the point of this
// module — Run fails with a clear "session not found" error without this
// step, confirmed live), loads its image, builds a multimodal message, runs
// it, and prints the agent's description.
func analyzeProduct(ctx context.Context, r *runner.Runner, sessionSvc session.Service, productID, imagePath string) error {
	fmt.Printf("\n--- Analyzing Product: %s ---\n", productID)

	userID := "catalog_admin"
	sessionID := "sess_" + productID
	if _, err := sessionSvc.Create(ctx, &session.CreateRequest{
		AppName:   appName,
		UserID:    userID,
		SessionID: sessionID,
	}); err != nil {
		return fmt.Errorf("creating session: %w", err)
	}

	imageBytes, err := os.ReadFile(imagePath)
	if err != nil {
		return fmt.Errorf("reading image: %w", err)
	}

	msg := genai.NewContentFromParts([]*genai.Part{
		genai.NewPartFromText(fmt.Sprintf("Analyze this image for product %s.", productID)),
		genai.NewPartFromBytes(imageBytes, "image/jpeg"),
	}, genai.RoleUser)

	fmt.Println("📸 Sending image to Gemini...")
	for event, err := range r.Run(ctx, userID, sessionID, msg, agent.RunConfig{}) {
		if err != nil {
			return fmt.Errorf("running agent: %w", err)
		}
		if !event.IsFinalResponse() || event.Content == nil {
			continue
		}
		for _, p := range event.Content.Parts {
			if p.Text != "" && !p.Thought {
				fmt.Printf("✅ Description:\n%s\n", p.Text)
			}
		}
	}
	return nil
}

func main() {
	ctx := context.Background()
	if err := godotenv.Load(); err != nil {
		fmt.Printf("ℹ️  No .env file loaded (%v) — continuing with the current environment.\n", err)
	}

	cfg := llm.LoadConfig()
	cfg.ModelType = llm.ModelTypeGemini // vision requires it — see package doc comment
	llmModel, modelName, err := llm.BuildModel(ctx, cfg)
	if err != nil {
		log.Fatalf("building model: %v", err)
	}
	fmt.Printf("🎨 visual-catalog using %s\n", modelName)

	rootAgent, err := visualcatalog.BuildRootAgent(llmModel)
	if err != nil {
		log.Fatalf("building agent: %v", err)
	}

	sessionSvc := session.InMemoryService()
	r, err := runner.New(runner.Config{
		AppName:        appName,
		Agent:          rootAgent,
		SessionService: sessionSvc,
		// AutoCreateSession left false (the zero value) deliberately — this
		// module's own lesson is the explicit session.Create call below.
	})
	if err != nil {
		log.Fatalf("building runner: %v", err)
	}

	products := []struct {
		id        string
		imagePath string
	}{
		{"HEADPHONES-01", "cmd/visual-catalog/images/headphones.jpg"},
		{"LAPTOP-02", "cmd/visual-catalog/images/laptop.jpg"},
	}

	for _, p := range products {
		if err := analyzeProduct(ctx, r, sessionSvc, p.id, p.imagePath); err != nil {
			log.Fatalf("analyzing %s: %v", p.id, err)
		}
	}
}
