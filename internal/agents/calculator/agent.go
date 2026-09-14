// Package calculator defines the Calculator agent: it performs arithmetic
// via four custom function tools (add, subtract, multiply, divide),
// confirmed live (module-9) to work through the local Ollama backend — no
// cloud fallback needed, unlike internal/agents/visualcatalog (module-7) and
// internal/agents/researcher (module-8), which both require Gemini because
// of confirmed local-SDK limitations for their own tool/content types.
// Custom function tools produce a plain genai.FunctionDeclaration — exactly
// the one shape model/openaimodel/tools.go's ensureFunctionToolOnly already
// accepts.
package calculator

import (
	"embed"
	"io/fs"
	"log"

	"github.com/v8tix/adk-training-go/internal/infrastructure/prompts"
	"google.golang.org/adk/v2/agent"
	"google.golang.org/adk/v2/agent/llmagent"
	"google.golang.org/adk/v2/model"
	"google.golang.org/adk/v2/tool"
	"google.golang.org/adk/v2/tool/functiontool"
)

// PromptNamespace identifies this package's entries in the shared prompts
// cache (internal/infrastructure/prompts).
const PromptNamespace = "calculator"

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

// BuildRootAgent constructs the Calculator agent around llmModel, wrapping
// each of the four arithmetic functions as a custom function tool. Unlike
// internal/agents/researcher's built-in geminitool.GoogleSearch, llmModel
// can be either backend — see the package doc comment.
func BuildRootAgent(llmModel model.LLM) (agent.Agent, error) {
	instruction, err := prompts.Get(PromptNamespace + "/calculator_instruction")
	if err != nil {
		return nil, err
	}

	addTool, err := functiontool.New(functiontool.Config{
		Name:        "add",
		Description: "Adds two numbers together. Use this tool when the user asks to find the sum of two numbers.",
	}, add)
	if err != nil {
		return nil, err
	}
	subtractTool, err := functiontool.New(functiontool.Config{
		Name:        "subtract",
		Description: "Subtracts the second number from the first number. Use this tool when the user asks to find the difference between two numbers.",
	}, subtract)
	if err != nil {
		return nil, err
	}
	multiplyTool, err := functiontool.New(functiontool.Config{
		Name:        "multiply",
		Description: "Multiplies two numbers together. Use this tool when the user asks to find the product of two numbers.",
	}, multiply)
	if err != nil {
		return nil, err
	}
	divideTool, err := functiontool.New(functiontool.Config{
		Name:        "divide",
		Description: "Divides the first number by the second number. Use this tool when the user asks to divide one number by another.",
	}, divide)
	if err != nil {
		return nil, err
	}

	return llmagent.New(llmagent.Config{
		Name:        "calculator_agent",
		Model:       llmModel,
		Description: "An agent that performs arithmetic using custom function tools.",
		Instruction: instruction,
		Tools:       []tool.Tool{addTool, subtractTool, multiplyTool, divideTool},
	})
}
