# Lab 5: Exploring Different Execution Modes (Go)

## Goal

Run and interact with the **Support Analyzer** agent (from modules 4-4.5) using all three execution modes the ADK Go SDK provides — Dev UI, headless CLI, and REST API — including the session-lifecycle lesson: a run fails without a session, succeeds after creating one.

## Lab Tasks

### 1. `web --port 9091 webui -api_server_address ... api` (Dev UI)

> **Port choice:** this lab uses `9091` instead of the ADK launcher's default of `8080` — `8080` is commonly already occupied by other local dev tools (see module-3's lab for a confirmed real collision on this session's own machine, from Docker Desktop's own proxy). If `9091` is also taken on your machine, check with `lsof -i :9091` and pick any other free port instead, adjusting every URL in this lab **and** the `-api_server_address` flag below to match — including in Task 3 below, which reuses this same port.
>
> **A real, confirmed gotcha: `--port` alone is not enough.** The Dev UI's frontend learns where to call the API from a *separate* flag, `webui`'s own `-api_server_address`, which defaults to the hardcoded `http://localhost:8080/api` regardless of `--port` — confirmed live (see module-3's README for the full finding). That's why the command below sets it explicitly. Task 3's API-only mode doesn't need this flag — there's no Dev UI frontend involved there, `curl` talks to the API directly.

```bash
go run ./cmd/support-analyzer web --port 9091 webui -api_server_address http://localhost:9091/api api
```

**Both `webui` and `api` are required** — the Dev UI's frontend calls the REST API for everything beyond serving its static page. Open `http://localhost:9091/ui/`, interact with the agent, and try the Trace view if the frontend exposes one (the backend routes for it are real — see the README — this repo doesn't claim the frontend UI was independently verified).

### 2. `console` (Headless CLI)

```bash
go run ./cmd/support-analyzer console
```

Interact with the agent directly in your terminal — no browser needed.

### 3. `web --port 9091 api` (REST API Server)

Stop the previous mode, then:

```bash
go run ./cmd/support-analyzer web --port 9091 api
```

Open a **separate terminal** to act as the client.

**Step A (The Failure):** try `run_sse` without creating a session first:

```bash
curl -X POST http://localhost:9091/api/run_sse \
     -H "Content-Type: application/json" \
     -d '{
           "appName": "support_analyzer_agent",
           "userId": "test_user",
           "sessionId": "missing_session",
           "newMessage": {"role": "user", "parts": [{"text": "Hello"}]}
         }'
```

Real, confirmed response: `404`, `failed to find the session: failed to get session: session not found: "missing_session"`.

**Note the two things that differ from the Python lab's literal curl commands, both confirmed live, not assumed:**
- Fields are **camelCase** (`appName`, `userId`, `sessionId`, `newMessage`) — Python's snake_case (`app_name`, ...) gets rejected with a real `400: unknown field`.
- `appName` is the agent's own **`Name`** (`support_analyzer_agent`, set in `cmd/support-analyzer/main.go`'s `buildRootAgent`) — not a Python-style project-folder name.

**Step B (The Fix):** create the session explicitly:

```bash
curl -X POST http://localhost:9091/api/apps/support_analyzer_agent/users/test_user/sessions/test_session
```

Real, confirmed response: `200`, `{"id":"test_session","appName":"support_analyzer_agent","userId":"test_user","lastUpdateTime":...,"events":[],"state":{}}`.

**Step C (Success):** send the message again, targeting the session you just created:

```bash
curl -X POST http://localhost:9091/api/run_sse \
     -H "Content-Type: application/json" \
     -d '{
           "appName": "support_analyzer_agent",
           "userId": "test_user",
           "sessionId": "test_session",
           "newMessage": {"role": "user", "parts": [{"text": "I am so happy with your service!"}]}
         }'
```

Verify you receive a `data:` SSE event whose `actions.stateDelta.last_ticket_analysis` contains the structured JSON analysis (`category`, `sentiment`, `summary`) — this repo's real, confirmed response for this exact input was `{"category": "general", "sentiment": "positive", "summary": "The customer reached out to provide feedback regarding their overall service experience."}`.

## Self-Reflection Questions
- In what scenarios would the detailed Trace View in the Dev UI be more useful than the simple chat interface of `console` mode?
- The `curl` commands above are a simple example of a programmatic client. What kind of real-world applications could you build that would interact with your agent's API this way?
- Why is it necessary to run the REST API server and the `curl` commands in two separate terminal windows? What does this separation represent in a real-world application architecture?
- This lab's own curl commands had to change from Python's (camelCase fields, agent-name instead of project-name). What does that suggest about testing a "port" against the real running server, rather than assuming a 1:1 translation of another language's example commands?

<hr/>

### Looking for the solution?

Hint: read `cmd/support-analyzer/main.go`'s launcher composition (`web.NewLauncher(webui.NewLauncher(), api.NewLauncher())`) — that's the entire mechanism. There's no new agent code in this module; it's entirely about how the same agent from module 4 gets run.
