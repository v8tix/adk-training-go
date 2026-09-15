# Module 4.5: Professional Model Configuration, Resiliency & Portability (Go)

## Theory

### One Function, Fully Configured

`llmagent.Config.Model` is typed `model.LLM` — an interface, never a bare model-name string — so a Gemini model always goes through `gemini.NewModel(ctx, name, cfg *genai.ClientConfig)`, with `*genai.ClientConfig` as the one place to put every production setting: retries, timeouts, credentials, all of it. Centralizing that configuration across every agent in this repo is just a plain constructor function that returns a fully-configured `model.LLM` — no separate type or subclass needed. This repo already has that function: `internal/infrastructure/llm`'s `newGeminiModel` (via `geminiClientConfig`), called through `llm.BuildModel` by every module so far.

### Confirmed in Source: Retry Options Are a Real, Autonomous Mechanism

`genai.HTTPRetryOptions` (`MaxDelay`, `ExpBase`, `Jitter`, `Attempts`, `InitialDelay`, `HTTPStatusCodes`) isn't just a config value the SDK quietly ignores: `google.golang.org/genai`'s own `doRequest` calls `retryHTTPRequest(req, retryOptions, client.Do)` internally, confirmed by reading the SDK's source. Setting `genai.ClientConfig.HTTPOptions.RetryOptions` genuinely changes how the client behaves on transient failures — a real, autonomous retry mechanism, not a passthrough hint the SDK might or might not honor.

```go
func productionRetryOptions() *genai.HTTPRetryOptions {
    maxDelay := 10.0
    expBase := 2.0
    jitter := 0.5
    return &genai.HTTPRetryOptions{
        MaxDelay:        &maxDelay,
        ExpBase:         &expBase,
        Jitter:          &jitter,
        HTTPStatusCodes: retryableStatusCodes,
    }
}
```

### Being Explicit About Which Errors Are Worth Retrying

Rather than leaving `HTTPStatusCodes` at the SDK's implicit default, this module sets it explicitly, as data:

| Status code(s) | Retried? | Why |
|---|---|---|
| 408 Request Timeout | Yes | Server-side timeout — worth retrying |
| 429 Too Many Requests | Yes | Rate-limited; backoff helps |
| 500 / 502 / 503 / 504 | Yes | Server-side fault, not a request problem |
| Any other 4xx (400, 401, 403, 404, ...) | No | The request itself is wrong — retrying wastes the budget without any chance of succeeding |

Retrying a client error never helps — the request itself is wrong, and retrying just wastes the budget without any chance of succeeding. Being explicit about that boundary, rather than trusting an SDK default you haven't read, is worth the few extra lines.

### Model-Agnostic by Design: One Factory, Any Backend

Swapping model backends without changing any calling code is already solved in this repo, since module-2: `internal/infrastructure/llm.BuildModel`'s factory registry (`MODEL_TYPE=ollama`/`gemini`) is the single place that decision gets made. Every module since has called the same function regardless of backend — this module's own retry work builds directly on that, rather than adding a second abstraction on top of one that already works.

### Beyond the Lab: Retry Parity on the Ollama Path Too

This repo's own `MODEL_TYPE` factory registry means both backends share one call site (`llm.BuildModel`) — leaving one of them without a retry policy would be a real, silent asymmetry, even though the retry policy above is Gemini-specific on its own. `openaimodel.ClientConfig.Options` accepts `openai-go`'s own request options; `ollamaClientConfig` sets `option.WithMaxRetries(4)` and `option.WithMaxRetryDelay(10 * time.Second)` — 5 total attempts (matching `productionRetryOptions`'s `Attempts: 5`) capped at the same 10-second delay ceiling.

**This is parity in attempt count and delay ceiling, not byte-for-byte identical behavior — confirmed by reading `openai-go`'s own retry loop, not assumed:** `option` has no equivalent of `ExpBase`, `Jitter`, or a configurable `HTTPStatusCodes` list. `openai-go` hardcodes its own base-2 exponential backoff with up-to-25%-of-delay jitter, and its own retryable-status set (408, 409, 429, and every 5xx — close to, but not identical to, this repo's explicit table, since 409 isn't in it and openai-go treats *every* 5xx as retryable rather than a named list of six). Both are reasonable production defaults; they just can't be made numerically identical because `genai` exposes more retry knobs than `openai-go` does.

### Key Takeaways
- `*genai.ClientConfig.HTTPOptions.RetryOptions` is a real, autonomous SDK retry mechanism, confirmed by reading the client's own request-handling code — not a passthrough hint.
- Centralizing production config across every agent is just "one constructor function everyone calls" (`newGeminiModel`/`llm.BuildModel`) — no separate type hierarchy needed.
- Explicit, data-driven transient-vs-permanent status-code classification beats trusting an implicit SDK default — this repo sets `HTTPStatusCodes` itself, with each choice justified.
- This repo's `MODEL_TYPE` factory registry already delivers model-agnosticism, unchanged since module-2 — swapping backends never requires touching calling code.
- Both backends now carry a retry policy — `openai-go`'s more limited retry API means "parity" is attempt-count-and-delay-ceiling parity, not identical backoff math.

<hr/>

> **Coming from Python?** Python's lab walks three levels of model configuration (a bare model-name string, a `Gemini` object with retry options, and a `Gemini` subclass centralizing settings across many agents), plus `LiteLlm` for backend-agnosticism. None of that maps 1:1: Go's `llmagent.Config.Model` is always a typed `model.LLM` interface (no string shorthand), Go has no subclassing (so "centralize config" is just one constructor function), and this repo's own `MODEL_TYPE` factory registry already does `LiteLlm`'s job. `HTTPRetryOptions` is the direct equivalent of Python's `retry_options` parameter.
