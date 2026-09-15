# Module 3: Your First Agent: The "Echo" Agent (Go)

## Theory

### The Core of an ADK Agent

At its heart, an ADK Agent is a blueprint that tells a Large Language Model (LLM) how to behave. In Go, that blueprint is `llmagent.Config`:

* **`Name`:** A unique identifier for your agent.
* **`Model`:** The `model.LLM` that acts as the agent's "brain" — built via `internal/infrastructure/llm.BuildModel` in this repo (local Ollama by default, Gemini opt-in).
* **`Instruction`:** The most critical part. This is the detailed prompt that defines the agent's persona, goals, and constraints.
* **`Description`:** A short, human-readable summary of the agent's purpose.

### Defining an Agent in Code

You build an agent with `llmagent.New(llmagent.Config{...})` — plain Go code, no config file to write or parse:

```go
rootAgent, err := llmagent.New(llmagent.Config{
    Name:        "echo_agent",
    Model:       llmModel,
    Description: "An agent that repeats the user's input.",
    Instruction: instruction,
})
```

`instruction` doesn't come from a Go string constant — instructions get long and repetitive fast (see `prompts/echo_instruction.md`), and a multi-line backtick constant is awkward to read and edit. Instead it comes from `internal/infrastructure/prompts`, a small shared cache every `cmd/` program can register its own prompts into:

```go
//go:embed prompts/*.md
var promptFS embed.FS

func init() {
    promptFiles, _ := fs.Sub(promptFS, "prompts") // required — see below
    prompts.Register("echo-agent", promptFiles, ".md")
}

// in main():
instruction, err := prompts.Get("echo-agent/echo_instruction")
```

`//go:embed` is directory-scoped — a directive can only reach files in or below the directory of the file that declares it, so the embed itself can never move into the shared package; each `cmd/` program still embeds its own local `prompts/*.md`. What's shared is the cache and lookup, keyed by `"<namespace>/<name>"` (namespace is the program's own name, so two programs' same-named prompt files can't collide). `fs.Sub(promptFS, "prompts")` is required, not optional: `//go:embed prompts/*.md` keeps the `"prompts/"` prefix inside the resulting `embed.FS`, so skipping `fs.Sub` silently registers everything one level too deep.

### Running the Agent: `cmd/launcher`

Defining the agent is only the first step. This repo's `cmd/echo-agent` wraps `rootAgent` with `agent.NewSingleLoader(rootAgent)` and hands it to a launcher:

```go
config := &launcher.Config{AgentLoader: agent.NewSingleLoader(rootAgent)}
l := universal.NewLauncher(console.NewLauncher(), web.NewLauncher(webui.NewLauncher(), api.NewLauncher()))
l.Execute(ctx, config, os.Args[1:])
```

This gives two run modes for free, confirmed by running both this session:

* **`go run ./cmd/echo-agent web --port 9091 webui -api_server_address http://localhost:9091/api api`** — starts a local Dev UI at `http://localhost:9091/ui/` (note the flag position: `web`'s own flags go directly after `web`, then each sub-launcher keyword followed by its own flags; the lab uses `9091` instead of the launcher's default `8080`, which is commonly already occupied by other local dev tools — see the lab for a confirmed real collision). **Both `webui` and `api` are required together** — confirmed live in module-5: the Dev UI's frontend calls the REST API (`api`) for everything beyond serving its own static page, so `webui` alone starts a UI that 404s the moment you try to actually use it. **`webui`'s `-api_server_address` must be set explicitly whenever `--port` isn't 8080** — it's a separate flag telling the browser-side frontend where to call the API, with its own hardcoded `http://localhost:8080/api` default independent of `--port`; skip it and the Dev UI keeps trying port 8080 no matter what port the server actually runs on, confirmed live via `/ui/assets/config/runtime-config.json`.
* **`go run ./cmd/echo-agent console`** — a no-browser CLI chat mode, a nice option whenever you don't want a browser open at all.

**Caveat, confirmed by actually running it:** both the `console` and `web` UIs render the model's raw response, including its chain-of-thought, when using this course's default thinking-capable model — they don't apply the `Thought`-filtering this repo's own code does elsewhere (`firstAnswerText`). If you see visible reasoning text before the echoed answer in the UI, that's the SDK's own renderer, not a bug in this module's code.

There's no confirmed Go equivalent of the Python Dev UI's "Trace" tab specifically — the launcher's `web` mode does wire in OpenTelemetry (the `telemetry` package), which strongly suggests *some* observability view exists, but this wasn't verified directly this session.

### Starting a New Agent Program

There's no scaffolding command to generate a new agent project — you just write `main.go` directly, the same way `cmd/verify-setup` and `cmd/echo-agent` in this repo already do. `cmd/adkgo`, the SDK's own installable CLI, is for *deployment* (Cloud Run, Agent Engine), not project generation.

### Key Takeaways
- An ADK agent in Go is defined entirely in code: `llmagent.Config{Name, Model, Instruction, Description}`.
- `cmd/launcher` gives you `web` (Dev UI) and `console` (CLI chat) run modes for free once an agent is wrapped in `agent.NewSingleLoader`.
- A new agent program starts as a hand-written `main.go` — there's no generator command, so you build it up from the patterns in this repo's own `cmd/` programs.
- A thinking-capable model's raw reasoning shows through in the launcher's own UI rendering; filter it yourself (as this repo's tests do) whenever you need the answer alone, not just displayed to a human.

<hr/>

> **Coming from Python?** Python's course offers a second way to define an agent (a YAML config file) and a scaffolding wizard (`uv run adk create <name>`) that generates a project skeleton. Go's SDK has neither — `llmagent.Config` in code is the only way to define an agent, and every `cmd/` program here started as a hand-written `main.go`.
