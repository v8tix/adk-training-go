# Module 5: Running and Interacting with Agents (Go)

## Theory

### Three Modes, One Launcher

Python's course walks through three separate CLI commands (`adk web`, `adk run`, `adk api_server`). Go's SDK gives you the same three modes through one composable launcher, already partly in use since module-3:

| Python | Go | Confirmed by |
|---|---|---|
| `uv run adk run` | `console.NewLauncher()` | Used since module-3 |
| `uv run adk web` | `web.NewLauncher(webui.NewLauncher(), api.NewLauncher())` | Used since module-3; `api` addition this module |
| `uv run adk api_server` | `web.NewLauncher(..., api.NewLauncher())` — same package, no dev UI needed | New this module |

```go
l := universal.NewLauncher(
    console.NewLauncher(),
    web.NewLauncher(webui.NewLauncher(), api.NewLauncher()),
)
l.Execute(ctx, config, os.Args[1:])
```

### A Real Bug This Module Found (and Fixed Retroactively)

`webui` and `api` are **both required together**, confirmed live, not assumed: the Dev UI is a static frontend that calls the REST API for everything beyond rendering its own page. Register `webui` alone and the page loads, `/health` returns `200` — but `curl http://localhost:PORT/api/list-apps` 404s, because no REST API is actually running. This was true of `cmd/echo-agent` since module-3 and `cmd/support-analyzer` since module-4 — both fixed as part of this module's work. **`/health` returning `200` is not evidence the Dev UI works** — check an `/api/...` route.

### The REST API Server, Confirmed Route-by-Route

Go's `google.golang.org/adk/v2/server/adkrest` implements the same REST surface Python's `adk api_server` does, confirmed by reading its router source directly:

| Route | Method | Purpose |
|---|---|---|
| `/api/list-apps` | GET | List loaded agents by their `Name` |
| `/api/apps/{app_name}/users/{user_id}/sessions/{session_id}` | POST / GET / DELETE | Create / inspect / delete a session |
| `/api/apps/{app_name}/users/{user_id}/sessions` | GET | List a user's sessions |
| `/api/run` | POST | Non-streaming run |
| `/api/run_sse` | POST | Streaming run (Server-Sent Events) |
| `/api/run_live` | GET | Live/bidirectional run |

(`/api` is the default path prefix — `web`'s own `--help` documents `-path_prefix`, overridable if you ever need something else.)

### Two Real Syntax Divergences From Python, Confirmed Live

1. **Request bodies are camelCase, not snake_case.** `RunAgentRequest{AppName, UserId, SessionId, NewMessage}` (confirmed in `server/adkrest/internal/models/runtime.go`) serializes as `appName`/`userId`/`sessionId`/`newMessage`. Python's `app_name`/`user_id`/`session_id`/`new_message` gets rejected outright — Go's decoder is strict and returns a real `400` naming the unknown field. The nested message shape (`role`, `parts`, `text`) stays lowercase, matching Python.
2. **`app_name` in the URL/body is the agent's own `Name`, not a Python-style project-folder name.** This repo's Support Analyzer sets `llmagent.Config.Name = "support_analyzer_agent"`, so every session-lifecycle call targets `support_analyzer_agent`, not `support_analyzer` (there's no Go equivalent of a project directory to derive that second name from).

### The Trace View: Backend Confirmed, Frontend Not Independently Verified

Module-3 left the Trace View as "not confirmed." This module upgrades that: `server/adkrest/internal/routers/debug.go` defines real routes — `/dev/apps/{app_name}/debug/trace/session/{session_id}` and `/dev/apps/{app_name}/debug/trace/{event_id}` — so the data a Trace view would need is genuinely served by the Go SDK. Whether `webui`'s compiled frontend bundle renders this as a visible tab wasn't independently confirmed (that would mean reverse-engineering minified JS, not reading this repo's or the SDK's own source) — the honest claim is "the backend capability is real," not "the UI feature is pixel-identical to Python's."

### App & Runner, Once More

Same architecture note from earlier modules, now exercised a third way: `console` and `web`'s Dev UI both use `runner.NewInMemory` under the hood (confirmed since module-2/3); the REST API server manages sessions itself via the same `session.Service` interface, backed by an in-memory implementation by default (the launcher logs `"No session service configured. Using an in-memory one..."` on startup) — matching Python's note that `api_server` uses "a production-style environment where sessions and state are managed by the framework."

### Key Takeaways
- All three Python execution modes have a direct, confirmed Go equivalent — no gaps, no invented parallels.
- `webui` and `api` must be registered (and named on the command line) together — a real bug in this repo until this module, now fixed everywhere it appeared.
- The REST API's JSON is camelCase and keys sessions by the agent's `Name`, not a Python-style project folder name — two concrete divergences to expect, not a broken port.
- The Trace View's backend routes are real and confirmed in source; the frontend rendering wasn't independently verified.
