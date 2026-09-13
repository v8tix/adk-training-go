// Command verify-setup checks that this repository's Go environment is ready
// for ADK 2.0 development: the ADK Go SDK is resolvable, the Go toolchain
// meets the minimum version, and a model backend actually responds.
package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"runtime"
	"runtime/debug"

	"github.com/joho/godotenv"

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
)

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
	cfg := loadConfig()

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

// verifyConnectivity builds whichever model buildModel's factory produces for
// cfg.ModelType, then runs one prompt against it and prints the result.
func verifyConnectivity(ctx context.Context, cfg config) error {
	llmModel, modelName, err := buildModel(ctx, cfg)
	if err != nil {
		return err
	}

	fmt.Printf("🚀 Connecting to %s...\n", modelName)

	answer, err := runPrompt(ctx, llmModel, "Reply with exactly: ADK 2.0 is Ready!", "Hello!")
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
