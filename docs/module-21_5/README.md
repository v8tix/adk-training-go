# Module 21.5: MAS Knowledge Milestone — Architecture Choice (Go) 🏆

## Theory

You've now built six real, working multi-agent systems in this repo, each solving orchestration in a genuinely different way. Nice! Choosing the right geometry for a new graph — *before* writing any code — is the most important decision this whole section of the course teaches.

### The Six Patterns, Recapped 📋

| Pattern | Module | Go Package (shipped, this repo) | Key Go Primitives |
|---|---|---|---|
| Static Orchestration | [16](../module-16/README.md) | `internal/agents/newsaggregator` | `workflow.Edge`, `workflow.NewJoinNode` |
| Structured Routing | [17](../module-17/README.md) | `internal/agents/marketrouter` | `workflow.NewFunctionNode` returning `*session.Event` with `.Routes` set, `workflow.EdgeBuilder.AddRoutes` |
| Dynamic Orchestration | [18](../module-18/README.md) | `internal/agents/supportrouter` | `workflow.NewDynamicNode`, `workflow.RunNode` |
| Collaborative Teams | [19](../module-19/README.md) | `internal/agents/travelplanner` | `llmagent.Config.Mode` (`ModeTask`/`ModeSingleTurn`) |
| Cyclic Workflows | [20](../module-20/README.md) | `internal/agents/essayrefiner` | a plain Go `for` loop inside `workflow.NewDynamicNode`, calling `workflow.RunNode` repeatedly |
| Distributed Graphs | [21](../module-21/README.md) | `internal/agents/researchspecialist`, `internal/agents/a2aorchestrator` | `cmd/launcher/web/a2a`, `agent/remoteagent/v2.NewA2A` |

Every link goes straight to that module's own README — this milestone's job is the recap and the decision framework, not a second copy of six modules' worth of Theory.

### How to Choose, in Go's Own Terms 🧭

Three questions, translated from the abstract into what they actually mean once you're holding real Go code:

1. **Is the path predictable?** If every possible branch can be declared up front as a `workflow.Edge` — sequential, parallel, or `Route`-tagged — reach for **Static** (16) or **Structured Routing** (17). These are the cheapest to reason about: the edge list *is* the whole graph.
2. **Does the logic need ordinary Go control flow — a `for` loop, an `if`/`else` chain, a `try`/`recover`-style branch — that doesn't reduce to a fixed set of edges?** Use a **Dynamic** node (18) — a `workflow.NewDynamicNode` whose body is just Go code calling `workflow.RunNode`. If that logic needs to *repeat* until some model-driven condition is met, that's specifically the **Cyclic** shape (20) — the same `NewDynamicNode`/`RunNode` mechanism, with a loop and a hard iteration cap.
3. **Do the agents need to live in genuinely separate environments** — different processes, different machines, different teams' codebases? That's **Distributed** (21) — `agent/remoteagent/v2.NewA2A` over real A2A HTTP, not a local `SubAgents` entry.

So where does **Collaborative Teams** (19) fit? It's the odd one out on purpose 😄 — it's not about graph *shape* at all, but about how much control you cede to the sub-agent itself. `Mode: ModeSingleTurn`/`ModeTask` gives you LLM-driven delegation with a *guaranteed* return, without writing any orchestration code — the right call when a specialist's own judgment about when it's "done" is good enough, and you'd rather not hand-write the loop or the routing edge yourself.

### Details Worth Keeping in Mind 🔍

Two of this repo's own six modules confirmed something genuinely worth remembering, way beyond just "how to build the pattern":

- **Module 19** confirmed `llmagent.Config` needs *no* resumability configuration at all for task-mode's multi-turn pause/resume — the framework sets the workflow-level equivalent automatically for every `LlmAgent` node.
- **Module 20** confirmed a dynamic node's own loop result lives on a distinct terminal event authored by the *root* workflow agent's name, with `Content` nil and `Output` set — not on whichever chat-content event happens to come last.

Both are exactly the kind of detail that decides whether your own test (or your own trace-inspection code) actually proves what you think it proves. Module-21's own review caught precisely this category of mistake in a *test*, not just documentation: an assertion that checked only an event's author, which turned out to say nothing about whether the call had actually succeeded.

### Key Takeaways ✅
- There's no one-size-fits-all architecture — most of this repo's own later, larger builds mix these patterns together (think a static fan-out feeding a dynamic loop).
- Prefer the simplest geometry that solves the problem: don't reach for a `NewDynamicNode` if a fixed `workflow.Edge` list already says everything you need.
- Every pattern this milestone recaps has a real, live-tested Go package sitting right in this repo — when in doubt about a mechanism's exact behavior, that package's own tests and README are the ground truth, not this recap table.

<hr/>

> **Coming from Python?** 🐍 Python's own milestone recaps `Workflow`/`@node`/`ctx.run_node()`/`sub_agents`/`RemoteA2aAgent` by name. This Go mirror recaps the same six patterns by pointing at real, shipped Go packages instead — each one already independently verified live during its own module, including at least two places (modules 19 and 20) where the Go mechanism turned out to behave in a genuinely better or more surprising way than Python's own material describes, not just a renamed equivalent.
