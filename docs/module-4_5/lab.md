# Lab 4.5 Challenge: Production-Ready Support Analyzer (Go) 🛡️

## Goal

Upgrade the Module 4 Support Analyzer **in place** with retry resiliency, keeping its structured-output contract (`SupportAnalysis`, `OutputSchema`, `OutputKey`) fully working.

## Lab Tasks

1. **This module touches zero files in `cmd/support-analyzer`.** The retry policy lives in the one place every module already shares: `internal/infrastructure/llm`. `cmd/support-analyzer` inherits it automatically the next time it runs with `MODEL_TYPE=gemini` — this is the whole point of centralizing model configuration from module-2 onward. 🎯
2. **Read `internal/infrastructure/llm/resiliency.go`.** `productionRetryOptions()` returns a `*genai.HTTPRetryOptions` with `MaxDelay: 10s, ExpBase: 2.0, Jitter: 0.5`, plus an explicit `HTTPStatusCodes` list (408/429/500/502/503/504 retryable, every other 4xx not).
3. **Read `internal/infrastructure/llm/factory.go`'s `geminiClientConfig`.** This is where centralizing production config actually happens: one function, called by `newGeminiModel`, itself called by every module through `llm.BuildModel`.
4. **The local-vs-cloud switch already exists.** `MODEL_TYPE=ollama` (default) or `MODEL_TYPE=gemini`, read by `internal/infrastructure/llm.LoadConfig()` since module-2. Setting `MODEL_TYPE=gemini` and providing `GOOGLE_AI_STUDIO_API_KEY` gets you the resilient Gemini path this module adds; leaving it unset keeps the local-first Ollama default, unaffected by this module's changes.
5. **Verify the configuration (unit-level, no live Gemini call required):**
   ```bash
   go test ./internal/infrastructure/llm/... -v
   ```
   `TestGeminiClientConfig_UsesProductionRetryOptions` proves `geminiClientConfig` wires the retry policy in; `TestRetryableStatusCodes` proves each status code in the table above is classified correctly. Neither needs a `GOOGLE_AI_STUDIO_API_KEY` or a network call — this is a config-construction guarantee, not a live-retry demonstration (the Python lab makes the same point: *"you won't 'see' the retries unless a network error occurs"*).
6. **Confirm no regression in the Module 4 contract:**
   ```bash
   go test ./cmd/support-analyzer/... -v
   ```
   Passes unchanged — `cmd/support-analyzer` never touches Gemini client construction directly, so this module's change is invisible to it except when `MODEL_TYPE=gemini` is actually selected. ✅

## Self-Reflection Questions 🤔
- Why is "Jitter" important in a retry policy for a high-traffic production application? (Same question as the Python lab — the answer doesn't change with the language.)
- Python's Level 3 centralizes config via subclassing. Go has no subclassing. What does this repo use instead, and where did that pattern first appear in this repo's history?
- Why does an explicit, data-driven status-code table (this module's `retryableStatusCodes`) beat trusting an SDK's implicit default, even when the implicit default happens to be reasonable?
- This repo never needed a LiteLLM equivalent. What earlier module's decision made that true, and what would have had to exist instead if it hadn't?

<hr/>

### Looking for the solution? 🔍

Hint: read `internal/infrastructure/llm/resiliency.go` and `factory.go`'s `geminiClientConfig`/`newGeminiModel`, then `resiliency_test.go` and `factory_test.go`'s `TestGeminiClientConfig_UsesProductionRetryOptions` — that's the whole mechanism, end to end. There's no `cmd/` file to read for this module; that's the point.
