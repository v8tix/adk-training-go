# Module 1: Introduction to AI Agents & Google ADK (Go) 🤖

## Theory

### Why AI Agents Are a Big Deal 🚀

Chatbots that just answer questions? Old news. The exciting stuff right now is **AI Agents** — systems that can understand a goal, come up with a plan, and actually go *do* things to hit that goal.

Here's the key difference from a normal program: a regular script follows a fixed script (pun intended) — if X, do Y. An agent reasons, adapts, and takes action on its own. LLMs like Google's Gemini are what make this possible — they're the "brain" behind the whole thing.

### So What Exactly *Is* an Agent? 🧠

An AI Agent can:

1. **Perceive** — take in info, like a user typing a request in plain English.
2. **Reason & Plan** — use an LLM to break a big goal into smaller, doable steps.
3. **Act with Tools** — actually execute those steps: call an API, search a database, run code, even hand off to another agent.
4. **Observe & Adjust** — check how it went, and tweak the plan if needed, until the goal's done.

Think of it less like "chatting with an AI" and more like delegating a task to a capable coworker. 💪

### Meet the Google Agent Development Kit (ADK) — Now in Go! 🎉

Building a production-ready agent is more than just prompting an LLM and calling it a day. You've got conversation history to manage, tools to wire up, workflows to orchestrate, performance to evaluate, and infrastructure to scale.

That's exactly what the **Google Agent Development Kit (ADK)** handles. It lets you build, manage, evaluate, and deploy agents on the **Gemini Enterprise Agent Platform** (you might know it as Vertex AI). And as of ADK 2.0, there's an official Go SDK: `google.golang.org/adk/v2` (source: `github.com/google/adk-go`, currently tagged `v2.4.0`; needs Go 1.25+ — this course targets Go 1.27). Get it with:

```bash
go get google.golang.org/adk/v2
```

📚 Full docs live at `https://adk.dev/get-started/go/`.

#### The ADK Philosophy

Three words: **modularity, flexibility, scalability**. You get a set of core primitives you snap together like LEGO bricks — from a tiny single-purpose agent all the way up to a full multi-agent system.

#### ADK 2.0's Core Idea: Everything Is a Graph 🕸️

Forget monolithic agents and rigid hierarchies — ADK 2.0 thinks in **graphs**. Here's how each concept maps onto the Go SDK:

| Graph Architecture concept | Go ADK (`google.golang.org/adk/v2`) |
|---|---|
| **Node** — a discrete unit of work | `workflow.NewFunctionNode(name, fn, workflow.NodeConfig{...})` for plain code, or an `llmagent.New(...)` agent used directly as a node for LLM-powered steps |
| **Edge** — flow of control and data between nodes | `workflow.Chain(workflow.Start, nodeA, nodeB, ...)` for sequential flow; explicit `workflow.Edge{}` values with `StringRoute`/`IntRoute`/`BoolRoute` for branching |
| **Workflow** — the container and orchestrator | `workflowagent.New(workflowagent.Config{Name, Description, Edges: edges})` — wraps a set of edges so it satisfies the same interface as a single agent |
| **App & Runner** — the infrastructure layer | `runner.NewInMemory(appName, agent)` builds a runner you drive directly from your own code: `(*Runner).Run(ctx, userID, sessionID, msg, cfg, opts...)` yields the agent's events one at a time. The `get-started` quickstart instead goes through the higher-level `cmd/launcher` (`full.NewLauncher()` + `agent.NewSingleLoader(...)`) for a full CLI/dev-UI/API-server app, free. Reach for `runner` directly when you just need to run an agent from a script; reach for `cmd/launcher` when you want a ready-made app shell. There's also an `App` type for advanced cases (plugins, caching, lifecycle). |
| **Tool** — capability interfaces | `tool.Tool` interface; built-ins like `tool/geminitool.GoogleSearch{}`, custom ones via `tool/functiontool` |
| **Session & State** — context and memory across a run | Package `session` |

Throughout this course, you'll learn to think in **Graphs and Nodes**, mastering ADK 2.0 for Go to build scalable, production-grade AI applications. Let's go! 🚀

### Key Takeaways ✅

- AI Agents perceive, reason, and act using tools — autonomously.
- **ADK 2.0** thinks in a **Graph Architecture**: agents and tools are **Nodes**, connected by **Edges** — and yes, it's available for Go today via `google.golang.org/adk/v2`.
- ADK-built agents all deploy on the **Gemini Enterprise Agent Platform** (formerly Vertex AI), no matter which language SDK you built them with.
- Go gives you `runner.NewInMemory` + `Run` for driving an agent straight from your own code, and `cmd/launcher` when you want a ready-made CLI/dev-UI app instead — pick whichever fits your project.

<hr/>

> **Coming from Python?** 🐍 Go's `runner.NewInMemory` is the same idea as Python's `InMemoryRunner` — its own doc comment says so directly — and `cmd/launcher` is the closest thing to `adk web`'s ready-made app shell.
