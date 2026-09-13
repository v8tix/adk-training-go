# Module 3: Your First Agent: The "Echo" Agent (Go)

## Theory

### The Core of an ADK Agent

At its heart, an ADK Agent is a blueprint that tells a Large Language Model (LLM) how to behave. In Go, that blueprint is `llmagent.Config`:

* **`Name`:** A unique identifier for your agent.
* **`Model`:** The `model.LLM` that acts as the agent's "brain" — built via `internal/infrastructure/llm.BuildModel` in this repo (local Ollama by default, Gemini opt-in).
* **`Instruction`:** The most critical part. This is the detailed prompt that defines the agent's persona, goals, and constraints.
* **`Description`:** A short, human-readable summary of the agent's purpose.

### Defining an Agent: Go Has One Way, Not Two

The Python course offers two ways to define an agent — a YAML config file, or Python code — and settles on Python as "the modern standard" for everything past this module. The Go SDK only has the programmatic form: `llmagent.New(llmagent.Config{...})`. There's no YAML-config path to skip here, because there was never a second option to begin with.

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

* **`go run ./cmd/echo-agent web --port 8080 webui api`** — starts a local Dev UI at `http://localhost:8080/ui/` (note the flag position: `web`'s own flags go directly after `web`, then the sub-launcher keywords). **Both `webui` and `api` are required together** — confirmed live in module-5: the Dev UI's frontend calls the REST API (`api`) for everything beyond serving its own static page, so `webui` alone starts a UI that 404s the moment you try to actually use it. This is the Go equivalent of `uv run adk web`.
* **`go run ./cmd/echo-agent console`** — a no-browser CLI chat mode. No direct parallel in the Python course; a bonus this launcher gives you for free.

**Caveat, confirmed by actually running it:** both the `console` and `web` UIs render the model's raw response, including its chain-of-thought, when using this course's default thinking-capable model — they don't apply the `Thought`-filtering this repo's own code does elsewhere (`firstAnswerText`). If you see visible reasoning text before the echoed answer in the UI, that's the SDK's own renderer, not a bug in this module's code.

There's no confirmed Go equivalent of the Python Dev UI's "Trace" tab specifically — the launcher's `web` mode does wire in OpenTelemetry (the `telemetry` package), which strongly suggests *some* observability view exists, but this wasn't verified directly this session.

### No `adk create` Scaffolding Wizard in Go

Python's `uv run adk create <name>` is an interactive wizard that generates a project skeleton. The Go SDK has no equivalent — `cmd/adkgo`, the SDK's own installable CLI, only has *deployment* subcommands (Cloud Run, Agent Engine), not project scaffolding. In Go, you just write `main.go` directly, the same way `cmd/verify-setup` and `cmd/echo-agent` in this repo already do.

### Key Takeaways
- An ADK agent in Go is defined by `llmagent.Config{Name, Model, Instruction, Description}` — no YAML alternative exists, unlike Python.
- `cmd/launcher` gives you `web` (Dev UI) and `console` (CLI chat) run modes for free once an agent is wrapped in `agent.NewSingleLoader`.
- There's no Go equivalent to `adk create`'s scaffolding wizard — you hand-write `main.go`.
- A thinking-capable model's raw reasoning shows through in the launcher's own UI rendering; filter it yourself (as this repo's tests do) whenever you need the answer alone, not just displayed to a human.
