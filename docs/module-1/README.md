# Module 1: Introduction to AI Agents & Google ADK (Go)

## Theory

### The Rise of AI Agents

In the rapidly evolving landscape of artificial intelligence, we are moving beyond simple chatbots and predictive models. The next frontier is **AI Agents**: autonomous systems that can understand goals, make plans, and use tools to interact with their environment to accomplish complex tasks.

Unlike traditional programs that follow a rigid set of instructions, an agent can reason, adapt, and act on its own. This paradigm shift is powered by the sophisticated reasoning capabilities of Large Language Models (LLMs) like Google's Gemini.

### What is an AI Agent?

An AI Agent is a system that can:

1. **Perceive its environment:** It takes in information, such as a user's request in natural language.
2. **Reason and Plan:** It uses an LLM as its "brain" to break down a high-level goal into a sequence of smaller, actionable steps.
3. **Act using Tools:** It executes those steps by interacting with its environment. This could mean calling an API, searching a database, running a piece of code, or even using another agent.
4. **Observe the Outcome:** It analyzes the results of its actions and adjusts its plan accordingly until the goal is achieved.

Think of an agent as an autonomous worker that you can delegate complex tasks to, moving from just "chatting" with an AI to collaborating with it.

### Introducing the Google Agent Development Kit (ADK) — for Go

Building robust, production-ready AI agents is a complex task. It involves much more than just prompting an LLM. You need to manage conversation history, handle tool integrations, orchestrate complex workflows, evaluate performance, and deploy the agent to a scalable infrastructure.

The **Google Agent Development Kit (ADK)** solves these challenges, letting you build, manage, evaluate, and deploy agents on the **Gemini Enterprise Agent Platform** (formerly known as Vertex AI). As of ADK 2.0 it has an official Go SDK: `google.golang.org/adk/v2` (source: `github.com/google/adk-go`, currently tagged `v2.4.0`; requires Go 1.25+ — this course targets Go 1.27). Install it with:

```bash
go get google.golang.org/adk/v2
```

Full docs: `https://adk.dev/get-started/go/`.

#### The ADK Philosophy

The ADK is built on a philosophy of **modularity, flexibility, and scalability**. It provides a set of core primitives that you can compose like building blocks to create everything from simple, single-purpose agents to complex, multi-agent systems.

#### Core Concepts of ADK 2.0: The Graph Architecture

ADK 2.0 moves away from monolithic agents and rigid hierarchies toward a flexible **Graph-based Architecture**. Here's how each primitive maps to the Go SDK:

| Graph Architecture concept | Go ADK (`google.golang.org/adk/v2`) |
|---|---|
| **Node** — a discrete unit of work | `workflow.NewFunctionNode(name, fn, workflow.NodeConfig{...})` for plain code, or an `llmagent.New(...)` agent used directly as a node for LLM-powered steps |
| **Edge** — flow of control and data between nodes | `workflow.Chain(workflow.Start, nodeA, nodeB, ...)` for sequential flow; explicit `workflow.Edge{}` values with `StringRoute`/`IntRoute`/`BoolRoute` for branching |
| **Workflow** — the container and orchestrator | `workflowagent.New(workflowagent.Config{Name, Description, Edges: edges})` — wraps a set of edges so it satisfies the same interface as a single agent |
| **App & Runner** — the infrastructure layer | Go *does* have a direct `Runner` equivalent: `runner.NewInMemory(appName, agent)` — its own doc comment calls it "equivalent to Python's InMemoryRunner" — then `(*Runner).Run(ctx, userID, sessionID, msg, cfg, opts...)` yields events, just like `run_debug`. The `get-started` quickstart instead goes through the higher-level `cmd/launcher` (`full.NewLauncher()` + `agent.NewSingleLoader(...)`) to get a full CLI/dev-UI/API-server app for free — use `runner` directly when you just need to run an agent programmatically (e.g. a script), `cmd/launcher` when you want a ready-made app shell. An `App` type exists for advanced cases (plugins, caching, lifecycle). |
| **Tool** — capability interfaces | `tool.Tool` interface; built-ins like `tool/geminitool.GoogleSearch{}`, custom ones via `tool/functiontool` |
| **Session & State** — context and memory across a run | Package `session` |

In this course, you will learn to think in **Graphs and Nodes**, mastering ADK 2.0 for Go to build scalable, production-grade AI applications.

### Key Takeaways

- AI Agents are autonomous systems that perceive, reason, and act using tools.
- **ADK 2.0** uses a **Graph Architecture** where Agents and Tools are **Nodes** connected by **Edges** — and it's available for Go today via `google.golang.org/adk/v2`.
- ADK-built agents deploy on the **Gemini Enterprise Agent Platform** (formerly Vertex AI), regardless of which language SDK built them.
- Go's SDK has a direct `Runner` equivalent (`runner.NewInMemory` + `Run`), just like Python — but the quickstart samples favor the higher-level `cmd/launcher` for a ready-made CLI/dev-UI app; reach for `runner` directly when you want to drive an agent from your own code.
