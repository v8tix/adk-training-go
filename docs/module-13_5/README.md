# Module 13.5: Extending ADK — Custom Persistence with Redis (Go) 🗄️

*This module sits alongside module-13, not after the state/memory modules, on purpose: writing a custom `session.Service` is itself an act of extending the ADK's toolkit, not a production-operations concern. Think of it as "building a new kind of tool" — just one that stores state instead of taking action.*

## Theory

### `session.Service` Is a Pluggable Interface 🔌

Every prior module used `session.InMemoryService()` without a second thought. Turns out it's just one implementation of a plain Go interface:

```go
type Service interface {
    Create(context.Context, *CreateRequest) (*CreateResponse, error)
    Get(context.Context, *GetRequest) (*GetResponse, error)
    List(context.Context, *ListRequest) (*ListResponse, error)
    Delete(context.Context, *DeleteRequest) error
    AppendEvent(context.Context, Session, *Event) error
}
```

`runner.Runner` doesn't care which implementation you hand it — swap in your own via `runner.Config`:

```go
sessionService := redissession.NewService(redisClient)
r, err := runner.New(runner.Config{
    AppName:        "extensibility_demo",
    Agent:          rootAgent,
    SessionService: sessionService, // instead of session.InMemoryService()
})
```

From here on, every `r.Run(...)` call persists through your storage — no changes needed anywhere in your agent instructions or tool code. 🎉

### The Contract Has No Default Implementation ⚠️

`session.Service` is a plain Go interface — implementing it means writing every method's full behavior yourself, including how a `StateDelta` gets applied to state. Confirmed by reading the SDK's own `InMemoryService` directly: that application isn't a flat copy. A `StateDelta` key gets scoped three ways:

- `app:key` — shared across **every session for the app**, regardless of user
- `user:key` — shared across every session for **one user**, within the app
- any other key — private to that one session
- `temp:key` — applied to the session's state for the rest of the *current* invocation, but **never written to durable storage** — a fresh `Get` afterward never sees it

A custom `session.Service` has to implement this scoping itself — there's no shared default to fall back on. `internal/infrastructure/redissession`'s `AppendEvent` does exactly that:

```go
appDelta, userDelta, sessionDelta := extractStateDeltas(event.Actions.StateDelta)
mergeHash(ctx, appStateKey(appName), appDelta)           // shared app-wide
mergeHash(ctx, userStateKey(appName, userID), userDelta) // shared per-user
maps.Copy(stored.State, sessionDelta)                     // private to this session
```

### Proving Correctness Against the SDK's Own Test Suite ✅

Python's lab verifies its Firestore provider by running it once and eyeballing the result. Go's got something way stronger: `session/sessiontestsuite`, a real, backend-agnostic conformance suite the SDK ships and uses to test its *own* built-in SQL-backed service:

```go
sessiontestsuite.RunServiceTests(t, sessiontestsuite.SuiteOptions{
    SupportsUserProvidedSessionID: true,
}, func(t *testing.T) session.Service {
    client.FlushDB(ctx)
    return redissession.NewService(client)
})
```

Here's the receipts: this is what actually caught the three-tier state scoping above, live, while building this module — an earlier version treated all state as session-private, and `RunServiceTests` failed two subtests (`app_state_is_shared`, `user_state_is_user_specific`) with a precise, actionable diff. Not a guess. Not a vibe. A real diff pointing right at the bug. 🎯

### A Real Backend, Verified with Testcontainers 🐳

Python's lab requires a real Google Cloud project and `gcloud auth application-default login`. This course has never needed a full cloud account before, so this module swaps in **Redis** instead — named explicitly in Python's own Theory section as a valid alternative for exactly this use case ("near-instant response times a standard persistent database might not provide"). `internal/infrastructure/redissession`'s tests use [Testcontainers](https://testcontainers.com/) to spin up a real, ephemeral Redis container — no manually-managed test database, no mocks standing in for the real thing, and nothing left running once the test suite finishes.

### Key Takeaways ✅
- `session.Service` is a plain interface, injected into `runner.Config.SessionService` — any implementation works, zero changes needed to agent or tool code.
- Go's interface has no default method bodies to inherit — a custom implementation owns the full contract, including the three-tier `app:`/`user:`/session state split.
- `session/sessiontestsuite.RunServiceTests` verifies a custom implementation against the same suite the SDK uses on its own built-in service — a much stronger correctness story than manual spot-checking.
- Testcontainers provisions a real, ephemeral backend for integration tests — the first module in this course that needs it.

<hr/>

> **Coming from Python?** 🐍 Python's `BaseSessionService` and Go's `session.Service` play the same role — a pluggable persistence interface injected at the `Runner`. The real difference is inheritance: Python's abstract base class hands a subclass real default logic via `super()`; Go's interface hands a custom implementation nothing but a method set to satisfy. Python's own lab also picks Firestore specifically — this module picks Redis instead, one of the alternatives Python's own Theory section names, to keep this course's no-cloud-account requirement intact.
