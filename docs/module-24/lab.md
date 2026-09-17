# Lab 24: A Golden-Path Trajectory Test for the Calculator Agent (Go) 🧪📊

## Goal

Build a Go test that records a "golden path" — a question, the exact ordered sequence of tool calls it should trigger, and what the final answer should contain — then proves that test can actually catch a wrong trajectory, not just pass by coincidence.

## Before You Start

This lab does **not** modify anything in `internal/agents/calculator` from Module 9 — `agent.go`, `tools.go`, `agent_test.go`, and `tools_test.go` all stay exactly as they were. You're adding one new, wholly separate file that reuses the already-shipped Calculator agent's public `BuildRootAgent`, the same "never touch the real lesson's files" discipline this repo has used for every bonus/bolt-on demonstration before.

## Lab Tasks

### 1. Read `internal/agents/calculator/agent_test.go` first

Specifically `askCalculator` and `assertCalculatorAdds` — this already proves one tool call happened with the right result, by checking the tool's own `FunctionResponse`, not just the final text. This lab's new file does the same kind of thing, just for a *sequence* of calls instead of one.

### 2. Read `internal/agents/calculator/golden_path_eval_test.go`

- `expectedToolCall`/`goldenPathCase` — a plain Go struct standing in for Python's `EvalCase`: a question, an ordered list of expected tool calls, and a substring the final answer must contain.
- `runGoldenPath` — drives `BuildRootAgent` through `runner.NewInMemory` + `Run()`, collecting **every** `FunctionCall` event in the order it arrives (not just one named tool, unlike `askCalculator`).
- `goldenPathCases` — two cases: a single-tool sanity check (`"What is 42 + 118?"` → `add` only), and the real trajectory case: `"First add 10 and 5. Then multiply that result by 2."` → `add(a=10,b=5)` then `multiply(a=15,b=2)`, in that exact order. `multiply`'s own arguments genuinely depend on `add`'s own result (15), so this isn't two calls that just happen to look sequential — a wrong order or wrong argument here is a real mistake, not a coincidence.

### 3. Run it — no `.env`, no API key 🖥️

```bash
go test ./internal/agents/calculator/... -run TestGoldenPath -v
```

Real, confirmed output from this exact command:

```
=== RUN   TestGoldenPath_Calculator_Ollama
=== RUN   TestGoldenPath_Calculator_Ollama/single_tool_call
=== RUN   TestGoldenPath_Calculator_Ollama/multi-step_trajectory:_add_then_multiply,_in_order
--- PASS: TestGoldenPath_Calculator_Ollama (20.23s)
    --- PASS: TestGoldenPath_Calculator_Ollama/single_tool_call (5.29s)
    --- PASS: TestGoldenPath_Calculator_Ollama/multi-step_trajectory:_add_then_multiply,_in_order (14.94s)
=== RUN   TestGoldenPath_Calculator_Gemini
    golden_path_eval_test.go:146: skipping: GOOGLE_AI_STUDIO_API_KEY is not set
--- SKIP: TestGoldenPath_Calculator_Gemini (0.00s)
PASS
```

### 4. Prove the assertion is real, not decorative

Don't just trust that the test *would* fail on a real mistake — watch it happen. Temporarily swap the two entries in the multi-step case's `wantTrajectory` (`multiply` first, `add` second) and re-run:

```bash
go test ./internal/agents/calculator/... -run TestGoldenPath_Calculator_Ollama -v
```

Real, confirmed failure output from that exact edit:

```
    golden_path_eval_test.go:134: trajectory[0].tool = "add", want "multiply" — the tool call sequence is out of order or wrong
    golden_path_eval_test.go:134: trajectory[0] (add) argument "a" = 10, want 15
    golden_path_eval_test.go:134: trajectory[0] (add) argument "b" = 5, want 2
    golden_path_eval_test.go:134: trajectory[1].tool = "multiply", want "add" — the tool call sequence is out of order or wrong
    golden_path_eval_test.go:134: trajectory[1] (multiply) argument "a" = 15, want 10
    golden_path_eval_test.go:134: trajectory[1] (multiply) argument "b" = 2, want 5
--- FAIL: TestGoldenPath_Calculator_Ollama (16.53s)
```

This is the same discipline Python's own Step 5 ("Test a Failure") walks through by temporarily breaking the `add` tool's math — here, the "break" is in the test's own expectation, proving the comparison logic itself is sound. **Revert the swap before moving on.**

### 5. Compare this to what Python's lab does instead

Python: open the Dev UI, have a conversation, click "Add current session to eval set," click "Run Evaluation," read a Pass/Fail card with a `tool_trajectory_score`. Go: write a struct literal and a loop. Same underlying question ("did the agent do the right things, in the right order, and say something reasonable?"), answered with the tools each language actually has.

## Self-Reflection Questions 🤔
- Why does `runGoldenPath` collect **every** `FunctionCall`, while `agent_test.go`'s own `askCalculator` only tracks one named tool's `FunctionResponse`? What would `askCalculator` miss that this lab's version catches?
- The multi-step case's `multiply` arguments (`15`, `2`) depend on `add`'s own result. Why does that dependency matter for proving the test isn't a coincidence?
- `assertGoldenPath` uses `strings.Contains` for the final answer, not exact equality. What would break if it used `==` instead?
- If Go ever ships a real evaluation package, what would you expect to change in this test file, and what would probably stay the same?

<hr/>

### Looking for the solution? 🔍

Hint: read `internal/agents/calculator/golden_path_eval_test.go` directly — the whole mechanism is one file, built on `BuildRootAgent` and `runner.NewInMemory`, both already familiar from Module 9.
