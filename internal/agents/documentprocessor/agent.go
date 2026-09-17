// Package documentprocessor defines the Document Processor agent: a
// four-step pipeline (extract, summarize, chart, report) where each step
// reads and writes versioned artifacts through agent.Context.Artifacts() —
// the same one-context pattern module-22 confirmed for session state
// (agent.Context.State()), just for files instead of key-value data.
//
// Confirmed live this module: Go's artifact backends (InMemoryService and
// gcsartifact) both number a file's first save as version 1, not version 0
// as Python's own docs describe — traced directly in their Save
// implementations, not assumed from either language's documentation.
package documentprocessor

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
const PromptNamespace = "documentprocessor"

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

// BuildRootAgent constructs the Document Processor agent around llmModel,
// wrapping all four pipeline-step tools.
func BuildRootAgent(llmModel model.LLM) (agent.Agent, error) {
	instruction, err := prompts.Get(PromptNamespace + "/document_processor_instruction")
	if err != nil {
		return nil, err
	}

	extractTextTool, err := functiontool.New(functiontool.Config{
		Name:        "extract_text",
		Description: "Extracts and cleans a document's text, saving it as a new artifact version.",
	}, extractText)
	if err != nil {
		return nil, err
	}
	summarizeTool, err := functiontool.New(functiontool.Config{
		Name:        "summarize_document",
		Description: "Summarizes a document's previously extracted text into a new artifact.",
	}, summarizeDocument)
	if err != nil {
		return nil, err
	}
	generateChartTool, err := functiontool.New(functiontool.Config{
		Name:        "generate_chart",
		Description: "Generates a visual chart of a document's stats, saved as a binary artifact.",
	}, generateChart)
	if err != nil {
		return nil, err
	}
	createReportTool, err := functiontool.New(functiontool.Config{
		Name:        "create_report",
		Description: "Compiles a document's extracted text, summary, and chart into a final report artifact.",
	}, createReport)
	if err != nil {
		return nil, err
	}

	return llmagent.New(llmagent.Config{
		Name:        "document_processor_agent",
		Model:       llmModel,
		Description: "A document processing pipeline that extracts, summarizes, charts, and reports on a document, using versioned artifacts at every step.",
		Instruction: instruction,
		Tools: []tool.Tool{
			extractTextTool,
			summarizeTool,
			generateChartTool,
			createReportTool,
		},
	})
}
