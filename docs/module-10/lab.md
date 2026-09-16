# Lab 10: Building a "Memory" Agent with Stateful Tools (Go) 🧠

## Goal

Build an agent that remembers a user's name across turns, using `agent.Context.State()` to give one of its tools real, persistent memory.

## Lab Tasks

### 1. Read `internal/agents/memory/tools.go`

Two handlers: `storeName` writes to `ctx.State()` under the key `"user_name"`; `recallName` reads it back, returning `"Stranger"` when nothing's been stored yet (`errors.Is(err, session.ErrStateKeyNotExist)`). Notice `recallName`'s parameter type, `RecallNameArgs struct{}` — empty, not omitted, since `functiontool.New` needs a struct or map even for a tool that takes no data.

### 2. Read `internal/agents/memory/agent.go`

Same shape as `internal/agents/calculator/agent.go` — two `functiontool.New` calls, attached via `Tools`. No forced backend; this agent's happy on the local default, just like the calculator was.

### 3. Run it — console mode, entirely locally 🖥️

```bash
go run ./cmd/memory console
```

No `.env`, no API key needed. Real, confirmed output from this exact command (thinking-model reasoning trimmed for readability):

```
🧠 memory using qwen3.8:27b

User -> Hi, I'm Mario.
Agent -> Hi Mario! Great to meet you. How can I help you today?

User -> What is my name?
Agent -> Your name is Mario! Is there anything else I can help you with today?
```

Two separate turns, same session — the second answer proves `recall_name` genuinely read back what `store_name` wrote in the first turn, not that the model just remembered "Mario" from the raw chat text (the instruction's `# Constraints` section requires the tool be used either way — see `internal/agents/memory/agent_test.go` for the real, structural test of this). ✨

### 4. Read `internal/agents/memory/tools_test.go`

`TestStoreName` and `TestRecallName` — pure unit tests, no LLM. They use a small hand-written `fakeState` plus `agent.StrictContextMock` (the SDK's own test double for `agent.Context`) instead of spinning up a real agent — the fast, deterministic base of this module's test pyramid.

### 5. Read `internal/agents/memory/agent_test.go`

`TestMemory_RemembersNameAcrossTurns_Ollama` and `_Gemini` — both call the agent **twice**, via two separate `Run()` calls against the same session ID, and check the second call's answer mentions "Mario". This is deliberately *not* one `Run()` call with two messages — the whole point is proving state survives *across* `Run()` boundaries, which is exactly this module's big idea.

## Self-Reflection Questions 🤔
- Why is it more reliable to stash data in `agent.Context.State()` than just leaning on the LLM's own chat history?
- What would happen if two different sessions both stored a name under the key `"user_name"`? (Hint: sessions are isolated automatically — check `runner.NewInMemory`'s session-ID parameter in `agent_test.go`.)
- How would you extend this agent to remember something else, like a favorite color? What would you need to add, and where?
- `agent.StrictContextMock` panics on any method you don't override. Why's that a better default for a test double than quietly returning a zero value?

<hr/>

### Looking for the solution? 🔍

Hint: read `internal/agents/memory/tools.go` and `agent.go` for the real mechanism — two tool functions, both touching `ctx.State()`, wrapped the same way `internal/agents/calculator`'s tools were.
