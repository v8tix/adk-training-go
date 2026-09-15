# Lab 11: Building a "Global Market Analyst" Agent (Go)

## Goal

Build an agent that retrieves live currency exchange rates from a real public REST API, using a tool declared from a spec instead of hand-written.

## Lab Tasks

### 1. Read `internal/infrastructure/openapitool/openapitool.go`

`OperationSpec` describes one REST operation: an ID, a summary, a base URL, a path, and a list of parameters. `NewToolset` turns one or more of these into a working `tool.Toolset`. Notice `operationTool` (the actual tool implementation) is unexported — you only ever work with the spec.

### 2. Read `internal/agents/marketanalyst/agent.go`

`frankfurterSpec` describes the real Frankfurter currency API's `/latest` endpoint. `BuildRootAgent` turns it into a toolset and attaches it via `Toolsets: []tool.Toolset{toolset}` — not the plain `Tools` field modules 9-10 used, since a toolset is one value producing multiple tools, not a single tool.

### 3. Run it — console mode, entirely locally

```bash
go run ./cmd/market-analyst console
```

No `.env`, no API key needed — the Frankfurter API itself requires no authentication either. Real, confirmed output from this exact command (thinking-model reasoning trimmed for readability):

```
💱 market-analyst using qwen3.8:27b

User -> Convert 100 USD to EUR.
Agent -> Based on the latest exchange rate (as of September 14, 2026):
**100 USD = 86.57 EUR**

User -> Convert 500 AUD to XYZ.
Agent -> I'm sorry, but "XYZ" is not a recognized currency code, so I
wasn't able to fetch a rate for that conversion. Could you double-check
the code you meant?
```

A real conversion using the real, current exchange rate, and a graceful explanation when the currency code is invalid — not a crash, not a fabricated rate.

### 4. Read `internal/infrastructure/openapitool/openapitool_test.go`

Pure tests against a local `httptest.Server`, not the live Frankfurter API — real exchange rates change daily, so a test asserting an exact rate would be flaky by tomorrow. These prove the parameter-encoding and response-decoding logic, plus both of `Run`'s two failure paths: a real API error response (structured result) and a genuine network failure (Go error).

### 5. Read `internal/agents/marketanalyst/agent_test.go`

`TestMarketAnalyst_ConvertsCurrency_Ollama` and `_Gemini` — both real, hitting the live API. They assert on the tool's actual `FunctionResponse` structurally (the currency codes present, a positive numeric rate), never an exact pinned value, for the same reason.

## Self-Reflection Questions
- What are the advantages of describing a tool with data (a spec) instead of writing a Go function for it? What do you lose?
- If two different `OperationSpec`s in the same `Toolset` had the same `OperationID`, what do you think would happen? (Check `toolutils.PackTool`'s behavior on a duplicate name.)
- Real REST APIs publish their own OpenAPI specs, often at a predictable URL. If you wanted `openapitool` to build a `Toolset` directly from one of those (instead of a hand-written `OperationSpec`), what would you need to add?
- Why does a `404` from the real API belong in a structured result the model can read, while a DNS failure belongs in a Go `error`?

<hr/>

### Looking for the solution?

Hint: read `internal/infrastructure/openapitool/openapitool.go` (`OperationSpec`, `NewToolset`, `operationTool`) and `internal/agents/marketanalyst/agent.go` (`frankfurterSpec`, `BuildRootAgent`) — that's the whole mechanism, end to end.
