// Package researcher defines the Researcher agent: it answers questions
// about current events by using the ADK's built-in google_search tool,
// unlike every prior module's agent, which could only reason over its own
// training data. Like internal/agents/visualcatalog, this agent returns
// plain text — Python's lab version doesn't set an output_schema either.
//
// google_search is a Gemini-native built-in tool that runs inside the model
// itself, not a local function the ADK framework calls — confirmed live
// (module-8): the local Ollama backend's Go client (model/openaimodel)
// unconditionally rejects any non-function tool, including this one
// ("openai: non-function tools are not supported"), the same category of
// gap module-7 found for image Parts. So this agent requires
// MODEL_TYPE=gemini — see cmd/researcher and docs/module-8/README.md for
// the full finding, including the confirmed live result that the plain
// GOOGLE_AI_STUDIO_API_KEY path (no Vertex AI) is sufficient.
package researcher

import (
	"embed"
	"io/fs"
	"log"

	"github.com/v8tix/adk-training-go/internal/infrastructure/prompts"
	"google.golang.org/adk/v2/agent"
	"google.golang.org/adk/v2/agent/llmagent"
	"google.golang.org/adk/v2/model"
	"google.golang.org/adk/v2/tool"
	"google.golang.org/adk/v2/tool/geminitool"
)

// PromptNamespace identifies this package's entries in the shared prompts
// cache (internal/infrastructure/prompts).
const PromptNamespace = "researcher"

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

// BuildRootAgent constructs the Researcher agent around llmModel, resolving
// its own instruction from the shared prompts cache. llmModel must be able
// to accept the built-in geminitool.GoogleSearch tool — see the package doc
// comment.
func BuildRootAgent(llmModel model.LLM) (agent.Agent, error) {
	instruction, err := prompts.Get(PromptNamespace + "/researcher_instruction")
	if err != nil {
		return nil, err
	}
	return llmagent.New(llmagent.Config{
		Name:        "researcher_agent",
		Model:       llmModel,
		Description: "Answers questions about current events using Google Search.",
		Instruction: instruction,
		Tools:       []tool.Tool{geminitool.GoogleSearch{}},
	})
}
