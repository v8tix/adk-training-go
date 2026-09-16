# Lab 21.5: MAS Architecture Design Challenge (Go) 🏗️

## Goal

No code this time! You're stepping into the shoes of a Lead AI Architect: given three business scenarios, pick the right pattern from modules 16-21, sketch the graph geometry, and justify your call. 🎯

---

### Scenario 1: The Legal Review Pipeline ⚖️

A law firm needs a system to process incoming contracts.
- **Step A:** Extract all dates and names.
- **Step B:** Simultaneously check for "Privacy" violations AND "Liability" risks.
- **Step C:** If either check finds a "High" risk, send to a Senior Partner agent for final review.
- **Step D:** Otherwise, generate a "Safe to Sign" summary.

**Task:** Design the graph. Which Go primitive handles the "Simultaneous" part? Which handles the "High Risk" decision?

<details>
<summary>Think it through, then check the reasoning</summary>

This one's a **hybrid** of two patterns already built in this repo:

- Step B's "simultaneously" is a fan-out/fan-in — two edges sharing the same `From`, converging at a `workflow.NewJoinNode` before Step C can run. This is exactly `internal/agents/newsaggregator`'s own shape (Module 16): `{From: extractNode, To: privacyNode}`, `{From: extractNode, To: liabilityNode}`, both feeding a `syncer := workflow.NewJoinNode(...)`.
- Step C's "if either check is High risk" is a deterministic routing decision on data that's already been computed — no open-ended logic, no loop. That's Module 17's Structured Routing: a small `workflow.NewFunctionNode` reading both checks' results and returning a `*session.Event` with `.Routes` set to `"senior_partner"` or `"safe_to_sign"`, wired via `workflow.EdgeBuilder.AddRoutes`.

Neither Dynamic Orchestration (18) nor a loop (20) is needed here — the entire path, including the branch, is knowable before the graph ever runs. 👍
</details>

---

### Scenario 2: The Multi-Turn Story Writer ✍️

A creative agency wants an agent to write children's books.
- The agent must write a chapter, then send it to a "Critic" agent.
- If the Critic says "Too Scary," the agent must rewrite the chapter and send it back to the Critic.
- This continues until the Critic is satisfied.

**Task:** What type of graph geometry is this? Which module covered this pattern?

<details>
<summary>Think it through, then check the reasoning</summary>

This is **Module 20's Cyclic Workflow** — `internal/agents/essayrefiner`'s own exact shape. The number of rewrite iterations isn't knowable up front, so it can't be expressed as fixed edges; it needs a real Go `for` loop (capped at `maxIterations`, a required safety limit) inside a `workflow.NewDynamicNode`, calling `workflow.RunNode` on the critic and the refiner alternately, breaking early once the critic approves.

A genuinely non-obvious detail this repo's own Module 20 confirmed live: the loop's real final result doesn't show up as "whatever chat message came last" — it arrives on a distinct terminal event authored by the *root* workflow agent's own name. If you were building automated tests or trace tooling for this exact scenario, that's the event you'd need to read.
</details>

---

### Scenario 3: The Global Enterprise Support Bot 🌍

A multinational corp has a main website agent.
- When a user asks about "Shipping in Europe," the main agent must talk to a specialized "EU Logistics" agent.
- The "EU Logistics" agent is managed by a different team in a different country and runs in its own secure Google Cloud project.

**Task:** Which pattern allows agents in different projects/teams to work together?

<details>
<summary>Think it through, then check the reasoning</summary>

**Module 21's Distributed Graphs** — `internal/agents/researchspecialist`/`internal/agents/a2aorchestrator`'s own exact shape. The EU Logistics agent isn't a local `SubAgents` entry; it's a genuinely separate service, reachable via `agent/remoteagent/v2.NewA2A` and a `remoteagentv2.NewAgentCardProvider` pointed at its own `/.well-known/agent-card.json`. The main agent registers the resulting proxy exactly like a local sub-agent — the delegation mechanism (`transfer_to_agent`) is identical; only the transport crosses a real network boundary.

Worth remembering from Module 21's own review: a remote call's success can't be proven just by checking which agent's name shows up as an event's author — every failure path stamps the same local wrapper's name on its own synthesized error event. The real proof is error-free, non-empty content. 🔍
</details>

---

### Self-Reflection Questions 🤔
- Why is a "hybrid" approach (combining static and dynamic nodes) often the reality of production systems — and which of this repo's own six shipped packages comes closest to being a hybrid already?
- What are the risks of using `Mode: ModeTask`'s fluid, LLM-driven "collaborative team" pattern (Module 19) for a strictly regulated financial process, versus the fully deterministic edges of Modules 16/17?
- How does the graph mental model — nodes and edges you can point at in real Go code — help you communicate a system's design to a non-engineer stakeholder, compared to describing it as "a chatbot"?

<hr/>

> **Coming from Python?** 🐍 Python's own milestone poses these same three scenarios in terms of `Workflow`, `@node`, `sub_agents`, and `RemoteA2aAgent`. The intended answers are identical — this lab just names the real, shipped Go package and primitive for each, since every one of them already exists as tested code in this repo rather than a hypothetical design.
