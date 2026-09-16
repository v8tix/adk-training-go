# Lab 16: Building a Hybrid News Aggregator (Go)

## Goal

Build the two-agent design from module-15's own paper exercise for real — plus a third agent and a genuine hybrid graph: two research agents run in parallel, converge at a `JoinNode`, then a summarizer runs sequentially.

### The Architecture

```
        ┌──── tech_researcher ────┐
START ──┼                         ├──→ JoinNode → summarizer → END
        └──── market_researcher ──┘
```

### Step 1: The Three Agents

`internal/agents/newsaggregator/agent.go`'s `BuildRootAgent` builds three plain `llmagent`s — no tools, no built-in-tool capability, matching Python's own lab (neither researcher is given a real search tool here; this lesson is about the orchestration topology, not live retrieval):

```go
techResearcher, _ := llmagent.New(llmagent.Config{
    Name:        "tech_researcher",
    Instruction: techInstruction, // "Find 3 recent headlines about AI and robotics..."
    OutputKey:   "tech_news",
})
marketResearcher, _ := llmagent.New(llmagent.Config{
    Name:        "market_researcher",
    Instruction: marketInstruction,
    OutputKey:   "market_news",
})
summarizer, _ := llmagent.New(llmagent.Config{
    Name:        "summarizer",
    Instruction: summarizerInstruction, // references {tech_news} and {market_news}
})
```

### Step 2: Assemble the Hybrid Graph

```go
techNode, _ := workflow.NewAgentNode(techResearcher, workflow.NodeConfig{})
marketNode, _ := workflow.NewAgentNode(marketResearcher, workflow.NodeConfig{})
summarizerNode, _ := workflow.NewAgentNode(summarizer, workflow.NodeConfig{})
syncer := workflow.NewJoinNode("news_sync")

edges := []workflow.Edge{
    {From: workflow.Start, To: techNode},      // parallel branch 1
    {From: workflow.Start, To: marketNode},    // parallel branch 2
    {From: techNode, To: syncer},              // converge...
    {From: marketNode, To: syncer},            // ...both here
    {From: syncer, To: summarizerNode},        // sequential final step
}

rootAgent, _ := workflowagent.New(workflowagent.Config{Name: "NewsSystem", Edges: edges})
```

Five edges for a three-node hybrid graph — two for the fan-out, two for the fan-in, one for the sequential step, written as explicit `Edge{}` literals so each connection is visible at a glance. This module's README shows the real builder alternatives (`workflow.Chain`, `EdgeBuilder.AddFanOut`/`AddFanIn`) for the same shapes.

### Step 3: Run and Verify

```bash
go run ./cmd/news-aggregator console
```

Real, confirmed output from this exact command (Gemini backend — see the note below on why the local model behaves differently):

```
📰 news-aggregator using gemini-3.5-flash

User -> Give me today's update.
Agent -> 1. Toyota Partners with Boston Dynamics to Bring Advanced Artificial
Intelligence to the Atlas Humanoid Robot
2. Physical Intelligence Secures $400 Million in Funding to Develop a
Universal "Brain" Software for Diverse Robots
3. Nvidia Launches Project GR00T, a New AI Foundation Model Designed
Specifically for Humanoid Robot Development

[... market_researcher's own three headlines follow in the same event stream ...]

**Subject: The Weekly Buzz: Smart Robots, Market Highs, and More! 🤖📈**

Hey there, friends!
...
### 🤖 Tech Trends: The Rise of the Robots
* **Toyota & Boston Dynamics Team Up:** ...
### 💼 Market Watch: All-Time Highs & Shifting Trends
* **Record-Breaking Runs:** ...
```

Notice the summarizer's newsletter genuinely weaves in facts from *both* researchers — proof the fan-out/fan-in actually worked, not just that the graph didn't error.

### A Real, Confirmed Difference Between Backends

Since neither researcher has a real search tool, each backend fills the gap differently — confirmed live, not assumed. The local Ollama model, asked to "find headlines" with no way to actually search, honestly declined and offered alternative sources instead of inventing any. Gemini instead answered confidently from its own training data, producing plausible-sounding (but not actually live) headlines. Neither is wrong — this lab is about proving the graph's topology works, not about sourcing real news; module-12's `google_search` (Gemini-only) is the tool you'd reach for if live retrieval were the actual goal, but it can't share a `tools` list with these plain agents' custom-tool-shaped setup without the same `IncludeServerSideToolInvocations` flag from that module.

### Having Trouble?

- **The local model refuses to invent headlines:** that's a real, honest model behavior, not a bug — see the note above. Switch to `MODEL_TYPE=gemini` if you want a more filled-in example.
- **The summarizer's instruction shows literal `{tech_news}` instead of real content:** confirm the two `OutputKey`s exactly match the placeholder names in `summarizer_instruction.md` — they're matched by exact string, not fuzzy.

### Lab Summary

You built a real hybrid graph: fan-out with plain parallel edges, fan-in with `JoinNode`, and a sequential final step — all wired with `workflow.Edge`, and proven end-to-end with a test that checks both `OutputKey`s actually populated, not just that the graph ran without error.

### Self-Reflection Questions
- Why does `JoinNode`'s own doc comment call conditional routing into it a configuration error? What would happen if one of the two research branches used a `Route` that sometimes skipped it?
- The summarizer's instruction never directly touches the `JoinNode`'s own aggregated output. What does the `JoinNode` actually guarantee, if not the data itself?
- How would you add a third parallel branch (say, a `sports_researcher`)? What exactly would you need to add to the edges slice?

<hr/>

> **Coming from Python?** Python's lab tip notes a 3-element tuple `(A, B, C)` is shorthand for two edges. Go's direct equivalent is `workflow.Chain(A, B, C)` — this lab writes five explicit `Edge{From, To}` values instead, purely for clarity while learning the model, not because Go lacks the shorthand.
