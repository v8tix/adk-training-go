# Lab 13.5: Extending ADK with Custom Redis Persistence (Go)

## Goal

Build a custom `session.Service` backed by Redis, inject it into a real `runner.Runner`, and prove it survives a process restart — the actual point of the lesson.

### Prerequisites

A reachable Redis (default `localhost:6379`, override with `REDIS_ADDR`) and Docker (for the integration tests, which use Testcontainers). No cloud account needed.

### Step 1: The Concrete Types

`internal/infrastructure/redissession/session.go` defines `redisSession`, `redisState`, and `redisEvents` — plain, in-memory views satisfying `session.Session`, `session.State`, and `session.Events`. There's no exported concrete type in the SDK to embed; every custom `session.Service` builds its own.

### Step 2: The Service Implementation

Read `internal/infrastructure/redissession/service.go`'s `Create`, `Get`, `List`, `Delete`, and `AppendEvent`. Pay particular attention to `extractStateDeltas` and `mergeStates` — the three-tier `app:`/`user:`/session state split this module's README explains, reimplemented from scratch since Go's `session.Service` interface has no default logic to inherit.

### Step 3: Verify Against the SDK's Own Conformance Suite

```bash
go test ./internal/infrastructure/redissession/... -v
```

Real, confirmed output from this exact command (abbreviated — the full run also logs Testcontainers' own container-startup lines, every `Create`/`Get`/`AppendEvent` subtest individually, and three more top-level tests covering `session.State`/`session.Events` directly):

```
=== RUN   TestRedisSessionService
[...Testcontainers startup logging...]
=== RUN   TestRedisSessionService/Create
=== RUN   TestRedisSessionService/Create/full_key
[...]
=== RUN   TestRedisSessionService/StateManagement/app_state_is_shared
=== RUN   TestRedisSessionService/StateManagement/user_state_is_user_specific
=== RUN   TestRedisSessionService/StateManagement/temp_state_is_not_persisted
--- PASS: TestRedisSessionService (0.60s)
=== RUN   TestAppendEvent_TempStateVisibleWithinSameInvocation
--- PASS: TestAppendEvent_TempStateVisibleWithinSameInvocation (0.37s)
[...]
PASS
ok      github.com/v8tix/adk-training-go/internal/infrastructure/redissession 1.46s
```

`sessiontestsuite.RunServiceTests` starts a real Redis container via Testcontainers for the whole run and flushes it between subtests — no fixture files, no mocked Redis client.

### Step 4: Prove Persistence Survives a Process Restart

`internal/agents/persistentagent` defines a minimal, tool-free agent — this lesson is about storage, not agent logic. `cmd/persistent-agent` injects the Redis-backed service into a real `runner.Runner`:

```go
sessionService := redissession.NewService(redisClient)
r, err := runner.New(runner.Config{
    AppName:        "extensibility_demo",
    Agent:          rootAgent,
    SessionService: sessionService,
    AutoCreateSession: true,
})
```

Run it once to set the fact, in one process:

```bash
go run ./cmd/persistent-agent set
```

**Real captured output:**

```
🔥 persistent-agent using Redis at localhost:6379
Got it! I'll remember that your favorite color is blue. 💙 Let me know if there's anything I can help you with today!
```

Stop it completely, then run it *again* — a genuinely separate process — asking about it:

```bash
go run ./cmd/persistent-agent ask
```

**Real captured output:**

```
🔥 persistent-agent using Redis at localhost:6379
Your favorite color is **blue**! 💙
```

The second process never shared memory with the first — only the same Redis. That's the proof.

### Having Trouble?

- **`ask` doesn't remember what `set` stored:** confirm both invocations point at the same `REDIS_ADDR` and the same `appName`/`userID`/`sessionID` (`cmd/persistent-agent/main.go` hardcodes them as constants for this lab — a real application would derive them per-user).
- **Tests fail to start a container:** confirm Docker is running (`docker info`). The tests skip, rather than fail, if Testcontainers genuinely can't reach Docker at all — a build or connection error inside a running Docker is a real problem to investigate.

### Lab Summary

You implemented a custom `session.Service` from the ground up — including its three-tier `app:`/`user:`/session state-scoping logic — verified it against the SDK's own official conformance suite, and proved persistence survives a real process restart.

### Self-Reflection Questions
- `session.Service` has no default method bodies to fall back on. What would you have to get right yourself if you were implementing a fourth scope — say, a `global:` prefix shared across every app?
- `sessiontestsuite.RunServiceTests` caught a real bug (state scoping) that a hand-written, ad hoc test might have missed entirely. What made that possible?
- If you wanted to add a second backend (say, Postgres) alongside Redis, what would you need to change in `cmd/persistent-agent/main.go`?

<hr/>

> **Coming from Python?** Python's lab writes `firestore_provider.py` against real GCP Firestore and manually calls `super().append_event(...)` to get default state-delta handling. This lab writes the Go equivalent against Redis, with no `super()` to call — the state-scoping logic (`extractStateDeltas`/`mergeStates` in `service.go`) is reimplemented in full, then verified by the SDK's own conformance suite rather than a manual "run it and see" check.
