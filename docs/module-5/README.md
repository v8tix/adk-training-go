# Module 5: Running and Interacting with Agents (Go)

## Theory

### Three Modes, One Composable Launcher

You've already been using `console.NewLauncher()` and `web.NewLauncher(webui.NewLauncher(), api.NewLauncher())` since module-3. This module adds a third way to run the same agent: the REST API on its own, no Dev UI attached, by leaving `webui.NewLauncher()` out and keeping only `api.NewLauncher()`. All three modes come from the same composable `universal.NewLauncher`:

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

`google.golang.org/adk/v2/server/adkrest` gives you a real REST surface for driving an agent from any HTTP client, confirmed by reading its router source directly:

| Route | Method | Purpose |
|---|---|---|
| `/api/list-apps` | GET | List loaded agents by their `Name` |
| `/api/apps/{app_name}/users/{user_id}/sessions/{session_id}` | POST / GET / DELETE | Create / inspect / delete a session |
| `/api/apps/{app_name}/users/{user_id}/sessions` | GET | List a user's sessions |
| `/api/run` | POST | Non-streaming run |
| `/api/run_sse` | POST | Streaming run (Server-Sent Events) |
| `/api/run_live` | GET | Live/bidirectional run |

(`/api` is the default path prefix — `web`'s own `--help` documents `-path_prefix`, overridable if you ever need something else.)

### Two Things to Know About the Request Shape

1. **Request bodies are camelCase.** `RunAgentRequest{AppName, UserId, SessionId, NewMessage}` (confirmed in `server/adkrest/internal/models/runtime.go`) serializes as `appName`/`userId`/`sessionId`/`newMessage`. The decoder is strict — an unrecognized field name gets rejected with a real `400` naming it, so getting the casing right matters. The nested message shape (`role`, `parts`, `text`) stays lowercase.
2. **`app_name` in the URL/body is the agent's own `Name`.** This repo's Support Analyzer sets `llmagent.Config.Name = "support_analyzer_agent"`, so every session-lifecycle call targets that exact string.

### The Trace View: Backend Confirmed, Frontend Not Independently Verified

Module-3 left the Trace View as "not confirmed." This module upgrades that: `server/adkrest/internal/routers/debug.go` defines real routes — `/dev/apps/{app_name}/debug/trace/session/{session_id}` and `/dev/apps/{app_name}/debug/trace/{event_id}` — so the data a Trace view would need is genuinely served by the Go SDK. Whether `webui`'s compiled frontend bundle renders this as a visible tab wasn't independently confirmed (that would mean reverse-engineering minified JS, not reading this repo's or the SDK's own source) — the honest claim is "the backend capability is real," not "the UI feature is pixel-identical to Python's."

### App & Runner, Once More

Same architecture note from earlier modules, now exercised a third way: `console` and `web`'s Dev UI both use `runner.NewInMemory` under the hood (confirmed since module-2/3); the REST API server manages sessions itself via the same `session.Service` interface, backed by an in-memory implementation by default (the launcher logs `"No session service configured. Using an in-memory one..."` on startup) — a production-style environment where sessions and state are managed by the framework, not by whatever code happens to call `Run`.

### Key Takeaways
- One composable launcher (`universal.NewLauncher`) gives you a headless CLI, a full Dev UI, and a standalone REST server — pick sub-launchers to match what you need.
- `webui` and `api` must be registered (and named on the command line) together — a real bug in this repo until this module, now fixed everywhere it appeared.
- The REST API's JSON is camelCase and keys sessions by the agent's own `Name` — worth knowing before your first request gets rejected.
- The Trace View's backend routes are real and confirmed in source; the frontend rendering wasn't independently verified.

<hr/>

> **Coming from Python?** `console.NewLauncher()`, `web.NewLauncher(webui.NewLauncher(), api.NewLauncher())`, and `web.NewLauncher(api.NewLauncher())` (no `webui`) are the direct equivalents of `uv run adk run`, `uv run adk web`, and `uv run adk api_server`, respectively. Two real syntax differences to expect: request bodies are camelCase, not snake_case (Python's `app_name` gets rejected outright), and `app_name` is the agent's own `Name` field, not a Python-style project-folder name.
