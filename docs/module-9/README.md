# Module 9: Creating Custom Function Tools (Go) 🛠️

## Theory

### From Built-in to Custom

Module 8's `google_search` runs entirely inside the model — nice, but you don't get a say in how it works. A **custom function tool** flips that: it's your own code, called locally by the ADK framework, with the result handed back to the model. This is how you plug an agent into a proprietary database, some business-logic algorithm, or — in this lab — basic arithmetic. 🧮

### `functiontool.New`: Go's Auto-Schema Mechanism ✨

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

`functiontool.New[TArgs, TResults](cfg, handler)` infers the tool's parameter *schema* straight from `TArgs`'s Go type via reflection (`github.com/google/jsonschema-go`) — pretty slick. `Name` and `Description` still need to be passed explicitly in `Config`, since Go has no runtime docstring to steal them from, but per-*parameter* descriptions come free from the `jsonschema:"..."` struct tag on each field.

### A Uniform Result Shape

All four calculator tools share one result type, `CalcResult{Status string; Result float64; Error string}` — they're all producing the same kind of answer, so why not? Here's the one gotcha: `Func[TArgs, TResults]` returns `(TResults, error)`, but a Go `error` return fails the *tool call itself* at the framework level — the LLM never even gets to reason about it. So division by zero needs `(CalcResult{Status: "error", Error: "division by zero"}, nil)` — a structured result the model can actually read and explain, not a Go `error` that short-circuits the whole thing.

**A real bug worth knowing about — found in review:** `Result`'s `json` tag must **not** have `omitempty`. Why? `omitempty` on a `float64` treats a genuine `0` exactly the same as "absent" — so `add(0, 0)` or `multiply(7, 0)` would silently hand the model a `{"status":"success"}` response with no `result` key at all, forcing it to guess the number itself. 😬 Confirmed via `json.Marshal(CalcResult{Status: "success", Result: 0})`: with `omitempty`, the `result` key just vanishes entirely.

### `agent.Context` Is Always There

Every custom function tool's signature is `func(agent.Context, TArgs) (TResults, error)` — `agent.Context` is unconditionally the first argument, whether a given tool actually uses it or not. None of this lab's four tools need session state, but the capability's always sitting there ready to go (module-10 puts it to real use).

### Custom Function Tools Work Through the Local Backend — Confirmed Live, No Cloud Fallback 🎉

Unlike module-7's vision agent and module-8's `google_search` agent, this module's tools need **zero Gemini requirement**. `functiontool.New` produces a plain `genai.FunctionDeclaration` — exactly the one tool shape `model/openaimodel/tools.go`'s `ensureFunctionToolOnly` already accepts (it only rejects *non-function* built-ins). Confirmed live: an `add` tool run against this repo's default local model (`qwen3.8:27b`) was genuinely invoked and returned the correct sum. `cmd/calculator` forces no backend — the local-first default just works, no strings attached.

### Going Further: Mixing a Built-in and a Custom Tool

Attaching both `geminitool.GoogleSearch{}` and a custom function tool to the same agent fails by default — a real restriction from the Gemini API itself, not something ADK is being weird about. There IS a real, narrower workaround (`IncludeServerSideToolInvocations`), confirmed live to work — see [troubleshooting.md](./troubleshooting.md) for the exact error and the fix. You don't need this for this module's own lab (function tools only), but keep it in your back pocket for when you want both tool types in one agent.

### Key Takeaways ✅
- `functiontool.New[TArgs, TResults](cfg, handler)` wraps a Go function as a tool — the parameter schema comes from `TArgs`'s type; name and description are yours to give explicitly.
- A division-by-zero (or any tool-level failure) belongs in the structured result (`CalcResult{Status: "error", ...}`), not a Go `error` — a Go `error` fails the tool call itself, giving the LLM nothing to work with.
- A numeric result field must never have an `omitempty` JSON tag — a genuine `0` is a valid answer, not an absent one, and `omitempty` would quietly hide it from the model.
- `agent.Context` is always the first parameter of a custom function tool.
- Custom function tools work through the local Ollama backend, confirmed live — the first tools module needing zero cloud fallback. 🎊
- Mixing a built-in and a custom function tool has a real, narrower workaround via `IncludeServerSideToolInvocations` — Developer-API-only, confirmed via source.

<hr/>

> **Coming from Python?** 🐍 `functiontool.New` plays the same role as passing a plain Python function into an agent's `tools` list, with one real difference: Go has no runtime docstring, so `Name`/`Description` are explicit instead of read from one. `CalcResult` is the direct equivalent of Python's dict result shape (`{"status": "success", "result": ...}`, or an error dict). Python's `ToolContext` is an opt-in extra parameter; Go's `agent.Context` is always there. And Python's docs call mixing a built-in and custom function tool impossible outside multi-agent systems — this Go SDK has a real, narrower workaround, likely added to the Gemini API after those docs were written.
