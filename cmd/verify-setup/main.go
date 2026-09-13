// Command verify-setup checks that this repository's Go environment is ready
// for ADK 2.0 development: the ADK Go SDK is resolvable, the Go toolchain
// meets the minimum version, and a model backend actually responds.
package main

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"runtime"
	"runtime/debug"

	"github.com/joho/godotenv"

	"github.com/v8tix/adk-training-go/internal/infrastructure/llm"
	"github.com/v8tix/adk-training-go/internal/infrastructure/prompts"
	"google.golang.org/adk/v2/agent"
	"google.golang.org/adk/v2/agent/llmagent"
	"google.golang.org/adk/v2/model"
	"google.golang.org/adk/v2/runner"
	"google.golang.org/genai"
)

const (
	adkModulePath = "google.golang.org/adk/v2"

	// This course's chosen Go floor (matches go.mod), not just the ADK SDK's
	// own bare minimum (1.25) — mirrors the Python course checking its chosen
	// Python floor (3.10) separately from the ADK package's own requirement.
	minGoVersion = "go1.27"

	// promptNamespace identifies this program's entries in the shared
	// prompts cache (internal/infrastructure/prompts).
	promptNamespace = "verify-setup"
)

//go:embed prompts/*.md
var promptFS embed.FS

// init registers this program's prompts into the shared cache before
// anything — including tests, which never call main() — needs to read them
// via prompts.Get. fs.Sub is required: see internal/infrastructure/prompts's
// doc comment for why.
func init() {
	promptFiles, err := fs.Sub(promptFS, "prompts")
	if err != nil {
		panic(fmt.Sprintf("resolving prompts directory: %v", err))
	}
	if err := prompts.Register(promptNamespace, promptFiles, ".md"); err != nil {
		panic(fmt.Sprintf("registering prompts: %v", err))
	}
}

var (
	// ErrBuildingAgent indicates llmagent.New failed.
	ErrBuildingAgent = errors.New("building agent failed")
	// ErrBuildingRunner indicates runner.NewInMemory failed.
	ErrBuildingRunner = errors.New("building runner failed")
	// ErrRunningAgent indicates the agent run loop returned an error.
	ErrRunningAgent = errors.New("running agent failed")
	// ErrNoFinalResponse indicates the run loop completed without ever
	// yielding a final, non-thought response.
	ErrNoFinalResponse = errors.New("no final response received")
)

// wrapErr wraps err with sentinel via %w (so errors.Is(result, sentinel)
// finds it) while keeping err's own message for context.
func wrapErr(sentinel, err error) error {
	return fmt.Errorf("%w: %v", sentinel, err)
}

func main() {
	if err := godotenv.Load(); err != nil {
		fmt.Printf("ℹ️  No .env file loaded (%v) — continuing with the current environment.\n", err)
	}
	cfg := llm.LoadConfig()

	fmt.Println("🔍 Testing ADK 2.0 Go Environment...")

	if !checkADKVersion() {
		os.Exit(1)
	}
	if !checkGoVersion() {
		os.Exit(1)
	}

	if err := verifyConnectivity(context.Background(), cfg); err != nil {
		fmt.Printf("❌ %v\n", err)
		os.Exit(1)
	}

	fmt.Println("\n🎉 SETUP COMPLETE! You are running ADK 2.0 for Go.")
}

func checkADKVersion() bool {
	bi, ok := debug.ReadBuildInfo()
	if !ok {
		fmt.Println("❌ Could not read build info to verify the ADK dependency.")
		return false
	}

	version, found := resolvedModuleVersion(bi, adkModulePath)
	if !found {
		fmt.Printf("❌ %s is not a resolved dependency of this binary.\n", adkModulePath)
		return false
	}

	fmt.Printf("📦 Google ADK (Go) version: %s\n", version)
	fmt.Println("✅ ADK 2.0+ is installed correctly.")
	return true
}

func checkGoVersion() bool {
	ok, err := goVersionMeetsMinimum(runtime.Version(), minGoVersion)
	if err != nil {
		fmt.Printf("❌ Could not parse Go version %q: %v\n", runtime.Version(), err)
		return false
	}
	if !ok {
		fmt.Printf("❌ %s+ is required. You are using %s.\n", minGoVersion, runtime.Version())
		return false
	}

	fmt.Printf("✅ %s+ requirement met (%s).\n", minGoVersion, runtime.Version())
	return true
}

// verifyConnectivity builds whichever model llm.BuildModel's factory
// produces for cfg.ModelType, then runs one prompt against it and prints the
// result.
func verifyConnectivity(ctx context.Context, cfg llm.Config) error {
	llmModel, modelName, err := llm.BuildModel(ctx, cfg)
	if err != nil {
		return err
	}

	fmt.Printf("🚀 Connecting to %s...\n", modelName)

	instruction, err := prompts.Get(promptNamespace + "/verify_instruction")
	if err != nil {
		return err
	}

	answer, err := runPrompt(ctx, llmModel, instruction, "Hello!")
	if err != nil {
		return err
	}
	if answer == "" {
		return fmt.Errorf("%w: from %s", ErrNoFinalResponse, modelName)
	}

	fmt.Printf("✅ Agent response: %s\n", answer)
	return nil
}

// runPrompt builds a minimal single-turn agent around llmModel and runs
// prompt through it once, returning the model's final (non-thought) answer.
// Kept free of printing/status output so it can be driven by a fake
// model.LLM in tests, independent of a real network call.
func runPrompt(ctx context.Context, llmModel model.LLM, instruction, prompt string) (string, error) {
	verifyAgent, err := llmagent.New(llmagent.Config{
		Name:        "verify_agent",
		Model:       llmModel,
		Instruction: instruction,
	})
	if err != nil {
		return "", wrapErr(ErrBuildingAgent, err)
	}

	r, err := runner.NewInMemory("verify_app", verifyAgent)
	if err != nil {
		return "", wrapErr(ErrBuildingRunner, err)
	}

	msg := genai.NewContentFromText(prompt, genai.RoleUser)

	for event, err := range r.Run(ctx, "verify_user", "verify_session", msg, agent.RunConfig{}) {
		if err != nil {
			return "", wrapErr(ErrRunningAgent, err)
		}
		if !event.IsFinalResponse() || event.Content == nil {
			continue
		}
		if text, ok := firstAnswerText(event.Content.Parts); ok {
			return text, nil
		}
	}

	return "", nil
}
