// Command mcp-filesystem runs the Filesystem MCP agent
// (internal/agents/mcpfilesystem), sandboxed to this program's own
// test_files/ directory. Run with `web --port 8080 webui api` for the Dev
// UI, `web --port 8080 api` alone for just the REST API, or `console` for a
// no-browser CLI chat — same shape as cmd/pii-guardrail.
//
// The agent's own definition lives in internal/agents/mcpfilesystem — this
// file is just the CLI/launcher entrypoint plus sandbox-directory
// resolution. Requires Node.js/npx on PATH: the filesystem MCP server
// (@modelcontextprotocol/server-filesystem) is a third-party Node package,
// launched as a local subprocess over stdio.
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"

	"github.com/joho/godotenv"

	"github.com/v8tix/adk-training-go/internal/agents/mcpfilesystem"
	"github.com/v8tix/adk-training-go/internal/infrastructure/llm"
	"google.golang.org/adk/v2/agent"
	"google.golang.org/adk/v2/cmd/launcher"
	"google.golang.org/adk/v2/cmd/launcher/console"
	"google.golang.org/adk/v2/cmd/launcher/universal"
	"google.golang.org/adk/v2/cmd/launcher/web"
	"google.golang.org/adk/v2/cmd/launcher/web/api"
	"google.golang.org/adk/v2/cmd/launcher/web/webui"
)

// thisFileDir is this source file's own directory, resolved once via
// runtime.Caller — cwd-independent, unlike filepath.Abs("cmd/mcp-filesystem/...").
// A relative path resolves against the *process's* working directory, which
// for `go run ./cmd/mcp-filesystem` is the repo root but for `go test
// ./cmd/mcp-filesystem/...` is this package's own directory — using a
// cwd-relative path here previously created a doubled, stray
// cmd/mcp-filesystem/cmd/mcp-filesystem/test_files/ tree on every test run,
// caught live in Phase 5 review.
var thisFileDir = func() string {
	_, file, _, _ := runtime.Caller(0)
	return filepath.Dir(file)
}()

// sandboxDir resolves this program's own test_files/ directory to an
// absolute path, creating it (with a sample file) if it doesn't exist yet —
// the filesystem MCP server needs a real, existing directory on disk to
// sandbox itself to.
func sandboxDir() (string, error) {
	dir := filepath.Join(thisFileDir, "test_files")
	if err := ensureSandboxDir(dir); err != nil {
		return "", err
	}
	return dir, nil
}

// ensureSandboxDir creates dir (with a sample hello.txt) if it doesn't
// exist yet, and leaves an existing directory untouched. Split out from
// sandboxDir so a test can exercise both branches against an isolated
// t.TempDir() path, without touching the real, checked-in fixture.
func ensureSandboxDir(dir string) error {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("creating sandbox directory: %w", err)
		}
		sample := filepath.Join(dir, "hello.txt")
		if err := os.WriteFile(sample, []byte("Hello from the MCP world!"), 0o644); err != nil {
			return fmt.Errorf("writing sample file: %w", err)
		}
		return nil
	} else if err != nil {
		return err
	}
	return nil
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
	fmt.Printf("📂 mcp-filesystem using %s\n", modelName)

	dir, err := sandboxDir()
	if err != nil {
		log.Fatalf("resolving sandbox directory: %v", err)
	}
	fmt.Printf("Sandboxed to %s\n", dir)

	rootAgent, err := mcpfilesystem.BuildRootAgent(llmModel, dir)
	if err != nil {
		log.Fatalf("building agent: %v", err)
	}

	config := &launcher.Config{
		AgentLoader: agent.NewSingleLoader(rootAgent),
	}
	l := universal.NewLauncher(console.NewLauncher(), web.NewLauncher(webui.NewLauncher(), api.NewLauncher()))
	if err := l.Execute(ctx, config, os.Args[1:]); err != nil {
		log.Fatalf("run failed: %v\n\n%s", err, l.CommandLineSyntax())
	}
}
