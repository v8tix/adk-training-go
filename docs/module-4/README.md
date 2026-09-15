# Module 4: Core Agent Concepts: Agent Deep Dive (Go)

## Theory

### The "Brain" of the Operation

In Go, an agent's blueprint is `llmagent.Config`, the same type modules 2-3 already used:

* **`Name`:** A unique identifier for your agent.
* **`Model`:** The `model.LLM` powering the agent — built via `internal/infrastructure/llm.BuildModel` in this repo.
* **`Instruction`:** The most critical part — the detailed prompt defining the agent's persona, goals, and constraints.
* **`Description`:** A short summary of the agent's purpose.

### The Art of the Instruction

Write instructions the way you'd brief a new hire who can't ask follow-up questions: be clear and specific, use simple language, give an example for anything classification-style, and iterate once you see real output. `cmd/support-analyzer/prompts/support_analyzer_instruction.md` (Markdown, `# Instructions` / `# Constraints`) is this module's example — it enumerates the exact allowed values for `category` and `sentiment` rather than leaving them open-ended, the same "be explicit, don't assume the model infers your constraints" lesson module-3's echo instruction already taught.

### Two More Fields on `llmagent.Config`: Structured Output and State

Confirmed by reading `google.golang.org/adk/v2@v2.4.0`'s own source (not just `adk.dev`), `llmagent.Config` has two more fields worth knowing well: `OutputSchema`, which forces the model's final answer into a JSON shape you define, and `OutputKey`, which saves that answer into session state automatically.

#### 1. Enforcing JSON with `OutputSchema`

```go
schema := &genai.Schema{
    Type: genai.TypeObject,
    Properties: map[string]*genai.Schema{
        "category":  {Type: genai.TypeString},
        "sentiment": {Type: genai.TypeString},
        "summary":   {Type: genai.TypeString},
    },
    Required: []string{"category", "sentiment", "summary"},
}

analyzerAgent, err := llmagent.New(llmagent.Config{
    Name:         "support_analyzer_agent",
    Model:        llmModel,
    Instruction:  instruction,
    OutputSchema: schema, // Force JSON output
})
```

The SDK-sanctioned pattern here (used in the SDK's own `examples/multiagent/single_turn`) is a hand-written `*genai.Schema` literal, kept in sync by hand with whatever Go struct you define for typed access to the result (`SupportAnalysis` in this module's code) — `genai.InternalTSchema`/`InternalTJsonSchema` exist in the SDK but their own doc comment says "public only for internal purposes... external consumers must not use it," so they're not a shortcut here. Confirmed in source (`agent/llmagent/llmagent.go`, a literal `// TODO: add output schema validation and unmarshalling`): `OutputSchema` only shapes the *request* — the SDK never validates or parses the model's JSON reply. Your own code does that, the same way `cmd/support-analyzer`'s test does with `json.Unmarshal`.

`OutputSchema` and `Tools` can be set together — the agent can still call tools during its thought loop; only the final answer is constrained to the schema.

<hr/>

> **Coming from Python?** Python passes a Pydantic `BaseModel` and gets the schema *and* validation/parsing from the same declaration. Go has no equivalent auto-derivation — you write the `*genai.Schema` by hand and keep it in sync with your result struct yourself.

#### 2. Passing Data with `OutputKey`

```go
llmagent.Config{
    // ...
    OutputKey: "last_ticket_analysis", // Saves output to state["last_ticket_analysis"]
}
```

Confirmed in source: on the agent's final response event, the SDK concatenates all non-reasoning (`!Part.Thought`) text parts and writes the result into `event.Actions.StateDelta[OutputKey]`. This is directly inspectable from Go code driving the agent via `runner` — `cmd/support-analyzer`'s test reads `event.Actions.StateDelta["last_ticket_analysis"]` straight off the event stream, no separate session-service query needed (though `session.InMemoryService()` + `(Service).Get(...)` → `.Session.State()` is the equivalent path if you do need to inspect state from outside the run loop, e.g. after the fact).

### A Real Local-Inference Gotcha: Not Every Quantization Supports Structured Output

This module surfaced a genuine local-infra limitation, confirmed by testing (not assumed): some quantizations of this course's model family do **not** support JSON-schema-constrained decoding — Ollama itself returns `501 Not Implemented: "structured output is unavailable"` for any `OutputSchema`/`response_format: json_schema` request against them, confirmed on both `/v1/chat/completions` and `/v1/responses` (the endpoint `internal/infrastructure/llm`'s `openaimodel` adapter actually calls).

The fix stays entirely local, no cloud fallback needed: `qwen3.8:27b`, a GGUF `Q4_K_M` quantization of the same model family, **does** support it — confirmed with the identical request against the identical endpoints, `200 OK` with valid schema-conforming JSON both times, and confirmed working for plain-text modules too (echo-agent, verify-setup). Because of this, **`qwen3.8:27b` is now this repo's shared `OLLAMA_MODEL` default** (`internal/infrastructure/llm/config.go`) — no per-module override needed, here or in any earlier module.

### Key Takeaways
- `llmagent.Config{OutputSchema, OutputKey}` lets you force a schema-conforming JSON answer and automatically save it into session state — confirmed by reading the SDK's own source, not just its docs.
- There's no schema auto-derivation in Go: you hand-write a `*genai.Schema` and keep it in sync with your result struct yourself.
- The SDK shapes the *request* via `OutputSchema` but never validates or parses the *response* — your own code does that.
- `OutputKey` writes to `event.Actions.StateDelta`, directly inspectable from a `runner`-driven test, no session-service round trip required.
- Not every local model quantization supports structured output — verify empirically per model/quant, don't assume it "just works" because plain-text generation did.
