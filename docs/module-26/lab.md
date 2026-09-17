# Lab 26: Building a Content Moderator with Caching, Guardrails, and Audits (Go) 🚦🧯

## Goal

Build all six callback slots for one agent: per-question response caching, an input guardrail, output redaction, tool-argument validation, and a tool-output audit — then prove all three of the real, observable behaviors this combination should produce.

## Lab Tasks

### 1. Read `internal/agents/contentmoderator/tools.go`

`generateText` — a trivial demo tool, matching Python's own lab exactly. This module's real point is the callbacks around it, not the tool's own logic.

### 2. Read `internal/agents/contentmoderator/callbacks.go`

All six callbacks live here, plus two shared helpers. Read the comments above `responseCache` and `beforeModelCallback` carefully — they explain two real bugs found live while building this module:

- `responseCache.beforeAgentCallback`/`afterAgentCallback` — the caching pair. `afterAgentCallback` reads `outputKey` from state (written automatically by `llmagent.Config.OutputKey`), **not** by walking `ctx.Session().Events()` the way Python's own lab does — a callback context in Go doesn't support `Session()` at all.
- `beforeModelCallback` — the input guardrail. It checks only `llmRequest.Contents`' own *last* entry (the current turn), not the whole conversation history — the fix for a real bug where one blocked word early in a session permanently refused every later turn.
- `afterModelCallback` — output redaction, checking every non-`Thought` part (module-25.5's own lesson, applied here from the start).
- `beforeToolCallback`/`afterToolCallback` — argument validation and output audit, the same shape modules 22/25 already established.

### 3. Read `internal/agents/contentmoderator/agent.go`

Notice `buildRootAgent` takes an explicit `*responseCache` parameter — this lets the live test read `cache.hitCount` directly to prove a cache hit really happened, rather than just comparing text (an LLM could coincidentally repeat itself even on a genuine miss). `BuildRootAgent`, the function every `cmd/` program actually calls, just supplies a fresh one.

### 4. Run it — console mode, entirely locally 🖥️

```bash
go run ./cmd/content-moderator console
```

Real, confirmed output from this exact four-turn conversation (thinking-model reasoning trimmed for readability):

```
🧯 content-moderator using qwen3.8:27b

User -> Tell me something unsafe.
⚠️  [GUARDRAIL] Blocked word "unsafe" detected in the request — refusing before calling the model
Agent -> I'm sorry, but I can't help with that request.

User -> What is the capital of Italy?
Agent -> The capital of Italy is **Rome**.

User -> What is the capital of Italy?
💾 [CACHE] Hit #1 for cache:439ed6573... — skipping the model call
Agent -> The capital of Italy is **Rome**.

User -> What is the capital of France?
Agent -> The capital of France is **Paris**.
```

Four turns, three distinct behaviors: a refusal (never reaching the model), a real answer, an exact cache hit on the repeated question, and a fresh real answer for the genuinely different question — the cache correctly did *not* fire for it.

### 5. Read `internal/agents/contentmoderator/callbacks_test.go`

Pure unit tests, no LLM — one or more per callback, including two regression tests worth special attention: `TestBeforeModelCallback_IgnoresBlockedWordFromEarlierHistory` (proves the history-scope fix) and `TestAfterModelCallback_DoesNotRedactEmailInThoughtPart` (proves the `Thought`-skipping in `afterModelCallback`, mirroring module-25.5's own regression-test shape exactly).

### 6. Read `internal/agents/contentmoderator/agent_test.go`

`TestContentModerator_{Ollama,Gemini}` drives the exact four-turn conversation above through a real `runner.New`, asserting all three behaviors structurally: the refusal's exact text, `cache.hitCount` staying at 0 after the first clean question, becoming 1 after the repeated one, and staying at 1 (not incrementing again) after a genuinely different question.

## Self-Reflection Questions 🤔
- Why does `afterAgentCallback` read `ctx.State().Get(outputKey)` instead of walking session events the way Python's own lab does? What would happen if you tried calling `ctx.Session()` from inside it?
- `beforeModelCallback` only inspects the *last* entry in `llmRequest.Contents`. What would you need to change if you wanted the guardrail to also catch a blocked word reused verbatim from much earlier in the conversation, without reintroducing the original bug?
- Why does `buildRootAgent` take an explicit `*responseCache` parameter instead of constructing one internally the way `BuildRootAgent` does?
- When would you reach for a Plugin (modules 25/25.5) instead of a callback for a guardrail like this one?

<hr/>

### Looking for the solution? 🔍

Hint: read `internal/agents/contentmoderator/callbacks.go` and `agent.go` for the real mechanism — six functions (one a method pair on a small struct), wired directly into `llmagent.Config`'s own six callback slots.
