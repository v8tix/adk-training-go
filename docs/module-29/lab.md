# Lab 29: Building a Simple Custom Chat UI (Go) 💬🌐

## Goal

Build a standalone HTML/JavaScript chat client, backed by this repo's own REST API server, demonstrating exactly what AG-UI's pre-built components abstract away: the raw request/response and SSE streaming mechanics.

### Prerequisites

This lab itself runs entirely on the local Ollama default, no cloud credentials required. Its committed browser test (Step 6) needs Google Chrome or Chromium installed — see the top-level [README's Tooling section](../../README.md#-tooling) — and skips cleanly if it isn't found.

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

Builds `adkrest.NewServer` directly (no launcher, no CLI) wrapped in `httptest.NewServer` — a real, in-process HTTP server. `TestRunSSE_StreamsBothThoughtAndFinalParts` drives the exact same session-create + `/run_sse` flow the browser client will, and proves the module's own central risk is real, not hypothetical: the live SSE stream genuinely contains a `"thought":true` part *and* a real final answer, in the same response. This is the module's server-side regression proof; the client's own JavaScript filtering logic is verified live in a real browser instead (Step 6's committed test) — this repo has no JavaScript test runner to unit-test it with directly.

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

### 6. Read `cmd/ui-agent/browser_test.go`

The full-stack check from Step 5, permanently committed instead of a one-off script: `TestChatUI_RendersStreamedAnswerWithoutLeakingThought_Ollama` builds the real `cmd/ui-agent` binary, launches it through its own real CLI (`web --port=9093 api -webui_address ...` — the only test in this module that exercises the launcher's actual `/api` mounting and CORS wiring, not a bypassed direct `adkrest.NewServer` construction), serves the real `static/index.html` in-process, and drives a real headless Chrome (`github.com/chromedp/chromedp`) against it — typing a message, submitting, and reading the rendered DOM back. It skips cleanly (not fails) if Chrome/Chromium isn't installed, or if port 9093 is already taken by something else.

```bash
go test ./cmd/ui-agent/... -run TestChatUI -race -v
```

Real, confirmed output from this exact command:

```
=== RUN   TestChatUI_RendersStreamedAnswerWithoutLeakingThought_Ollama
--- PASS: TestChatUI_RendersStreamedAnswerWithoutLeakingThought_Ollama (8.60s)
```

The server's own `/run_sse` response for that turn genuinely contained a `thought: true` part (see Step 3's own test) — the rendered page correctly shows only the real final answer, proving the client's filtering logic actually works end-to-end, in a real browser, on every test run — not just once, by hand, during Build.

### Checkpoint

- [ ] `go test ./cmd/ui-agent/... -race` passes (both the server-side and the full-stack browser test)
- [ ] With Chrome/Chromium installed, `TestChatUI_RendersStreamedAnswerWithoutLeakingThought_Ollama` actually runs (not skips) and passes

## Self-Reflection Questions 🤔
- Server-Sent Events stream text to the client as it's generated. What would a traditional (non-streaming) request-response chat endpoint feel like to a user, compared to this?
- This lab's client generates a new `sessionId` on every page load. What would you need to change to make a conversation survive a page refresh?
- `cmd/ui-agent/main_test.go` builds `adkrest.NewServer` directly instead of going through `cmd/launcher`. Why does that mean the test hits `/run_sse` instead of `/api/run_sse`, and what does that tell you about where the `/api` prefix actually comes from?
- The AG-UI Protocol would replace most of `static/index.html`'s own JavaScript with a handful of pre-built React components. What exactly would you be trading away by adopting it — and what would you gain?

<hr/>

### Looking for the solution? 🔍

Hint: read `internal/agents/uiagent/agent.go`, `cmd/ui-agent/main.go`, and `cmd/ui-client-server/static/index.html` for the real mechanism — a trivial agent, a REST API launcher, and a hand-written chat client talking to it over `/api/run_sse`.
