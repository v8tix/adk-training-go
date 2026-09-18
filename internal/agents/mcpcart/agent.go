// Package mcpcart defines the Shopping Agent: an llmagent whose only tools
// come from a custom MCP server this repo builds itself
// (cmd/cart-mcp-server) — the client-side counterpart to module-28's real
// lesson, becoming a tool *provider*, not just a consumer as in module-27.
// The agent launches the server as a subprocess via `go run`, over stdio,
// reusing the exact mcptoolset/mcp.CommandTransport mechanism module-27
// already proved against a third-party server.
package mcpcart

import (
	"embed"
	"io/fs"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/v8tix/adk-training-go/internal/infrastructure/prompts"
	"google.golang.org/adk/v2/agent"
	"google.golang.org/adk/v2/agent/llmagent"
	"google.golang.org/adk/v2/model"
	"google.golang.org/adk/v2/tool"
	"google.golang.org/adk/v2/tool/mcptoolset"
)

// PromptNamespace identifies this package's entries in the shared prompts
// cache (internal/infrastructure/prompts).
const PromptNamespace = "mcpcart"

//go:embed prompts/*.md
var promptFS embed.FS

func init() {
	promptFiles, err := fs.Sub(promptFS, "prompts")
	if err != nil {
		log.Fatalf("resolving prompts directory: %v", err)
	}
	if err := prompts.Register(PromptNamespace, promptFiles, ".md"); err != nil {
		log.Fatalf("registering prompts: %v", err)
	}
}

// RepoRoot resolves the repository root via `go env GOMOD`'s own directory
// — cwd-independent, unlike a relative "../../.." chain or a hardcoded
// literal path. Confirmed live during module-27's own review: a
// cwd-relative path silently breaks under `go test`, which runs with a
// different working directory than `go run` does. Shared by
// cmd/shopping-agent/main.go and this package's own test, so both resolve
// the same real repo root regardless of their own working directory.
//
// Caveat, confirmed during Phase 5 review (not applicable to this
// single-module repo, but worth knowing before reusing this pattern
// elsewhere): run from a `go.work` workspace root itself, rather than from
// inside one of its member modules, `go env GOMOD` returns "/dev/null" —
// filepath.Dir of that silently yields "/dev", not a real repo root.
func RepoRoot() (string, error) {
	out, err := exec.Command("go", "env", "GOMOD").Output()
	if err != nil {
		return "", err
	}
	return filepath.Dir(strings.TrimSpace(string(out))), nil
}

// buildServerCmd builds the cart-mcp-server subprocess command, with its
// own Stderr forwarded to stderr — split out from BuildRootAgent so this
// specific, previously-silent gotcha stays under a direct regression test.
// mcp.CommandTransport wires the subprocess's stdin/stdout to the MCP
// protocol itself, but never touches Stderr — confirmed by reading cmd.go
// directly. Left nil, exec.Cmd discards it, silently swallowing every
// log.Printf line cart-mcp-server's own handlers emit. Forwarding it to
// this process's own Stderr is what actually lets a learner see the
// server's own "[Server]: ..." lines, matching Python's own lab step
// ("Examine the server logs in the console").
func buildServerCmd(repoRoot string) *exec.Cmd {
	serverCmd := exec.Command("go", "run", "./cmd/cart-mcp-server")
	serverCmd.Dir = repoRoot
	serverCmd.Stderr = os.Stderr
	return serverCmd
}

// BuildRootAgent constructs the Shopping Agent around llmModel, launching
// cmd/cart-mcp-server (via `go run`, working directory repoRoot) as its MCP
// server subprocess. repoRoot is an explicit parameter, not resolved
// internally — the same "explicit over hidden" design mcpfilesystem's own
// sandboxDir parameter established in module-27, after a cwd-relative path
// there silently broke under go test. mcptoolset.New's own doc comment
// confirms the MCP session is created lazily, so this function never
// launches the subprocess itself, even for an invalid repoRoot.
func BuildRootAgent(llmModel model.LLM, repoRoot string) (agent.Agent, error) {
	instruction, err := prompts.Get(PromptNamespace + "/shopping_instruction")
	if err != nil {
		return nil, err
	}

	toolset, err := mcptoolset.New(mcptoolset.Config{
		Transport: &mcp.CommandTransport{Command: buildServerCmd(repoRoot)},
	})
	if err != nil {
		return nil, err
	}

	return llmagent.New(llmagent.Config{
		Name:        "shopping_agent",
		Model:       llmModel,
		Description: "A shopping assistant that can add items to a cart and report its contents.",
		Instruction: instruction,
		Toolsets:    []tool.Toolset{toolset},
	})
}
