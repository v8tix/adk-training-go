# Module 24: Evaluating Agent Performance (Go) 🧪📊

## Theory

### Why "It Passed Once" Isn't Proof

`assert add(2, 2) == 4` works because `add` is deterministic. An LLM-backed agent isn't: ask it the same question twice and you might get "The result is 15" one time, "That comes out to 15" the next — both fine, but a plain string-equality check would reject the second one. And a passing final answer doesn't even mean the agent reasoned correctly: an easy sum like `42 + 118` is well within a "thinking" model's own arithmetic, so a correct-looking reply can happen *without the model ever calling your tool*. Evaluating an agent needs to check more than "did the text come out right."

### Trajectory Matters as Much as the Answer

The sequence of tool calls an agent makes — which tools, in what order, with what arguments — is its **trajectory**, and it's often the more important thing to check. Two agents can land on the identical final answer while one got there by calling the right tool with the right numbers and the other guessed. If a multi-step calculation needs `add`'s own result fed into `multiply` afterward, and the agent calls `multiply` first, its trajectory is wrong even if it somehow still produces a correct-looking number.

### Go's Answer: Write the Test That Proves It

There's no dedicated evaluation framework to reach for here — confirmed directly in the pinned SDK's own source, `server/adkrest/internal/routers/eval.go`:

> *"ADK Go has no evaluation implementation. The routes exist so the endpoints the web UI calls are recognised and answered deliberately, with 501 and a readable body... Use adk-python for eval workflows."*

Every eval-related REST endpoint is wired up but deliberately returns `501 Not Implemented`, and the one that does respond (`metrics-info`) returns an empty list — its own comment: *"which is none."* So the test **is** the tool. `internal/agents/calculator/golden_path_eval_test.go` records a "golden path" as a plain Go struct — a question, the expected ordered tool calls, and a substring the final answer must contain:

```go
type expectedToolCall struct {
    tool string
    args map[string]any
}

type goldenPathCase struct {
    name                 string
    question             string
    wantTrajectory       []expectedToolCall
    wantResponseContains string
}
```

A table-driven test drives the real agent through `runner.Run()`, collects every `FunctionCall` event in the order it arrives, and checks it against the expected trajectory field by field — tool name, then each argument, in sequence — plus a `strings.Contains` (not `==`) check on the final answer, since the exact wording shouldn't matter as long as the right number shows up.

### A Real, Proven Catch — Not Just a Plausible One

A test that can't fail on a real mistake proves nothing. This module's own test was checked the hard way: the two expected calls in its multi-step case (`add` then `multiply`) were deliberately swapped, and the test failed immediately with a clear, itemized diagnostic — wrong tool name and wrong arguments at both positions. Reverting the swap restored a clean pass. That's the real bar a trajectory check has to clear, confirmed live rather than assumed.

### Key Takeaways ✅
- A correct-looking final answer doesn't prove an agent reasoned correctly — it might not have called a tool at all.
- **Trajectory** (which tools, in what order, with what arguments) is often more revealing than the final answer alone.
- Go has no dedicated evaluation framework — confirmed directly from the SDK's own source, not inferred from its absence in `go doc`.
- The Go-idiomatic answer is the same one Go gives everywhere else: write a test that actually exercises the real event stream, and prove it can fail before trusting that it can pass.

<hr/>

> **Coming from Python?** 🐍 Python's own course teaches a full, dedicated evaluation framework built around this same trajectory/response-quality split: record a real conversation through the Dev UI's "Eval" tab, save it as an **Evaluation Case** inside an **Eval Set** (a `.evalset.json` file), then replay it later — after any prompt or tool change — via the Dev UI, the `adk eval` CLI, or `pytest` in CI, scored against metrics like `tool_trajectory_avg_score` (exact/in-order/any-order tool-call matching), `response_match_score`/`final_response_match_v2` (n-gram overlap or LLM-judged semantic equivalence), and `hallucinations_v1`/`safety_v1` (groundedness and harmful-content checks). A further layer, **User Simulation**, lets an LLM play the user against a `ConversationScenario` to stress-test an agent against conversations no fixed test case could anticipate. None of this — the file format, the metric registry, the Dev UI recording workflow, the simulator — has a Go equivalent; if you need that exact tooling, it lives only in `adk-python` today. What this module's own test gives you is the same underlying confidence (does the agent do the right things, in the right order, and say something reasonable?), just as a hand-written, compiled Go test instead of a declarative, LLM-scored one.
