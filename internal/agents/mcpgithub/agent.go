// Package mcpgithub defines the GitHub MCP agent: an llmagent whose tools
// come from GitHub's own hosted, remote MCP server over StreamableHTTP —
// the "Bonus" transport from module-27's lab, swapping the local stdio
// subprocess (mcpfilesystem) for a network call to a server GitHub runs and
// maintains. Matches Python's own "github_agent" role exactly.
package mcpgithub

import (
	"embed"
	"io/fs"
	"log"
	"net/http"

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
const PromptNamespace = "mcpgithub"

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

// githubEndpoint is GitHub's own hosted, remote MCP server — matching
// Python's own lab exactly.
const githubEndpoint = "https://api.githubcopilot.com/mcp/"

// githubAuthTransport sets the two headers GitHub's MCP server expects on
// every request. A small custom http.RoundTripper is enough for two static
// headers — no need for golang.org/x/oauth2's token-refresh machinery,
// which solves a problem (expiring/refreshable tokens) a fixed personal
// access token doesn't have.
type githubAuthTransport struct {
	token string
	base  http.RoundTripper
}

func (t *githubAuthTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req = req.Clone(req.Context())
	req.Header.Set("Authorization", "Bearer "+t.token)
	req.Header.Set("X-MCP-Readonly", "true")
	base := t.base
	if base == nil {
		base = http.DefaultTransport
	}
	return base.RoundTrip(req)
}

// BuildRootAgent constructs the GitHub MCP agent around llmModel,
// authenticated with githubToken. Like mcpfilesystem's BuildRootAgent, this
// never dials the server itself — mcptoolset.New's own doc comment confirms
// the MCP session is created lazily, on the first request to the LLM — so
// construction succeeds even for an empty or invalid token.
func BuildRootAgent(llmModel model.LLM, githubToken string) (agent.Agent, error) {
	instruction, err := prompts.Get(PromptNamespace + "/github_instruction")
	if err != nil {
		return nil, err
	}

	toolset, err := mcptoolset.New(mcptoolset.Config{
		Transport: &mcp.StreamableClientTransport{
			Endpoint:   githubEndpoint,
			HTTPClient: &http.Client{Transport: &githubAuthTransport{token: githubToken}},
		},
	})
	if err != nil {
		return nil, err
	}

	return llmagent.New(llmagent.Config{
		Name:        "github_agent",
		Model:       llmModel,
		Description: "A helpful assistant that gets information from GitHub repositories.",
		Instruction: instruction,
		Toolsets:    []tool.Toolset{toolset},
	})
}
