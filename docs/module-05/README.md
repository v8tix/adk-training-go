# Module 5: Running and Interacting with Agents (Go) 🖥️

## Theory

### Three Modes, One Composable Launcher 🎛️

You've been using `console.NewLauncher()` and `web.NewLauncher(webui.NewLauncher(), api.NewLauncher())` since module-3. Now let's add a third mode: the REST API on its own, no Dev UI attached — just drop `webui.NewLauncher()` and keep `api.NewLauncher()`. All three modes come from the same composable `universal.NewLauncher`:

```go
l := universal.NewLauncher(
    console.NewLauncher(),
    web.NewLauncher(webui.NewLauncher(), api.NewLauncher()),
)
l.Execute(ctx, config, os.Args[1:])
```

### A Real Bug This Module Found (and Fixed Retroactively) 🐛

Here's a gotcha confirmed live, not just assumed: `webui` and `api` are **both required together**. The Dev UI is a static frontend that calls the REST API for everything beyond rendering its own page. Register `webui` alone and the page loads, `/health` returns `200` — but `curl http://localhost:PORT/api/list-apps` 404s, because no REST API is actually running. This bit `cmd/echo-agent` since module-3 and `cmd/support-analyzer` since module-4 — both fixed as part of this module's work. **Lesson: `/health` returning `200` is NOT evidence the Dev UI works** — go check an `/api/...` route instead.

### The REST API Server, Confirmed Route-by-Route 🗺️

`google.golang.org/adk/v2/server/adkrest` gives you a real REST surface for driving an agent from any HTTP client — confirmed by reading its router source directly:

| Route | Method | Purpose |
|---|---|---|
| `/api/list-apps` | GET | List loaded agents by their `Name` |
| `/api/apps/{app_name}/users/{user_id}/sessions/{session_id}` | POST / GET / DELETE | Create / inspect / delete a session |
| `/api/apps/{app_name}/users/{user_id}/sessions` | GET | List a user's sessions |
| `/api/run` | POST | Non-streaming run |
| `/api/run_sse` | POST | Streaming run (Server-Sent Events) |
| `/api/run_live` | GET | Live/bidirectional run |

(`/api` is the default path prefix — `web`'s own `--help` documents `-path_prefix` if you ever need to change it.)

### Two Things to Know About the Request Shape 📋

1. **Request bodies are camelCase.** `RunAgentRequest{AppName, UserId, SessionId, NewMessage}` (confirmed in `server/adkrest/internal/models/runtime.go`) serializes as `appName`/`userId`/`sessionId`/`newMessage`. The decoder is strict too — get the casing wrong and you'll get a real `400` naming the offending field. The nested message shape (`role`, `parts`, `text`) stays lowercase, though.
2. **`app_name` in the URL/body is the agent's own `Name`.** This repo's Support Analyzer sets `llmagent.Config.Name = "support_analyzer_agent"`, so every session-lifecycle call needs to target that exact string.

### The Trace View: Backend Confirmed, Frontend Not Independently Verified 🔍

Module-3 left the Trace View as "not confirmed." Good news — we can upgrade that a bit: `server/adkrest/internal/routers/debug.go` defines real routes — `/dev/apps/{app_name}/debug/trace/session/{session_id}` and `/dev/apps/{app_name}/debug/trace/{event_id}` — so the data a Trace view would need really is served by the Go SDK. Whether `webui`'s compiled frontend bundle actually renders this as a visible tab? We didn't independently confirm that (that would mean reverse-engineering minified JS, not reading source code) — so the honest claim here is "the backend capability is real," not "the UI feature is pixel-identical to Python's."

### App & Runner, Once More 🔄

Same architecture note as earlier modules, now exercised a third way: `console` and `web`'s Dev UI both use `runner.NewInMemory` under the hood (confirmed since module-2/3); the REST API server manages sessions itself via the same `session.Service` interface, backed by an in-memory implementation by default (the launcher logs `"No session service configured. Using an in-memory one..."` on startup) — a production-style setup where the framework owns sessions and state, not whatever code happens to call `Run`.

### Key Takeaways ✅
- One composable launcher (`universal.NewLauncher`) gives you a headless CLI, a full Dev UI, and a standalone REST server — pick sub-launchers to match what you need.
- `webui` and `api` must be registered (and named on the command line) together — a real bug in this repo until this module, now fixed everywhere it showed up.
- The REST API's JSON is camelCase and keys sessions by the agent's own `Name` — good to know before your first request gets rejected. 😅
- The Trace View's backend routes are real and confirmed in source; the frontend rendering wasn't independently verified.

<hr/>

> **Coming from Python?** 🐍 `console.NewLauncher()`, `web.NewLauncher(webui.NewLauncher(), api.NewLauncher())`, and `web.NewLauncher(api.NewLauncher())` (no `webui`) are the direct equivalents of `uv run adk run`, `uv run adk web`, and `uv run adk api_server`, respectively. Two real syntax differences to expect: request bodies are camelCase, not snake_case (Python's `app_name` gets rejected outright), and `app_name` is the agent's own `Name` field, not a Python-style project-folder name.
