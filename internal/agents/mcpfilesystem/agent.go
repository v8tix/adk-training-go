// Package mcpfilesystem defines the Filesystem MCP agent: an llmagent whose
// only tools come from a real, external Model Context Protocol server —
// @modelcontextprotocol/server-filesystem, launched as a local subprocess
// over stdio and sandboxed to one directory. The agent never defines
// list_directory or read_file itself; mcptoolset discovers both from the
// server at runtime. Matches Python's own "filesystem_agent" role exactly.
package mcpfilesystem

import (
	"embed"
	"io/fs"
	"log"
	"os/exec"

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
const PromptNamespace = "mcpfilesystem"

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

// allowedTools restricts the MCP server's own tool surface to just the two
// the lab exercises — the direct equivalent of Python's
// tool_filter=["list_directory", "read_file"], expressed via
// tool.FilterToolset + tool.AllowedToolsPredicate (mcptoolset.Config's own
// ToolFilter field, and tool.StringPredicate, are both deprecated in favor
// of this current pairing).
var allowedTools = []string{"list_directory", "read_file"}

// BuildRootAgent constructs the Filesystem MCP agent around llmModel,
// sandboxed to sandboxDir. mcptoolset.New's own doc comment confirms the MCP
// session is created lazily, on the first request to the LLM — this function
// never launches the npx subprocess or dials anything itself, so it succeeds
// for any sandboxDir value, even one that doesn't exist yet.
func BuildRootAgent(llmModel model.LLM, sandboxDir string) (agent.Agent, error) {
	instruction, err := prompts.Get(PromptNamespace + "/filesystem_instruction")
	if err != nil {
		return nil, err
	}

	toolset, err := mcptoolset.New(mcptoolset.Config{
		Transport: &mcp.CommandTransport{
			Command: exec.Command("npx", "-y", "@modelcontextprotocol/server-filesystem", sandboxDir),
		},
	})
	if err != nil {
		return nil, err
	}

	return llmagent.New(llmagent.Config{
		Name:        "filesystem_agent",
		Model:       llmModel,
		Description: "A helpful assistant that can list files and read their content from a sandboxed directory.",
		Instruction: instruction,
		Toolsets:    []tool.Toolset{tool.FilterToolset(toolset, tool.AllowedToolsPredicate(allowedTools))},
	})
}
