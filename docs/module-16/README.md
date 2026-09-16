# Module 16: Static Orchestration — Linear and Parallel Edges (Go)

## Theory

### The Geometry of a Graph: `workflow.Edge`

A deterministic, code-defined orchestration is a graph of `workflow.Edge` values, each a plain struct connecting two nodes:

```go
type Edge struct {
    From  Node
    To    Node
    Route Route
}
```

`workflow.Start` is the exported sentinel node for the graph's entry point. A chain of edges runs sequentially — each node waits for its predecessor:

```go
edges := []workflow.Edge{
    {From: workflow.Start, To: researcherNode},
    {From: researcherNode, To: writerNode},
    {From: writerNode, To: editorNode},
}
```

### Wrapping an Agent as a Node

`Edge.From`/`Edge.To` need a `workflow.Node`, not an `agent.Agent` directly — `workflow.NewAgentNode(a agent.Agent, cfg NodeConfig) (*AgentNode, error)` is the adapter:

```go
researcherNode, err := workflow.NewAgentNode(researcherAgent, workflow.NodeConfig{})
```

Any plain `llmagent` works here — the graph doesn't care what kind of agent produced a node's output.

### Fan-Out: Parallel Edges

Multiple edges from the same source run concurrently — no separate "parallel" construct, just more edges sharing a `From`:

```go
edges := []workflow.Edge{
    {From: workflow.Start, To: techNode},    // starts immediately
    {From: workflow.Start, To: marketNode},  // starts at the same time
}
```

### Fan-In: `workflow.NewJoinNode`

`workflow.NewJoinNode(name string) *JoinNode` is a real, confirmed synchronization barrier. Its own doc comment states the contract precisely: it "is activated exactly once, after every predecessor declared by the graph edges has completed." A genuinely useful warning from that same doc comment, worth internalizing before you build one: routing only *some* of the time into a `JoinNode` is a configuration error — the barrier waits for every declared predecessor, and a route-skipped one simply never lets it fire.

```go
syncer := workflow.NewJoinNode("news_sync")

edges := []workflow.Edge{
    {From: workflow.Start, To: techNode},
    {From: workflow.Start, To: marketNode},
    {From: techNode, To: syncer},
    {From: marketNode, To: syncer},
    {From: syncer, To: summarizerNode},
}
```

### How Data Actually Flows: `OutputKey`, Not the Join's Own Output

`JoinNode`'s own `Run` method does emit an aggregated `map[string]any` (each predecessor's output, keyed by name) — but confirmed live this module, that's not what a downstream agent actually reads. The real mechanism is `llmagent.Config.OutputKey`, which writes an agent's final response into session state under a name:

```go
techResearcher, _ := llmagent.New(llmagent.Config{
    Name:        "tech_researcher",
    Instruction: techInstruction,
    OutputKey:   "tech_news",
})
```

...combined with real `{key}` interpolation in a later agent's own instruction — confirmed to work identically to Python's:

```
Combine the following into a short, friendly newsletter:

Tech news: {tech_news}

Market news: {market_news}
```

The `JoinNode` in the middle is what *guarantees* both keys are already populated by the time the summarizer's instruction gets resolved — its role is purely the synchronization barrier, not a data pipe.

### Building Edges: Literals or Real Builder Sugar

`workflow.Edge{}` is a plain struct — writing one out, as this module's own `agent.go` does for all five edges, states one connection at a time, plainly. But the package also ships real convenience builders for exactly the two shapes this lab needs, confirmed present in the pinned SDK:

```go
workflow.Chain(startNode, techNode, syncer) // → []Edge{{startNode, techNode}, {techNode, syncer}}
```

`workflow.Chain(nodes ...Node) []Edge` generates a chain's edges from a plain node list — the direct functional equivalent of Python's 3-element tuple shorthand `(A, B, C)`, just as a standalone function instead of special tuple syntax. `workflow.NewEdgeBuilder().AddFanOut(from, a, b).AddFanIn(to, a, b).Build()` covers the fan-out/fan-in shape this same lab builds. This module writes the five edges as explicit literals for teaching clarity — so each connection is visible at a glance while you're learning the model — not because no shorthand exists.

### Wrapping the Whole Graph as an Agent

`agent/workflowagent.New(workflowagent.Config{Name, Edges: edges})` wraps a `workflow.Workflow` as a plain `agent.Agent` — confirmed live, it runs through `runner.Run` and the standard launcher exactly like any other agent, no special wiring needed.

### Key Takeaways
- `workflow.Edge{From, To, Route}` is the graph's building block — a chain is sequential, multiple edges sharing a `From` fan out in parallel.
- `workflow.NewAgentNode` adapts any `agent.Agent` into a graph node.
- `workflow.NewJoinNode` is a real fan-in barrier — it fires exactly once, after every declared predecessor, never on a partial set.
- `OutputKey` + `{key}` instruction interpolation carries data between nodes — the `JoinNode`'s own aggregated output isn't what a downstream agent typically reads.
- `agent/workflowagent.New` wraps the whole graph as a plain `agent.Agent` — no special runner or launcher handling needed.
- `workflow.Chain` and `EdgeBuilder.AddFanOut`/`AddFanIn` are real builder functions for the two edge shapes this lab uses — explicit `Edge{}` literals are a teaching choice here, not the only option.

<hr/>

> **Coming from Python?** Python's `Workflow(edges=[(A, B, C)])` accepts a 3-element tuple as shorthand for a chain of two edges; Go's direct equivalent is the standalone function `workflow.Chain(A, B, C)`, called separately rather than embedded in tuple syntax. This lab writes explicit `Edge{}` values instead of reaching for `Chain`, purely for clarity while learning the model. Everything else — sequential chains, parallel fan-out, the `JoinNode` barrier, `output_key`/`{key}` interpolation — maps directly.
