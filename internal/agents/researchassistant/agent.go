// Package researchassistant defines three agents around the ADK's built-in
// google_search tool, all requiring Gemini — confirmed live (module-8) that
// the local Ollama backend's Go client (model/openaimodel) rejects any
// non-function tool, including this built-in one, before ever reaching the
// network.
//
// The Gemini API forbids mixing google_search with custom function tools in
// one request unless genai.ToolConfig.IncludeServerSideToolInvocations is
// set: without it, a combined request fails with "400 INVALID_ARGUMENT:
// Please enable tool_config.include_server_side_tool_invocations to use
// Built-in tools with Function calling." Two agents (BuildResearchAgent, BuildFormatterAgent)
// mirror Python's own sequential-composition workaround for that default
// restriction. A third (BuildCombinedAgent) sets the flag and combines both
// tool types in one agent — real, working Go-SDK behavior Python's own
// module never describes, confirmed live with real grounding metadata and a
// real custom-tool call in the same conversation. A fourth (BuildMapsGroundingAgent)
// builds the google_maps_grounding built-in tool against the Vertex AI
// backend, since genai.GoogleMaps requires it. See docs/module-12/README.md.
package researchassistant

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
	"google.golang.org/adk/v2/tool/geminitool"
	"google.golang.org/genai"
)

// PromptNamespace identifies this package's entries in the shared prompts
// cache (internal/infrastructure/prompts).
const PromptNamespace = "research-assistant"

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

func customTools() ([]tool.Tool, error) {
	extractFacts, err := functiontool.New(functiontool.Config{
		Name:        "extract_key_facts",
		Description: "Extracts key sentences from a block of text.",
	}, extractKeyFacts)
	if err != nil {
		return nil, err
	}
	formatNotes, err := functiontool.New(functiontool.Config{
		Name:        "format_research_notes",
		Description: "Formats research findings into a structured document.",
	}, formatResearchNotes)
	if err != nil {
		return nil, err
	}
	return []tool.Tool{extractFacts, formatNotes}, nil
}

// BuildResearchAgent constructs the search-only agent: it can use
// google_search, and nothing else. Mirrors Python's research_agent.
func BuildResearchAgent(llmModel model.LLM) (agent.Agent, error) {
	instruction, err := prompts.Get(PromptNamespace + "/research_instruction")
	if err != nil {
		return nil, err
	}
	return llmagent.New(llmagent.Config{
		Name:        "research_agent",
		Model:       llmModel,
		Description: "Searches the web to research a topic.",
		Instruction: instruction,
		Tools:       []tool.Tool{geminitool.GoogleSearch{}},
	})
}

// BuildFormatterAgent constructs the custom-tools-only agent: it never
// touches google_search, only extract_key_facts and format_research_notes.
// Mirrors Python's formatter_agent.
func BuildFormatterAgent(llmModel model.LLM) (agent.Agent, error) {
	instruction, err := prompts.Get(PromptNamespace + "/formatter_instruction")
	if err != nil {
		return nil, err
	}
	tools, err := customTools()
	if err != nil {
		return nil, err
	}
	return llmagent.New(llmagent.Config{
		Name:        "formatter_agent",
		Model:       llmModel,
		Description: "Extracts key facts from research findings and formats them into a report.",
		Instruction: instruction,
		Tools:       tools,
	})
}

// BuildMapsGroundingAgent constructs an agent around the google_maps_grounding
// built-in tool — real (geminitool.New + genai.GoogleMaps), but requiring the
// Vertex AI backend rather than the public Gemini API (genai.GoogleMaps's own
// doc comment: "This field is not supported in Gemini API"). llmModel must be
// built against genai.BackendVertexAI (llm.ModelTypeVertexAI) — this function
// itself doesn't care which backend built llmModel, but the tool call will
// fail against a plain Gemini API model. Previously documented in
// docs/module-12/README.md as a real construction this course didn't build or
// test, for lack of Vertex AI access; built and tested here once that access
// became available.
func BuildMapsGroundingAgent(llmModel model.LLM) (agent.Agent, error) {
	instruction, err := prompts.Get(PromptNamespace + "/maps_grounding_instruction")
	if err != nil {
		return nil, err
	}
	mapsGrounding := geminitool.New(
		"google_maps_grounding",
		"Answers location-based questions using Google Maps.",
		&genai.Tool{GoogleMaps: &genai.GoogleMaps{}},
	)
	return llmagent.New(llmagent.Config{
		Name:        "maps_grounding_agent",
		Model:       llmModel,
		Description: "Answers location-based questions, grounded in real Google Maps data.",
		Instruction: instruction,
		Tools:       []tool.Tool{mapsGrounding},
	})
}

// BuildCombinedAgent constructs a single agent carrying both google_search
// and the two custom tools, enabled by setting
// GenerateContentConfig.ToolConfig.IncludeServerSideToolInvocations — the
// Go-SDK-exposed alternative to sequential composition documented in the
// package doc comment. Has no Python equivalent in this course's curriculum.
func BuildCombinedAgent(llmModel model.LLM) (agent.Agent, error) {
	instruction, err := prompts.Get(PromptNamespace + "/combined_instruction")
	if err != nil {
		return nil, err
	}
	tools, err := customTools()
	if err != nil {
		return nil, err
	}
	tools = append([]tool.Tool{geminitool.GoogleSearch{}}, tools...)

	includeServerSide := true
	return llmagent.New(llmagent.Config{
		Name:        "combined_research_agent",
		Model:       llmModel,
		Description: "Researches a topic with google_search and formats the findings, all in one agent.",
		Instruction: instruction,
		Tools:       tools,
		GenerateContentConfig: &genai.GenerateContentConfig{
			ToolConfig: &genai.ToolConfig{
				IncludeServerSideToolInvocations: &includeServerSide,
			},
		},
	})
}
