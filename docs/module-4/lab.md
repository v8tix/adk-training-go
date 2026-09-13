# Lab 4 Challenge: Support Analyzer Agent (Go)

## Goal

Build a **Support Analyzer** agent that reads a customer support ticket and returns a structured JSON analysis — `category`, `sentiment`, and a one-sentence `summary` — instead of a plain-text reply, and saves that analysis into session state.

## Lab Tasks

1. No `adk create` scaffolding step in Go, same as module-3 — `cmd/support-analyzer/main.go` already exists in this repo, hand-written directly.
2. Read `cmd/support-analyzer/prompts/support_analyzer_instruction.md` (loaded at startup into the shared `internal/infrastructure/prompts` cache — see `main.go`'s `init()`). Notice it enumerates the exact allowed `category`/`sentiment` values rather than leaving them open-ended.
3. **Structured Output:** `main.go`'s `buildRootAgent` sets `OutputSchema: supportAnalysisSchema` — a hand-built `*genai.Schema` matching the `SupportAnalysis` struct's three fields. There's no Pydantic-style auto-derivation in Go; the schema and the struct are two separate declarations you keep in sync yourself.
4. **Session State:** `OutputKey: "last_ticket_analysis"` tells the SDK to save the agent's JSON reply into `event.Actions.StateDelta["last_ticket_analysis"]` on the final response event.
5. **Why the model matters here:** this repo's shared `OLLAMA_MODEL` default (`qwen3.8:27b`, a GGUF quantization) was chosen specifically because it supports JSON-schema-constrained output — this machine's faster MLX presets don't (confirmed: Ollama returns `501 "structured output is unavailable"` for them). No `.env` change or cloud credentials needed; the default just works.
6. **Run and Verify:**
   ```bash
   go run ./cmd/support-analyzer web --port 8080 webui
   ```
   Open `http://localhost:8080/ui/` and submit *"My screen is completely broken and I'm very angry about it!"* — verify the response is a valid JSON object with `category`, `sentiment`, and `summary`.

   > **Known quirk, not a bug (same as module-3):** the Dev UI and `console` mode render the model's raw response, including chain-of-thought reasoning, before the JSON answer — this is the SDK's own rendering, not filtered the way this module's own code and tests filter it.
7. **Inspect State:** After a few interactions, the JSON saved to `last_ticket_analysis` is what a downstream agent or tool would consume in a multi-agent system — this module doesn't build that consumer, just proves the value lands there.

### Alternative: Console Mode (Bonus, No Python Parallel)

```bash
go run ./cmd/support-analyzer console
```

### Verifying Automatically

```bash
go test ./cmd/support-analyzer/... -v
```

Drives the real agent via `runner` directly (bypassing the launcher's UI rendering), reads `event.Actions.StateDelta["last_ticket_analysis"]` off the event stream, `json.Unmarshal`s it into `SupportAnalysis`, and asserts all three fields are populated for two differently-toned tickets (and that sentiment actually differs between them).

## Self-Reflection Questions
- Why is it better to use `OutputSchema` instead of just asking the model to "respond in JSON" in the instruction text alone?
- The SDK sets the request's JSON-schema constraint but doesn't validate or parse the response itself — what does that mean for how much you can trust `event.Actions.StateDelta[OutputKey]`'s contents without your own `json.Unmarshal` check?
- This module's requirement (structured output) was the reason this repo's shared default model changed to a GGUF quantization. What does that suggest about verifying a new SDK capability against your actual runtime, rather than assuming "the model backend already worked before, so it'll keep working"?
- How could another agent in a future multi-agent system consume the `"last_ticket_analysis"` state value?

<hr/>

### Looking for the solution?

Hint: read `cmd/support-analyzer/main.go` (the `SupportAnalysis` struct, `supportAnalysisSchema`, and `buildRootAgent`), then `support_test.go`'s `analyzeTicket` function — that's the whole mechanism, end to end.
