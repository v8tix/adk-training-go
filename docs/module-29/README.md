# Module 29: Introduction to UI Integration (Go) 💬🌐

## Theory

### Why UI Integration Matters

Every agent so far has run from a CLI, a test, or the ADK's own Dev UI. A real end-user product usually needs its *own* interface — a chat widget in a product, a support-desk panel, a mobile app. Whatever it looks like, it talks to your agent the same way: over HTTP, to the REST API this repo has already been running since module-3.

### The UI Integration Landscape

| Approach | Best For | Key Features |
| :--- | :--- | :--- |
| **AG-UI Protocol** | Modern web apps (React/Next.js) | Pre-built components, official support |
| **Native ADK API** | Custom frameworks (Vue, Angular) | Full control, no dependencies |
| **Direct, in-process** | Data apps | No HTTP overhead |
| **Messaging Platforms** | Team bots (Slack, Teams) | Native platform UX |
| **Event-Driven** | High-scale, async workflows | Decoupled, scalable (Pub/Sub) |

**AG-UI**, developed through an official partnership between the ADK and CopilotKit teams, is Google's own recommended path for production React/Next.js apps — pre-built components handle streaming and state for you. It's a frontend library choice, not a Go backend API this repo's own SDK exposes anything special for — there's nothing to build here on the Go side beyond the REST API AG-UI's own adapter would call into, the exact same one this module's lab uses directly.

### The REST API You Already Have

`google.golang.org/adk/v2/cmd/launcher/web/api` — used by every `web ... api` invocation since module-3 — is the entire backend this module needs. No new server code, just a genuinely new kind of client: a hand-written HTML/JavaScript chat page, calling the same `/run_sse` streaming endpoint the ADK Dev UI itself uses internally.

```go
config := &launcher.Config{AgentLoader: agent.NewSingleLoader(rootAgent)}
l := universal.NewLauncher(console.NewLauncher(), web.NewLauncher(webui.NewLauncher(), api.NewLauncher()))
l.Execute(ctx, config, os.Args[1:])
```

Run it as `web --port=9093 api -webui_address http://localhost:9094` and you have a real, streaming REST backend — no `webui`, no Dev UI, just the API a custom client talks to.

### A Real, Confirmed Naming Difference: `/api`, camelCase, and CORS

Three concrete things a Go client must get right that a Python lab client wouldn't need to think about the same way:

1. **The REST API is mounted under `/api` by default** — confirmed in `cmd/launcher/web/api/api.go`'s own `-path_prefix` flag (default `"/api"`). Every endpoint this module's client calls is `http://localhost:9093/api/...`, not the bare root Python's `adk api_server` serves at.
2. **Request bodies are camelCase** (`appName`, `userId`, `sessionId`, `newMessage`), confirmed in the SDK's own `server/adkrest/internal/models/runtime.go` struct tags — Python's equivalent client sends snake_case (`app_name`, `user_id`, `session_id`, `new_message`) for the exact same request.
3. **CORS is controlled by `-webui_address`, not `--allow_origins`** — confirmed in the same `api.go`. Same purpose (allow a browser page on a different origin/port to call the API), different flag name.

### A Real, Confirmed Gotcha This Course's Own Default Model Creates

Confirmed live, hitting the real `/run_sse` endpoint directly: this repo's default thinking-capable Ollama model (`qwen3.8:27b`) streams a part marked `"thought": true` — its own raw reasoning — *before* the real final answer, in the very same event. A client that renders every part's text verbatim would show the user the model's internal monologue instead of (or alongside) its actual answer:

```json
{"content":{"role":"model","parts":[
  {"text":"The user wants me to say hello in exactly three words...","thought":true},
  {"text":"Hello there friend."}
]}}
```

This is the exact same "`Parts[0]` isn't the final answer" lesson this course has carried since its very first agent — here it has to be applied in JavaScript, in the browser, since that's genuinely where the filtering has to happen for a UI client. Skip any part with `thought === true` before appending its text to the chat.

### Advantages of SSE for Chat

Server-Sent Events stream text to the client as the model generates it — no waiting for the whole response, no bidirectional handshake overhead a chat UI doesn't need (unlike WebSockets). One `fetch`, one `ReadableStream`, `data: ...` lines parsed as they arrive.

### Session Persistence Is Your Job

The lab's own client generates a fresh `sessionId` on every page load — the conversation history is genuinely lost on refresh. A real app persists it in `localStorage` or a cookie so the same session survives a reload; this lab intentionally keeps that step out of scope to stay focused on the streaming mechanics themselves.

### Key Takeaways ✅
- The REST API server this repo has run since module-3 (`cmd/launcher/web/api`) is the entire backend a custom UI needs — no new Go server code, just a new client.
- Three real, confirmed differences from Python's own client: the `/api` path prefix, camelCase request bodies, and `-webui_address` instead of `--allow_origins` for CORS.
- This repo's default thinking-capable model genuinely streams `thought: true` parts through `/run_sse` — a browser client must filter them, verified live against the real endpoint, not assumed.
- AG-UI/CopilotKit is the officially-recommended path for a real React/Next.js production app, abstracting away exactly the streaming/session mechanics this lab builds by hand — there's no Go-SDK-side component to build for it, since it's a frontend library choice.

<hr/>

> **Coming from Python?** 🐍 Python's own lab client hits the bare root (`/run_sse`, `/apps/...`) with snake_case body fields and `--allow_origins` for CORS — this Go mirror's client hits `/api/...` with camelCase fields and `-webui_address`, for the exact same underlying reasons. Python's lab doesn't need to filter `thought` parts because its own module doesn't discuss a thinking-capable default model the way this course's Go mirror's own local-first setup does — the SSE mechanics themselves are otherwise identical on both sides.
