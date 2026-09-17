# Module 6: Programmatic Execution: Apps and Runners (Go) ⚙️

## Theory

### Agent and Runner: Everything Lives on One Config 📦

Building and driving an agent programmatically really comes down to two things: the agent itself, and a `*runner.Runner` to run it. Every setting you'd need — session storage, artifacts, memory, plugins, context-caching — is just a field on one struct, confirmed by reading `runner.Config`'s actual field list, not inferred:

```go
type Config struct {
    AppName string
    Agent   agent.Agent
    SessionService session.Service
    ArtifactService artifact.Service // optional
    MemoryService   memory.Service   // optional
    PluginConfig    PluginConfig     // optional
    Compaction      *compaction.Config // optional
}
```

`runner.New(cfg)` builds a fully-configured runner in one step — no separate infrastructure object to assemble first. Nice. 👍

### Driving the Event Stream Yourself 🌊

`Runner` has exactly two execution methods, confirmed via `go doc`:

```go
func (r *Runner) Run(ctx context.Context, userID, sessionID string, msg *genai.Content, cfg agent.RunConfig, opts ...RunOption) iter.Seq2[*session.Event, error]
func (r *Runner) RunLive(...) (...)
```

`Run` hands you an iterator over the agent's events — you range over it and pick out the one where `IsFinalResponse()` is true. This is the same small idiom every `cmd/` program in this repo has already been writing since module-3 (`runEcho`, `analyzeTicket`); `cmd/support-analyzer-runner`'s `runOnce` just names it explicitly, since naming the idiom is literally what this module teaches.

### One Runner, Two Isolated Users — the Actual Point of This Module 👥

```go
r, _ := runner.NewInMemory("support_analyzer_runner_app", rootAgent)

aliceResult, _ := runOnce(ctx, r, "alice", "alice_session", "I was overcharged $50")
bobResult, _ := runOnce(ctx, r, "bob", "bob_session", "My wifi is slow")
```

Same `userID`/`sessionID` isolation mechanism this repo's tests have used since module-3 — what's new here is deliberately building *one* `*runner.Runner` and driving *two different users* through it in the same process. That's exactly the shape a real backend (a web server handling concurrent requests) actually needs. Confirmed live: Alice's billing complaint and Bob's technical issue each get their own correct, independent analysis from the one shared runner — zero state leaks between them. ✨

### Sharing One Agent Between Two Programs 🤝

Once a second program needs the same agent definition, that definition needs its own importable package — a Go package can only have one `func main()`, so it can't live inside either `cmd/` program directly. This module pulls the Support Analyzer's definition out into `internal/agents/supportanalyzer`, and both `cmd/support-analyzer` (the CLI/launcher entrypoint, unchanged in behavior since module-5) and the new `cmd/support-analyzer-runner` (this module's actual deliverable) import it. This is the first `internal/agents/` package in this repo — a new category alongside `internal/infrastructure/`, for domain/agent-definition logic rather than external-system adapters: the natural home for "the agent itself" once more than one program needs to run it.

### Key Takeaways ✅
- `runner.Config` carries every infrastructure setting — session storage, artifacts, memory, plugins, compaction — directly, confirmed by its own field list. `runner.New(cfg)` builds a fully-configured runner in one step.
- `Run` returns an iterator, not a plain list of events — range over it and find the one where `IsFinalResponse()` is true, the idiom this repo has used since module-3.
- One `*runner.Runner`, many users, isolated by `userID`/`sessionID` — proven live with two genuinely different, correct results from one shared instance.
- `internal/agents/` is where an agent's own definition lives once more than one program needs to run it.

<hr/>

> **Coming from Python?** 🐍 Python names three objects — `Agent`, `App` (infrastructure: plugins, caching), and `Runner` (the execution engine). Go only has two: everything Python's `App` holds separately is just a field on Go's `runner.Config`, so `runner.New(cfg)` already *is* "wrap the agent, then build a runner" collapsed into one step — there's no `App` type to go looking for. Python's `runner.run_debug()` convenience wrapper has no Go equivalent either; you drive `Run`'s iterator yourself. And where Python's lab adds a second file (`main.py`) to the same project directory, Go expresses the same "agent vs. how you invoke it" split as a package boundary instead, since two `func main()`s can't share one package.
