# Lab 3 Challenge: Build and Run the "Echo" Agent (Go)

## Goal

Your task is to build and run a simple "Echo" agent using the Go ADK SDK.

**The Challenge:** Unlike a standard chatbot, this agent must act like a **parrot**. It should never answer questions or provide information; it must only repeat the user's input exactly as it was received.

### Expected Behavior

| User Input | Agent Response (Correct) | Agent Response (Wrong) |
| :--- | :--- | :--- |
| "Hello!" | "Hello!" | "Hi there, how can I help you?" |
| "What is the capital of France?" | "What is the capital of France?" | "The capital of France is Paris." |
| "12345" | "12345" | "You entered the numbers 1 through 5." |

This exact table is what `cmd/echo-agent/echo_test.go` asserts automatically.

## Lab Tasks

1. `cmd/echo-agent/main.go` already exists in this repo, hand-written directly — that's how every agent program in Go starts.
2. Read `cmd/echo-agent/prompts/echo_instruction.md` (loaded at startup into the shared `internal/infrastructure/prompts` cache — see `main.go`'s `init()` — kept as a plain text file instead of a Go string constant so it's easy to read and edit). Notice how explicit and repetitive it is ("never answer," "echo the question itself," "do not add commentary"). This isn't overkill: the default model is thinking-capable and will try to be "helpful" unless told plainly, more than once, not to.
3. **Instruction Strategy:** If you change the instruction, keep it at least this explicit — a softer instruction risks the model answering instead of echoing.
4. No `.env` configuration is required for the default (local Ollama) path. If you want to test against Gemini instead, copy `.env.example` to `.env` and set `MODEL_TYPE=gemini` plus `GOOGLE_AI_STUDIO_API_KEY`.
5. Run the agent:

   > **Port choice:** this lab uses `9091` instead of the ADK launcher's default of `8080` — `8080` is commonly already occupied by other local dev tools (confirmed on this session's own machine: Docker Desktop's own proxy was already listening on it). If `9091` is also taken on your machine, check with `lsof -i :9091` (macOS/Linux) and pick any other free port instead, adjusting every URL below **and** the `-api_server_address` flag below to match.
   >
   > **A real, confirmed gotcha: `--port` alone is not enough.** `--port` only changes what port the server itself binds to — the Dev UI's frontend learns where to call the API from a *separate* flag, `webui`'s own `-api_server_address`, which defaults to the hardcoded string `http://localhost:8080/api` regardless of `--port`. Skip it and the Dev UI will always try to call port 8080 no matter what port you actually started the server on — confirmed live: `curl .../ui/assets/config/runtime-config.json` returned `{"backendUrl":"http://localhost:8080/api"}` even when the server was started with `--port 9091`. This is why the command below passes `-api_server_address` explicitly, right after the `webui` keyword.

   ```bash
   go run ./cmd/echo-agent web --port 9091 webui -api_server_address http://localhost:9091/api api
   ```
   Note the flag position: `web`'s own flags (like `--port`) go directly after `web`, then each sub-launcher keyword followed by its own flags (`webui`'s `-api_server_address` here). **Both `webui` and `api` are required** — the Dev UI's frontend calls the REST API for everything beyond its static page, so `webui` alone starts a UI that can't actually interact with the agent (confirmed live: `/api/list-apps` 404s without `api` registered too).
6. Open `http://localhost:9091/ui/` and interact with the agent to verify it passes the Expected Behavior table above.

   > **Known quirk, not a bug:** the Dev UI shows the model's raw response, including its chain-of-thought, before or alongside the echoed answer — the launcher's own UI doesn't filter reasoning traces the way this repo's own code does elsewhere. If you see visible reasoning text, that's expected with this course's default model.

### Alternative: Console Mode

```bash
go run ./cmd/echo-agent console
```

A no-browser CLI chat — type input directly, see the agent's response inline. Same reasoning-trace quirk as the web UI applies here too.

### Verifying Automatically

```bash
go test ./cmd/echo-agent/... -v
```

This runs the exact three cases from the Expected Behavior table against the real local model and asserts an exact match — bypassing the launcher's UI rendering (and its reasoning-trace quirk) entirely, the same way `cmd/verify-setup`'s tests do.

## Self-Reflection Questions
- Go's ADK SDK has no YAML-based agent definition — only the programmatic `llmagent.Config` form. What do you gain or lose compared to Python's two options?
- Why does the echo agent's instruction need to be so explicit and repetitive, when a simpler instruction "works" most of the time?
- The Dev UI and console mode both show raw reasoning text from the model. What would you need to change (in this repo's code, not the SDK) to filter that out for end users, the way the automated test already does for verification purposes?

<hr/>

### Looking for the solution?

Hint: read `cmd/echo-agent/prompts/echo_instruction.md` and `main.go`'s `init()` (how it gets into the cache), then `echo_test.go`'s `runEcho`/`firstAnswerText` functions (how the test fetches and uses it) — that's the whole mechanism, end to end.
