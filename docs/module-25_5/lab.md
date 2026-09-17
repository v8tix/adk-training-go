# Lab 25.5: Building a "Fail-Closed" PII Guardrail (Go) 🛡️🚫

## Goal

Build a Plugin that blocks a leaked credit-card-shaped string from ever reaching the user, demonstrating the Fail-Closed pattern, then prove it actually catches a real leak from a real agent — not just a hand-crafted test event.

## Lab Tasks

### 1. Read `internal/agents/piiguardrail/agent.go`

A tiny demo agent — no tools at all — instructed to reproduce a fixed, obviously-fake card number on a specific trigger phrase ("test data"). It exists purely to give the guardrail something real to catch, the same role Python's own `leak_agent` plays.

### 2. Read `internal/agents/piiguardrail/pii_guardrail_plugin.go`

`piiGuardrail`'s `onEvent` checks `event.IsFinalResponse()`, then loops over **every** part of the response — skipping any part marked `Thought` — looking for a match against a simple credit-card regex. On a match, it overwrites that part's text with a safety message and increments a mutex-guarded counter.

**Read the comment above the `Thought` check carefully** — this loop exists because the first version of this file only checked `Parts[0]`, and that quietly let a real leak through when the underlying model is thinking-capable (the default). See the README's own "A Real Bug This Module's Own Build Caught" section for the full story.

### 3. Run it — console mode, entirely locally 🖥️

```bash
go run ./cmd/pii-guardrail console
```

Real, confirmed output from this exact command (thinking-model reasoning trimmed for readability in the second and third turns):

```
🛡️  pii-guardrail using qwen3.8:27b

User -> Give me some test data.
🛑 [SAFETY] Response blocked due to policy violation (invocation "e-c266bc88-...", block #1)
Agent -> I'm sorry, but I can't share that information.

User -> What is the capital of Italy?
Agent -> The capital of Italy is Rome.
```

The first turn triggers the leak — and the block — before the user ever sees the card number. The second, unrelated turn passes through completely unaffected.

### 4. Read `internal/agents/piiguardrail/pii_guardrail_plugin_test.go`

Pure unit tests, no LLM. Two are worth special attention:

- `TestPIIGuardrail_BlocksRealAnswerAmongThoughtParts` — a hand-built two-part event (a thought part, then the real leaked answer) proves the guardrail correctly finds and blocks the *second* part, not the first.
- `TestPIIGuardrail_DoesNotBlockMatchWithinAThoughtPart` — the mirror case: a match *inside* a thought part is deliberately left untouched, since the guardrail's job is protecting the user-visible answer, not scrubbing internal reasoning.

### 5. Read `internal/agents/piiguardrail/agent_test.go`

`TestPIIGuardrail_BlocksLeakedCardNumber_{Ollama,Gemini}` drives a real `runner.New` with the real agent and the real plugin wired into `PluginConfig.Plugins`, then two separate turns: one that triggers the leak (asserting the real card number never appears in the user-visible answer, and that the plugin's own `blockedCount` reflects a genuine interception), and one ordinary question (asserting the guardrail did *not* fire) — proving both the positive and negative case structurally, not by eyeballing console output.

## Self-Reflection Questions 🤔
- Why does `onEvent` skip parts marked `Thought` instead of scanning every part indiscriminately? What would go wrong (or right) if it didn't?
- The guardrail's regex is a simple pattern — what would it take to catch a credit card number written with spaces instead of dashes, and is that a change to the plugin or to the pattern alone?
- Why is `blockedCount` guarded by a mutex here, given this module's own lab never actually runs concurrent requests? What real deployment scenario makes that mutex load-bearing anyway?
- If you wanted this same guardrail to protect a second, completely different agent, what would you need to change in that agent's own code?

<hr/>

### Looking for the solution? 🔍

Hint: read `internal/agents/piiguardrail/pii_guardrail_plugin.go` and `agent.go` for the real mechanism — a regex, a loop over the real response's parts, and a plugin registered separately from the agent itself in `cmd/pii-guardrail/main.go`.
