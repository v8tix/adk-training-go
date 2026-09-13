# Lab 6: Programmatic Execution: Apps and Runners (Go)

## Goal

Move beyond the CLI (module-5) and trigger the Support Analyzer agent from your own Go code — build one `*runner.Runner` and drive two independent users through it, proving session isolation, exactly the shape a real backend needs.

## Lab Tasks

### 1. Read `internal/agents/supportanalyzer/agent.go`

This is where the agent's definition now lives — moved out of `cmd/support-analyzer` this module so more than one program can build it. `BuildRootAgent(llmModel)` is the whole public surface: give it a `model.LLM`, get back a ready `agent.Agent`. It resolves its own prompt internally, so callers never need to know the prompts-cache key.

### 2. Read `cmd/support-analyzer-runner/main.go`

No `cmd/launcher` import at all — this is a plain Go program, mirroring Python's `main.py` skeleton exactly:

```go
r, _ := runner.NewInMemory("support_analyzer_runner_app", rootAgent)

fmt.Println("--- User A (Alice) ---")
aliceResult, _ := runOnce(ctx, r, "alice", "alice_session", "I was overcharged $50")
fmt.Printf("Agent Response: %s\n", aliceResult)

fmt.Println("\n--- User B (Bob) ---")
bobResult, _ := runOnce(ctx, r, "bob", "bob_session", "My wifi is slow")
fmt.Printf("Agent Response: %s\n", bobResult)
```

`runOnce` is this repo's standard "drive one message through `Run`'s iterator, return the final structured result" helper — there's no `run_debug()` in Go to reach for instead (see the README for why).

### 3. Run it

```bash
go run ./cmd/support-analyzer-runner
```

Real, confirmed output from this exact command:

```
🎫 support-analyzer-runner using qwen3.8:27b
--- User A (Alice) ---
Agent Response: {"category": "billing", "sentiment": "negative", "summary": "The customer reports being overcharged $50 on a charge."}

--- User B (Bob) ---
Agent Response: {"category": "technical", "sentiment": "negative", "summary": "The customer reports that their wifi connection is slow."}
```

Two distinct, correct analyses — Alice's billing complaint categorized as `billing`, Bob's wifi issue as `technical` — both served by the *same* `*runner.Runner` instance. Wording will vary between runs (LLM output isn't reproducible token-for-token), but the category/sentiment shape and the isolation between the two users will hold.

### 4. Verify no regression in `cmd/support-analyzer`

The CLI/launcher entrypoint from modules 4-5 is unchanged in behavior — it just imports `internal/agents/supportanalyzer` now instead of defining the agent locally:

```bash
go run ./cmd/support-analyzer console
```

Should behave exactly as it did before this module.

## Self-Reflection Questions
- Why is Go's `runner.Config` considered a merge of Python's `App` and `Runner` concepts rather than a missing `App` feature?
- What would happen if `cmd/support-analyzer-runner` reused the same `sessionID` for both Alice and Bob instead of separate ones?
- In a real Go web server (e.g. built on `net/http`), where would you construct the `*runner.Runner` — inside each request handler, or once at startup as a shared variable? Why does that answer match Python's own guidance for its `Runner` singleton?
- This module extracted `internal/agents/supportanalyzer` specifically because Go can't have two `func main()`s in one directory. What earlier module's own code would have needed the same treatment if it had gained a second entrypoint?

<hr/>

### Looking for the solution?

Hint: read `internal/agents/supportanalyzer/agent.go` (`BuildRootAgent`) and `cmd/support-analyzer-runner/main.go` (`runOnce`, and the two `runOnce` calls for Alice and Bob) — that's the whole mechanism, end to end.
