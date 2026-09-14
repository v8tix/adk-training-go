# Module 9: Creating Custom Function Tools (Go)

## Theory

### From Built-in to Custom

Module 8's `google_search` runs entirely inside the model. A **custom function tool** is the opposite: your own code, which the ADK framework calls locally and hands the result back to the model. This is how you connect an agent to a proprietary database, a business-logic algorithm, or — in this lab — basic arithmetic.

### `functiontool.New`: Go's Auto-Schema Mechanism

```go
addTool, err := functiontool.New(functiontool.Config{
    Name:        "add",
    Description: "Adds two numbers together. Use this tool when the user asks to find the sum of two numbers.",
}, add)
```

```go
type AddArgs struct {
    A int `json:"a" jsonschema:"the first number"`
    B int `json:"b" jsonschema:"the second number"`
}

func add(_ agent.Context, args AddArgs) (CalcResult, error) {
    return CalcResult{Status: "success", Result: float64(args.A + args.B)}, nil
}
```

`functiontool.New[TArgs, TResults](cfg, handler)` infers the tool's parameter *schema* from `TArgs`'s Go type via reflection (`github.com/google/jsonschema-go`) — the same idea as Python's auto-schema-from-type-hints. **One real, precise difference: Go has no runtime docstring**, so the tool's `Name` and `Description` must be passed explicitly in `Config`, where Python pulls both straight from the function's own docstring. Per-*parameter* descriptions do have a direct equivalent, though: the `jsonschema:"..."` struct tag is Go's version of Python's docstring `Args:` line for that parameter.

### The Uniform Result Shape

Python's lab returns a dict (`{"status": "success", "result": ...}`, or an error dict for division by zero). This repo's `CalcResult{Status string; Result float64; Error string}` is the direct Go equivalent — one shared struct, since all four tools produce the same kind of result. The one subtlety: `Func[TArgs, TResults]` returns `(TResults, error)`, but a Go `error` return fails the *tool call itself* at the framework level — the LLM never sees it as something to reason about. Division by zero must return `(CalcResult{Status: "error", Error: "division by zero"}, nil)`, a structured result, not a Go `error`, matching Python's own "return an error dictionary" instruction exactly.

**A real bug worth knowing about, found in review:** `Result`'s `json` tag must **not** have `omitempty`. `omitempty` on a `float64` treats a genuine `0` the same as "absent" — so `add(0, 0)` or `multiply(7, 0)` would silently hand the model a `{"status":"success"}` response with no `result` key at all, forcing it to guess the number itself. Confirmed via `json.Marshal(CalcResult{Status: "success", Result: 0})`: with `omitempty`, the `result` key vanishes entirely.

### `agent.Context`: Always There, Not Opt-In

Python's `ToolContext` (for session-state access) is an optional extra parameter you add only when you need it. Go's `functiontool.Func[TArgs, TResults]` signature is `func(agent.Context, TArgs) (TResults, error)` unconditionally — every custom function tool always receives `agent.Context` as its first argument, whether it uses it or not. None of this lab's four tools need session state, but the capability is always present, never something to remember to add later.

### Custom Function Tools Work Through the Local Backend — Confirmed Live, No Cloud Fallback

Unlike module-7's vision agent and module-8's `google_search` agent, this module's tools need **no Gemini requirement at all**. `functiontool.New` produces a plain `genai.FunctionDeclaration` — exactly the one tool shape `model/openaimodel/tools.go`'s `ensureFunctionToolOnly` already accepts (it only rejects *non-function* built-ins). Confirmed live: an `add` tool run against this repo's default local model (`qwen3.8:27b`) was genuinely invoked and returned the correct sum. `cmd/calculator` therefore forces no backend — the local-first default just works.

### Going Further: Mixing Built-in and Custom Tools — a Real Update Over Python's Docs

Python's README states mixing `google_search` with a custom function tool always fails (`400 INVALID_ARGUMENT: Multiple tools are supported only when they are all search tools.`) and recommends multi-agent systems (module 15) as the only fix. Confirmed live in this Go SDK: the default behavior does fail the same way, but with a more specific error:

```
400 ... Please enable tool_config.include_server_side_tool_invocations
to use Built-in tools with Function calling.
```

Doing exactly that — setting `llmagent.Config.GenerateContentConfig.ToolConfig.IncludeServerSideToolInvocations = true` — made the same mixed-tool agent work, correctly computing a real sum with both a built-in and a custom tool attached. Confirmed via `genai` source that this field is **Developer-API-only** (`"This field is not supported in Vertex AI"`) — the workaround is available specifically through this repo's existing simpler `GOOGLE_AI_STUDIO_API_KEY` path. This isn't required for this module's own lab (which uses only function tools), and it doesn't replace module 15's multi-agent lesson — it's a real, narrower option worth knowing about, likely added to the Gemini API after Python's docs were written.

### Key Takeaways
- `functiontool.New[TArgs, TResults](cfg, handler)` is Go's direct equivalent of passing a Python function into an agent's `tools` list — the parameter schema is inferred from `TArgs`'s type; the name and description must be given explicitly (no docstring to read at runtime).
- A division-by-zero (or any tool-level failure) belongs in the structured result (`CalcResult{Status: "error", ...}`), not a Go `error` — a Go `error` fails the tool call itself, giving the LLM nothing to reason about.
- A numeric result field must not have an `omitempty` JSON tag — a genuine `0` is a valid answer, not an absent one, and `omitempty` would silently drop it from what the model sees.
- `agent.Context` is always the first parameter of a custom function tool in Go, unlike Python's opt-in `ToolContext`.
- Custom function tools work through the local Ollama backend, confirmed live — this is the first tools module needing no cloud fallback.
- Mixing a built-in and a custom function tool, which Python's docs call impossible outside multi-agent systems, has a real, narrower workaround in this Go SDK via `IncludeServerSideToolInvocations` — Developer-API-only, confirmed via source.
