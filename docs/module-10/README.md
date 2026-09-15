# Module 10: Giving Agents Memory with Stateful Tools (Go)

## Theory

### Why Tools Need Memory

Every tool your agent has used so far has been stateless — call it with some inputs, get an output, forget it ever happened. That's fine for a calculator, but plenty of real work needs a tool that remembers something between turns: a user's name, a preference, a running total from earlier in the conversation. This module is about giving a tool exactly that kind of memory.

### Every Tool Already Has a Way In: `agent.Context`

Every custom function tool you write takes `agent.Context` as its first parameter — you've been passing it along since Module 9 without using it for much. It's your tool's doorway into the running session: who's talking, what's already happened, and — the part this module is about — a small persistent store called session state.

```go
func storeName(ctx agent.Context, args StoreNameArgs) (StoreResult, error) {
    if err := ctx.State().Set("user_name", args.Name); err != nil {
        return StoreResult{}, err
    }
    return StoreResult{Status: "success"}, nil
}
```

`ctx.State()` gives you exactly two operations that matter here: `Set(key, value)` to remember something, and `Get(key)` to read it back later. That's the whole vocabulary.

### Reading Back What You Stored

A `Get` on a key that's never been set doesn't panic or silently return a zero value — it returns a specific, named error, `session.ErrStateKeyNotExist`, so your tool can tell "nothing here yet" apart from "something actually broke." That makes a sensible default straightforward to write:

```go
func recallName(ctx agent.Context, _ RecallNameArgs) (RecallResult, error) {
    val, err := ctx.State().Get("user_name")
    if errors.Is(err, session.ErrStateKeyNotExist) {
        return RecallResult{Name: "Stranger"}, nil
    }
    if err != nil {
        return RecallResult{}, err
    }
    name, _ := val.(string)
    return RecallResult{Name: name}, nil
}
```

### A Tool That Takes Nothing

`recall_name` doesn't need any input from the model at all — it just looks something up. Every tool still needs a typed arguments struct, though, so the honest way to say "no arguments" is an empty one:

```go
type RecallNameArgs struct{}
```

Confirmed live: the model calls this correctly with no arguments at all, exactly as you'd hope.

### The Memory Actually Survives Between Turns

Here's the part worth being skeptical about until you've seen it work: does state set during one exchange actually still exist several messages later? Confirmed with a real run against the local model: one turn ("Hi, I'm Mario.") calls `store_name`; a completely separate, later turn in the *same conversation* ("What is my name?") calls `recall_name` and gets "Mario" back. This isn't the model just remembering the earlier text — it's real, persisted state, tied to the session, and this module's test proves that directly by checking the tool's own result, not just the model's final reply.

### Testing Memory Without Running a Model

You don't need a real LLM call to test `storeName`/`recallName` — you need something that behaves like `agent.Context` well enough to exercise the logic. The SDK ships exactly that: `agent.StrictContextMock`, a small test double you embed and override only the one method you actually need. Anything you forgot to override panics loudly instead of quietly returning a fake zero value that could hide a real bug:

```go
type fakeContext struct {
    agent.StrictContextMock
    state *fakeState // a tiny map-backed session.State
}

func (c *fakeContext) State() session.State { return c.state }
```

### Key Takeaways
- `agent.Context.State()` gives a tool a small persistent key-value store, scoped to the current session.
- `Get` returns the named error `session.ErrStateKeyNotExist` when a key is missing, so a tool can build its own default behavior around that specific case.
- A tool with no arguments still needs a typed struct — an empty one, since there's no other way to say "no parameters."
- State genuinely persists across separate turns in the same session — confirmed live, not assumed.
- `agent.StrictContextMock` lets you unit-test a stateful tool without ever calling a real model.

<hr/>

> **Coming from Python?** This module's `agent.Context` plays the same role as Python's `ToolContext`, with one structural difference: Python's version is opt-in — you add `tool_context: ToolContext` as an extra parameter only when a tool needs it, and the model never sees it in the schema. In Go, every custom function tool already receives `agent.Context` as its first parameter, whether it uses it or not — there's no separate opt-in step. `ctx.State().Get`/`Set` map onto `tool_context.state`'s dict-like `get`/item-assignment; the one real difference is that Python's `.get(key, default)` takes a fallback value inline, where Go expresses "not found" as the `session.ErrStateKeyNotExist` sentinel your own code checks for explicitly.
