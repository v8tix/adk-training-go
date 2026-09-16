# Lab 9: Building a "Calculator" Agent (Go) 🧮

## Goal

Build an agent that does basic arithmetic using custom function tools.

## Lab Tasks

### 1. Read `internal/agents/calculator/tools.go`

Four handler functions (`add`, `subtract`, `multiply`, `divide`), each `func(_ agent.Context, args XxxArgs) (CalcResult, error)`. Notice `divide` returns `CalcResult{Status: "error", Error: "division by zero"}, nil` for a zero denominator — a structured result the model can read and explain, not a Go `error` that would blow up the tool call itself.

### 2. Read `internal/agents/calculator/agent.go`

Each handler gets wrapped via `functiontool.New(functiontool.Config{Name, Description}, handler)`, then all four attach via `Tools: []tool.Tool{addTool, subtractTool, multiplyTool, divideTool}`. Unlike `internal/agents/researcher` (module-8), there's no forced `MODEL_TYPE` here — this agent's happy running against the local default.

### 3. Run it — console mode, entirely locally 🖥️

```bash
go run ./cmd/calculator console
```

No `.env`, no API key, nothing. Real, confirmed output from this exact command (thinking-model reasoning trimmed for readability):

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

Three behaviors confirmed working: a real calculation, a graceful divide-by-zero explanation (not a crash or a made-up number), and a graceful decline of an off-topic question — all on the local model, zero cost. 🎉

### 4. Read `internal/agents/calculator/tools_test.go`

Table-driven unit tests hitting the four handlers directly — no LLM involved. This is the fast, deterministic base of this module's test pyramid; `agent_test.go` (next up) covers the LLM actually picking and calling the right tools.

### 5. Read `internal/agents/calculator/agent_test.go`

`TestCalculator_Adds_Ollama` and `TestCalculator_Adds_Gemini` — the first tools module (after 7 and 8's cloud-only agents) where both variants genuinely pass, confirming this module's own big finding: custom function tools need zero cloud fallback.

## Self-Reflection Questions 🤔
- Python leans hard on a function's docstring for the LLM to understand it. What's Go's equivalent, given Go functions don't have a runtime docstring at all? (Peek at `functiontool.Config.Description` and the `jsonschema` struct tags in `tools.go`.)
- Why's it good practice for a tool function to return a structured `{"status": ...}` result instead of raising an exception (Python) or a Go `error` for something that can fail, like division?
- How would you bolt on a new tool — say, `sqrt` — to this agent? What would you write, and where?
- This module found that mixing `google_search` with a custom function tool has a real, narrower workaround in this Go SDK (`IncludeServerSideToolInvocations`) that Python's docs don't even mention. Why might leaning on a finding like that in a real production agent be risky, compared to the documented multi-agent workaround?

<hr/>

### Looking for the solution? 🔍

Hint: read `internal/agents/calculator/tools.go` and `agent.go` for the real mechanism — four tool functions and four `functiontool.New` calls, and that's the whole implementation.
