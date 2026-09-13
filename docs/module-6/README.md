# Module 6: Programmatic Execution: Apps and Runners (Go)

## Theory

### Python's "Three Pillars," Go's Two

Python's course names three objects: `Agent` (intelligence), `App` (infrastructure — plugins, caching), `Runner` (the execution engine). Go only has two, confirmed by reading `runner.Config`'s actual field list, not inferred:

```go
type Config struct {
    AppName string
    Agent   agent.Agent
    SessionService session.Service
    ArtifactService artifact.Service // optional
    MemoryService   memory.Service   // optional
    PluginConfig    PluginConfig     // optional — Python's App-level plugins
    Compaction      *compaction.Config // optional — Python's App-level context caching
}
```

There's no separate `google.golang.org/adk/v2/app` package to import. What Python's `App` object holds *separately* from its `Runner` — plugins, context-caching/compaction settings — are just fields on Go's `runner.Config`. `runner.New(cfg)` already *is* "wrap the agent in an App, then build a Runner" collapsed into one step. This is a real simplification in Go's design, not a missing feature — worth knowing so you don't go looking for an `App` type that was never there.

### No `run_debug()` — and What Replaces It

Python's `runner.run_debug(message, user_id=...)` is a convenience wrapper: it drives the async event stream for you and hands back a plain list of events. Go's `Runner` has exactly two execution methods, confirmed via `go doc`:

```go
func (r *Runner) Run(ctx context.Context, userID, sessionID string, msg *genai.Content, cfg agent.RunConfig, opts ...RunOption) iter.Seq2[*session.Event, error]
func (r *Runner) RunLive(...) (...)
```

No synchronous "just give me the events" wrapper exists. This isn't new to this module — every `cmd/` program in this repo has already been writing the same small "range over `Run`'s iterator, find the event where `IsFinalResponse()` is true" idiom since module-3 (`runEcho`, `analyzeTicket`). `cmd/support-analyzer-runner`'s `runOnce` is the same pattern, just named for what this module teaches.

### One Runner, Two Isolated Users — the Actual Point of This Module

```go
r, _ := runner.NewInMemory("support_analyzer_runner_app", rootAgent)

aliceResult, _ := runOnce(ctx, r, "alice", "alice_session", "I was overcharged $50")
bobResult, _ := runOnce(ctx, r, "bob", "bob_session", "My wifi is slow")
```

Same `userID`/`sessionID` isolation mechanism this repo's tests have used since module-3 — the new thing here is deliberately building *one* `*runner.Runner` and driving *two different users* through it in the same process, the shape a real backend (a web server handling concurrent requests) actually needs. Confirmed live: Alice's billing complaint and Bob's technical issue each get their own correct, independent analysis from the one shared runner — no state leaks between them.

### The Structural Answer to Python's `agent.py` / `main.py` Split

Python's lab adds a second file, `main.py`, to the *same* `support_analyzer` project directory, importing `root_agent` from the existing `agent.py`. Go can't do this — two `func main()`s can't live in one package — so this module extracts the Support Analyzer's definition into a new, importable package: `internal/agents/supportanalyzer`. Both `cmd/support-analyzer` (the CLI/launcher entrypoint, unchanged in behavior since module-5) and the new `cmd/support-analyzer-runner` (this module's actual deliverable) import it. This is the same separation Python's file layout already expressed — "the agent" vs. "how you invoke it" — just enforced by a Go package boundary instead of a naming convention within one directory. It's also the first `internal/agents/` package in this repo, a new category alongside the existing `internal/infrastructure/` one, for domain/agent-definition logic rather than external-system adapters.

### Key Takeaways
- Go has no separate `App` type — `runner.Config` already carries what Python's `App` holds (plugins, compaction) directly, confirmed by its own field list.
- There's no `run_debug()` equivalent; the "range over the iterator, find the final response" idiom this repo has used since module-3 is the honest Go substitute.
- One `*runner.Runner`, many users, isolated by `userID`/`sessionID` — proven live with two genuinely different, correct results from one shared instance.
- `internal/agents/` is a new package category: where an agent's own definition lives once more than one program needs to run it, mirroring Python's own `agent.py`/`main.py` file separation as a Go package boundary.
