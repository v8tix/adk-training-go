# Module 12: Built-in Tools and Grounding (Go) 🌐

## Theory

### A Tool That Runs Inside the Model

Every tool you've built so far — `internal/agents/calculator`'s arithmetic functions, `internal/agents/marketanalyst`'s currency lookup — is code the ADK framework calls on your behalf, locally, in your own process. A **built-in tool** flips that entirely: it's a capability the model itself runs, inside Google's own infrastructure, with zero local code execution. `google_search` is one: give a Gemini 2.0+ model this tool, and it can decide, on its own, to search the live web before answering. 🔍

Adding it is a one-line change to an agent's `Tools`:

```go
import (
    "google.golang.org/adk/v2/agent/llmagent"
    "google.golang.org/adk/v2/tool"
    "google.golang.org/adk/v2/tool/geminitool"
)

rootAgent, err := llmagent.New(llmagent.Config{
    Name:        "research_agent",
    Model:       llmModel,
    Instruction: "Use google_search to find current information, then summarize it.",
    Tools:       []tool.Tool{geminitool.GoogleSearch{}},
})
```

`geminitool.GoogleSearch{}` is a zero-field struct satisfying the same `tool.Tool` interface every custom tool does (`Name`, `Description`, `ProcessRequest`) — it just adds a `genai.Tool{GoogleSearch: &genai.GoogleSearch{}}` entry to the request instead of a JSON function schema. `internal/agents/researcher` (module-8) already builds an agent around exactly this tool; this module takes it one step further: what happens when you *also* want your own custom tools in the mix?

### The Gemini API's Real Restriction

Try adding a custom function tool to that same agent, and the request itself fails — not at construction time, but the moment the model actually runs:

```
400 INVALID_ARGUMENT: Please enable tool_config.include_server_side_tool_invocations
to use Built-in tools with Function calling.
```

Confirmed live this module: this is a genuine Gemini API constraint, not something the ADK framework — Go or Python — imposes or could relax on its own. By default, a request can carry `google_search` *or* function-declared tools, never both. 🚫

### Lifting the Restriction, or Working Around It

The error message above straight-up names its own fix. Setting `genai.ToolConfig.IncludeServerSideToolInvocations` genuinely lifts the restriction — confirmed live: one agent, carrying both `geminitool.GoogleSearch{}` and two custom function tools, successfully searched the web (real `GroundingMetadata`, three grounding chunks) *and* called a custom tool, in the same conversation. 🎉

```go
includeServerSide := true
combinedAgent, err := llmagent.New(llmagent.Config{
    Name:        "combined_research_agent",
    Model:       llmModel,
    Instruction: "Search for the topic, then extract and format the findings.",
    Tools:       []tool.Tool{geminitool.GoogleSearch{}, extractFactsTool, formatNotesTool},
    GenerateContentConfig: &genai.GenerateContentConfig{
        ToolConfig: &genai.ToolConfig{
            IncludeServerSideToolInvocations: &includeServerSide,
        },
    },
})
```

`llmagent.Config.GenerateContentConfig` is a direct passthrough to `genai.GenerateContentConfig` — the exact same struct you'd hand the underlying Gemini client — so this needs zero extra wiring beyond setting one field. See `internal/agents/researchassistant.BuildCombinedAgent` for the working version, and `TestCombinedAgent_UsesSearchAndCustomTool_Gemini` for the live proof.

The other option: **sequential composition**. Keep the restriction in place, use two separate agents instead — one with only `google_search`, one with only your custom tools — calling the first, then feeding its plain-text output into the second as input. This is still a totally legitimate choice even now that the combined approach works: it keeps each agent's tool surface minimal and its own responsibility, and it's what this module's lab builds by hand:

```go
findings, err := runAgent(ctx, researchAgent, "research_app", "Research this topic: "+topic)
report, err := runAgent(ctx, formatterAgent, "formatter_app", "Topic: "+topic+"\n\nFindings: "+findings)
```

Two `runner.Run` calls, one per agent, in your own code — the same programmatic-execution shape module-6 and module-10 already used for other reasons.

### `google_maps_grounding`: A Real Construction, Not a Built Lab

Gemini also has a location-grounding built-in tool, `google_maps_grounding`, for questions like "what's near me" or "how do I get there." `geminitool` doesn't ship it as its own named type the way it does `GoogleSearch`, but its own package doc comment tells you exactly how to add any Gemini-native tool: `geminitool.New(name, description, &genai.Tool{...})`. `genai.GoogleMaps` is a real, present type, so this genuinely compiles:

```go
mapsGrounding := geminitool.New(
    "google_maps_grounding",
    "Answers location-based questions using Google Maps.",
    &genai.Tool{GoogleMaps: &genai.GoogleMaps{}},
)
```

This module doesn't build or test it though, matching this course's own scope here — `google_maps_grounding` needs the Vertex AI API, and this repo's setup only ever configures a plain AI Studio key.

### Key Takeaways ✅
- A built-in tool like `google_search` runs inside the model's own environment — no local code execution, added with one line: `Tools: []tool.Tool{geminitool.GoogleSearch{}}`.
- The Gemini API rejects mixing a built-in tool with custom function tools by default — confirmed live with the real `400 INVALID_ARGUMENT` error.
- Setting `genai.ToolConfig.IncludeServerSideToolInvocations` on `llmagent.Config.GenerateContentConfig` lifts that restriction, letting one agent genuinely use both kinds of tool together — confirmed live with real grounding metadata and a real custom tool call in the same conversation.
- Sequential composition — a search-only agent's output feeding a custom-tools-only agent, via two separate `runner.Run` calls — stays a legitimate, simpler-scoped alternative even now that the combined approach works.
- `google_maps_grounding` is real (`geminitool.New` + `genai.GoogleMaps`) but needs Vertex AI, outside this course's scope.
- There's no `ManagedAgent` type anywhere in the pinned `google.golang.org/adk/v2 v2.4.0` source — a confirmed scoping gap in this SDK version, not a pattern to build against yet.

<hr/>

> **Coming from Python?** 🐍 Python's own module-12 describes the mixed-tools restriction as an absolute one, worked around only by splitting into two agents — it never mentions `include_server_side_tool_invocations`. That flag exists at the Gemini API level, not just in this Go SDK, so the same combined-agent approach should be reachable from Python too; it just isn't part of this course's Python curriculum. Treat the two-agent pattern as the one this course teaches on both sides, and the combined agent as a genuine bonus this Go module surfaces. Separately, Python's ADK also has a preview `ManagedAgent` for Google's server-hosted first-party agents — this Go SDK version has no equivalent yet, per the Key Takeaways above.
