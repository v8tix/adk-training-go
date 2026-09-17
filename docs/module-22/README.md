# Module 22: State and Memory — Persistent Agent Context (Go) 🧠📚

## Theory

### One Store, Four Scopes

You met `agent.Context.State()` back in Module 10 to remember a single name. This module's about using it deliberately — because not every piece of data should live the same amount of time or be visible to the same audience. The SDK draws that line with a key prefix, and there are exactly four scopes:

| Prefix | Constant | Lives for... | Visible to... |
|---|---|---|---|
| *(none)* | — | This one session | Just this conversation |
| `user:` | `session.KeyPrefixUser` | Every session this user ever has | This user, across all their sessions |
| `app:` | `session.KeyPrefixApp` | Forever, until changed | Every user, every session |
| `temp:` | `session.KeyPrefixTemp` | This one invocation only | Nobody past the current turn |

Reading or writing any of them is the exact same call — `ctx.State().Get(key)`/`ctx.State().Set(key, value)` — the prefix is just text baked into the key string itself. There's no separate API per scope. 🎯

### One Context, Not Two

Python's docs draw a line between `ToolContext` (for tools) and `CallbackContext` (for callbacks). Go doesn't need that distinction — `agent.Context.State()` is the *one* accessor every custom function tool receives, established all the way back in Module 9. Reading straight from the SDK's own `callbackContextState.Set` confirms it does two things at once on every write: it updates the event's `EventActions.StateDelta` *and* the live session state directly — the same "always go through the context, never poke a fetched session by hand" discipline Python's docs insist on, just collapsed into one type instead of two.

### `temp:` Really Does Disappear

This one's worth being skeptical about, so this module's own test doesn't just trust the doc comment — it proves it. A tool sets `temp:percentage` mid-turn; later in that *same* turn, it's readable. But once that turn's `Run()` call finishes and a brand-new, separate `Run()` call starts, `session.State().Get("temp:percentage")` comes back `session.ErrStateKeyNotExist` — gone, structurally confirmed, not assumed. `user:` keys set in that same turn, meanwhile, are still sitting there waiting. That contrast — one prefix survives, one doesn't — is the whole scope system in one experiment.

### Optional Placeholders in Instructions: `{key?}`

An agent's instruction string can pull live state straight into the prompt with `{key}` — but what if that key was never set? A plain `{app:course_version?}` (note the trailing `?`) tells the SDK "inject it if it's there, skip it silently if it's not," instead of erroring on a deployment that never got around to seeding that `app:` key. Confirmed straight from `internal/llminternal/instruction_processor.go`: `InjectSessionState` checks for that trailing `?` explicitly and treats a missing key as "leave it blank," not "fail."

### Long-Term, Searchable Recall: `memory.Service`

Session state answers "what does *this* conversation know." A `memory.Service` answers a different question: "has *any past* conversation with this user touched on this topic?" Two methods do the whole job — `AddSessionToMemory(ctx, session)` ingests a finished session, `SearchMemory(ctx, *SearchRequest)` returns matching entries later, ranked by how many distinct query words they hit. `memory.InMemoryService()` is the real, working, in-process implementation this module's own tests exercise directly — no mocking, no LLM call needed to prove it works.

### Key Takeaways ✅
- Four state scopes, one API — the prefix on the key string is the only thing that changes.
- `agent.Context.State()` unifies what Python splits into `ToolContext`/`CallbackContext`, and every `Set` writes both the event's delta and live state at once.
- `temp:` state is genuinely gone by the next separate `Run()` call — proven with a direct session-service read, not inferred from a docstring.
- `{key?}` lets an instruction reference state that might not exist yet, without erroring.
- `memory.Service`/`InMemoryService` gives real cross-session recall — `AddSessionToMemory` then `SearchMemory`, both demonstrated live in this module's own tests.

<hr/>

> **Coming from Python?** 🐍 Python's own module covers the same four prefixes, the same `ToolContext`/`CallbackContext` split (unified here into one `agent.Context`), the same `{key?}` optional templating, and the same `MemoryService` shape (`add_session_to_memory`/`search_memory` → `AddSessionToMemory`/`SearchMemory`). The one thing worth flagging plainly: Python's `.get(key, default)` takes its fallback inline; Go expresses "not found" as the explicit `session.ErrStateKeyNotExist` sentinel your own code checks for, the same pattern Module 10 already established.
