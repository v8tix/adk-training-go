# Lab 9: Building a "Calculator" Agent (Go)

## Goal

Build an agent that performs basic arithmetic using custom function tools.

## Lab Tasks

### 1. Read `internal/agents/calculator/tools.go`

Four handler functions (`add`, `subtract`, `multiply`, `divide`), each `func(_ agent.Context, args XxxArgs) (CalcResult, error)`. Notice `divide` returns `CalcResult{Status: "error", Error: "division by zero"}, nil` for a zero denominator — a structured result the model can read and explain, not a Go `error` that would fail the tool call itself.

### 2. Read `internal/agents/calculator/agent.go`

Each handler is wrapped via `functiontool.New(functiontool.Config{Name, Description}, handler)`, then all four are attached via `Tools: []tool.Tool{addTool, subtractTool, multiplyTool, divideTool}`. Unlike `internal/agents/researcher` (module-8), there's no forced `MODEL_TYPE` — this agent works against the local default.

### 3. Run it — console mode, entirely locally

```bash
go run ./cmd/calculator console
```

No `.env`, no API key needed at all. Real, confirmed output from this exact command (thinking-model reasoning trimmed for readability):

```
🧮 calculator using qwen3.8:27b

User -> What is 42 + 118?
Agent -> 42 + 118 = **160**

User -> What is 10 divided by 0?
Agent -> I'm sorry, but division by zero is not defined in mathematics.
No number multiplied by 0 can give 10, so there is no valid result
for 10 ÷ 0. As the calculation tool confirmed, this operation
results in an error.

User -> What is the capital of France?
Agent -> I appreciate the question, but I'm a calculator assistant —
I can only help with arithmetic operations like addition,
subtraction, multiplication, and division. I'm not equipped
to answer general knowledge questions.
```

Three real behaviors confirmed working: a real calculation, a graceful divide-by-zero explanation (not a crash or a fabricated number), and a graceful decline of an off-topic question — all running entirely on the local model, zero cost.

### 4. Read `internal/agents/calculator/tools_test.go`

Table-driven unit tests against the four handlers directly — no LLM involved. These are the fast, deterministic base of this module's test pyramid; `agent_test.go` (below) covers the LLM actually choosing and invoking the tools correctly.

### 5. Read `internal/agents/calculator/agent_test.go`

`TestCalculator_Adds_Ollama` and `TestCalculator_Adds_Gemini` — the first tools module (after 7 and 8's cloud-only agents) with both variants genuinely passing, confirming this module's own finding that custom function tools need no cloud fallback.

## Self-Reflection Questions
- The docstring for each Python function is critical to the LLM's understanding. What's Go's equivalent, given Go functions have no runtime docstring? (Look at `functiontool.Config.Description` and the `jsonschema` struct tags in `tools.go`.)
- Why is it good practice for a tool function to return a structured `{"status": ...}` result instead of raising an exception (Python) or a Go `error` for an operation that can fail, like division?
- How would you add a new tool, like a `sqrt` function, to this agent? What would you need to write, and where?
- This module found that mixing `google_search` with a custom function tool has a real, narrower workaround in this Go SDK (`IncludeServerSideToolInvocations`) that Python's docs don't mention. Why might it be risky to rely on a finding like that in a real production agent, rather than the documented multi-agent workaround?

<hr/>

### Looking for the solution?

Hint: read `internal/agents/calculator/tools.go` and `agent.go` for the real mechanism — four tool functions and four `functiontool.New` calls are the entire implementation.
