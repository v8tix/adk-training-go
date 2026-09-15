# Lab 8: Building a "Researcher" Agent with Google Search (Go)

## Goal

Build an agent that can search the web to answer questions about current events — this repo's mirror of Python's "Researcher" lab, using the ADK's built-in `google_search` tool.

## Lab Tasks

### 1. Read `internal/agents/researcher/agent.go`

Same shape as `internal/agents/visualcatalog` (module-7) — `BuildRootAgent(llmModel)` resolves its own prompt internally — with one addition: `Tools: []tool.Tool{geminitool.GoogleSearch{}}` in the `llmagent.Config`. That one field is the entire wiring needed for a built-in tool.

### 2. Read `internal/agents/researcher/prompts/researcher_instruction.md`

Notice it doesn't just say "you have a search tool" — it explicitly says *when* to use it (current events, live data, anything time-sensitive) and when not to (anything answerable from general knowledge). This is deliberate: an agent with a tool but no guidance on when to use it may search unnecessarily, or fail to search when it should.

### 3. Run it — console mode

```bash
go run ./cmd/researcher console
```

(Requires `GOOGLE_AI_STUDIO_API_KEY` in your environment or `.env` — see `.env.example`.)

Real, confirmed output from this exact command:

```
🔎 researcher using gemini-3.5-flash

User -> What is the current weather in Tokyo right now?
Agent -> As of the evening of Monday, September 14, 2026, in Tokyo, the current weather conditions are:

* **Temperature:** Approximately 83°F to 84°F (28°C)
* **Conditions:** Mostly cloudy with passing clouds
...
```

A real, current, grounded answer — not something a model's own training data could contain.

### 4. Run it — Dev UI mode, and inspect the Trace view

> **Port choice:** this lab uses `9091` instead of the ADK launcher's default of `8080` — `8080` is commonly already occupied by other local dev tools (see module-3's lab for a confirmed real collision on this session's own machine, from Docker Desktop's own proxy). If `9091` is also taken on your machine, check with `lsof -i :9091` and pick any other free port instead, adjusting the URL below **and** the `-api_server_address` flag below to match.
>
> **A real, confirmed gotcha: `--port` alone is not enough.** The Dev UI's frontend learns where to call the API from a *separate* flag, `webui`'s own `-api_server_address`, which defaults to the hardcoded `http://localhost:8080/api` regardless of `--port` — confirmed live (see module-3's README for the full finding). That's why the command below sets it explicitly.

```bash
go run ./cmd/researcher web --port 9091 webui -api_server_address http://localhost:9091/api api
```

Open `http://localhost:9091/`, ask the same weather question, then open the **Trace** view for that turn. You should see the `google_search` function call and its result as a distinct step before the model's final answer — Python's own verification method, still available here.

Confirmed live in this repo: `curl http://localhost:9091/api/list-apps` returns `["researcher_agent"]`, and the Dev UI itself (`http://localhost:9091/`, redirecting to `/ui/`) returns `200`.

### 5. See the local-backend limitation, on purpose

Open `internal/agents/researcher/agent.go`'s package doc comment and `cmd/researcher/main.go`'s doc comment — both explain that `MODEL_TYPE` is forced to `gemini` in code. If you're curious why, try building a `researcher_agent` against the Ollama backend yourself (see `temp/module-8/probe` if you have access to this repo's working tree, or just read its confirmed output below) — you'll get the same real error this module is about:

```
RUN ERROR: openai: non-function tools are not supported (tool 0)
```

## Self-Reflection Questions
- Why is it important to explicitly instruct the agent *when* to use the `google_search` tool? What might happen if you just gave it the tool with no instructions? (Look at `researcher_instruction.md`'s `# Constraints` section for this repo's answer.)
- The Python version of this lab claims `google_search` requires an Agent Platform (Vertex AI) configuration. This repo found that claim doesn't hold for its own setup — what's the risk of taking a framework's documentation at face value instead of testing a claim like that directly?
- How does giving an agent access to real-time information fundamentally change the kinds of problems it can solve compared to an agent that only relies on its internal knowledge?
- `event.GroundingMetadata` lets this repo's test verify the tool fired without a human opening the Trace view. What's a downside of relying only on an automated structural check like that, instead of ever looking at the Trace view yourself?

<hr/>

### Looking for the solution?

Hint: read `internal/agents/researcher/agent.go` and `cmd/researcher/main.go` for the real mechanism — attaching a built-in tool is one field on `llmagent.Config`, and the rest is the same launcher pattern module-5 already established.
