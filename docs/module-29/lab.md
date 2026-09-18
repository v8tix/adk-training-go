# Lab 29: Building a Simple Custom Chat UI (Go) 💬🌐

## Goal

Build a standalone HTML/JavaScript chat client, backed by this repo's own REST API server, demonstrating exactly what AG-UI's pre-built components abstract away: the raw request/response and SSE streaming mechanics.

### Prerequisites

None beyond what every prior module needed — this lab runs entirely on the local Ollama default, no cloud credentials required.

## Lab Tasks

### 1. Read `internal/agents/uiagent/agent.go`

A deliberately trivial, tool-less agent — this module's real lesson is the client talking to it, not what the agent itself can do.

### 2. Read `cmd/ui-agent/main.go`

Standard launcher wiring, `api`-only relevant here (no `webui`, no `console` — this module's own custom client is the "UI"). Run it:

```bash
go run ./cmd/ui-agent web --port=9093 api -webui_address http://localhost:9094
```

`-webui_address` is `api`'s own CORS flag (confirmed in the vendored SDK's `cmd/launcher/web/api/api.go`) — set to the static client's own origin (port 9094, below) so the browser's cross-origin `fetch` calls succeed.

### 3. Read `cmd/ui-agent/main_test.go`

Builds `adkrest.NewServer` directly (no launcher, no CLI) wrapped in `httptest.NewServer` — a real, in-process HTTP server. `TestRunSSE_StreamsBothThoughtAndFinalParts` drives the exact same session-create + `/run_sse` flow the browser client will, and proves the module's own central risk is real, not hypothetical: the live SSE stream genuinely contains a `"thought":true` part *and* a real final answer, in the same response. This is the module's server-side regression proof; the client's own JavaScript filtering logic is verified live in a real browser instead (Step 5) — this repo has no JavaScript test runner to unit-test it with directly.

### 4. Read `cmd/ui-client-server/static/index.html`

The real, hand-written chat client. Three things to notice against Python's own lab version:
- Every endpoint is prefixed `/api` (`API_BASE = 'http://localhost:9093/api'`) — the Go REST API's own default path prefix.
- The `/run_sse` request body uses camelCase (`appName`, `userId`, `sessionId`, `newMessage`) — confirmed against this SDK's own `RunAgentRequest` struct tags.
- The SSE-parsing loop explicitly skips any part with `part.thought` truthy before appending its text — without this, the chat would show the model's raw reasoning trace to the user.

### 5. Run the full stack — two processes, one browser 🖥️

**Terminal 1 (agent server):**
```bash
go run ./cmd/ui-agent web --port=9093 api -webui_address http://localhost:9094
```

**Terminal 2 (static client server):**
```bash
go run ./cmd/ui-client-server
```

Then open `http://localhost:9094` in a browser and send a message.

**Real, confirmed verification from this exact setup** — driven with `chromedp` (a real headless-Chrome Go driver, `github.com/chromedp/chromedp`), not just a manual click-through: navigated to the page, typed "Say hello in exactly three words.", submitted the form, and polled the DOM until the assistant's message div held real text.

```
=== Rendered assistant message ===
Hello to you
===================================
✅ No obvious reasoning-trace markers found in the rendered text — thought-filtering appears to work.
```

The server's own `/run_sse` response for that exact turn genuinely contained a `thought: true` part (see Step 3's test) — the rendered page correctly shows only the real three-word answer, proving the client's filtering logic actually works end-to-end, not just in isolation.

### Checkpoint

- [ ] `go test ./cmd/ui-agent/... -race` passes
- [ ] A real browser (or a headless-Chrome check) shows only the final answer — no leaked reasoning text — after sending a message

## Self-Reflection Questions 🤔
- Server-Sent Events stream text to the client as it's generated. What would a traditional (non-streaming) request-response chat endpoint feel like to a user, compared to this?
- This lab's client generates a new `sessionId` on every page load. What would you need to change to make a conversation survive a page refresh?
- `cmd/ui-agent/main_test.go` builds `adkrest.NewServer` directly instead of going through `cmd/launcher`. Why does that mean the test hits `/run_sse` instead of `/api/run_sse`, and what does that tell you about where the `/api` prefix actually comes from?
- The AG-UI Protocol would replace most of `static/index.html`'s own JavaScript with a handful of pre-built React components. What exactly would you be trading away by adopting it — and what would you gain?

<hr/>

### Looking for the solution? 🔍

Hint: read `internal/agents/uiagent/agent.go`, `cmd/ui-agent/main.go`, and `cmd/ui-client-server/static/index.html` for the real mechanism — a trivial agent, a REST API launcher, and a hand-written chat client talking to it over `/api/run_sse`.
