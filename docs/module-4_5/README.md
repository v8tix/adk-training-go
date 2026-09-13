# Module 4.5: Professional Model Configuration, Resiliency & Portability (Go)

## Theory

### Why Python Has Three Levels and Go Has One

The Python course walks three levels of model configuration: a bare model-name string, a `Gemini` object with retry options, and a `Gemini` subclass centralizing production settings across many agents. This doesn't map 1:1 onto Go, and it's worth being explicit about why rather than pretending it does:

- **Level 1 (bare string) doesn't exist in Go.** Python's `Agent(model="gemini-3.5-flash", ...)` accepts a string as shorthand. Go's `llmagent.Config.Model` is typed `model.LLM` — an interface, never a string. Every Gemini model in Go already goes through `gemini.NewModel(ctx, name, cfg *genai.ClientConfig)`, so there's no "prototype" code path to graduate from.
- **Levels 2 and 3 collapse into one question in Go: how much of `*genai.ClientConfig` do you fill in?** Python's Level 3 relies on subclassing — a mechanism Go doesn't have. The equivalent "centralize configuration across many agents" pattern in Go is a plain constructor function that returns a fully-configured `model.LLM`, not a type hierarchy. This repo already has that function: `internal/infrastructure/llm`'s `newGeminiModel` (via `geminiClientConfig`). There was never a separate "Level 2" to build here — the production configuration goes straight into the one function every module already calls through `llm.BuildModel`.

### Confirmed in Source: Retry Options Are a Real, Autonomous Mechanism

`genai.HTTPRetryOptions` has the exact fields Python's lab uses — `MaxDelay`, `ExpBase`, `Jitter` (plus `Attempts`, `InitialDelay`, `HTTPStatusCodes`) — and this isn't just a config value the SDK ignores: `google.golang.org/genai`'s own `doRequest` calls `retryHTTPRequest(req, retryOptions, client.Do)` internally, confirmed by reading the SDK's source. Setting `genai.ClientConfig.HTTPOptions.RetryOptions` genuinely changes how the client behaves on transient failures — the same guarantee Python's `retry_options` parameter gives.

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

This mirrors the same reasoning Python's own instructor notes give for retry policies in general: retrying a client error never helps, and being explicit about the boundary (rather than trusting an SDK default you haven't read) is worth the few extra lines.

### No LiteLLM Port — And No Gap to Fill

There is no Go port of Python's `LiteLlm`. That's not a missing piece: LiteLLM's actual job — swap model backends without changing the calling code — is exactly what this repo's `internal/infrastructure/llm.BuildModel` factory registry (`MODEL_TYPE=ollama`/`gemini`) has provided since module-2. Every module since has called the same function regardless of backend; this module doesn't add a second abstraction on top of one that already works.

### Beyond the Lab: Retry Parity on the Ollama Path Too

Python's lab is Gemini-specific, but this repo's own `MODEL_TYPE` factory registry means both backends share one call site (`llm.BuildModel`) — leaving one of them without a retry policy would be a real, silent asymmetry. `openaimodel.ClientConfig.Options` accepts `openai-go`'s own request options; `ollamaClientConfig` sets `option.WithMaxRetries(4)` and `option.WithMaxRetryDelay(10 * time.Second)` — 5 total attempts (matching `productionRetryOptions`'s `Attempts: 5`) capped at the same 10-second delay ceiling.

**This is parity in attempt count and delay ceiling, not byte-for-byte identical behavior — confirmed by reading `openai-go`'s own retry loop, not assumed:** `option` has no equivalent of `ExpBase`, `Jitter`, or a configurable `HTTPStatusCodes` list. `openai-go` hardcodes its own base-2 exponential backoff with up-to-25%-of-delay jitter, and its own retryable-status set (408, 409, 429, and every 5xx — close to, but not identical to, this repo's explicit table, since 409 isn't in it and openai-go treats *every* 5xx as retryable rather than a named list of six). Both are reasonable production defaults; they just can't be made numerically identical because `genai` exposes more retry knobs than `openai-go` does.

### Key Takeaways
- Go's `*genai.ClientConfig` + `HTTPRetryOptions` gives the same retry-resiliency guarantee as Python's `Gemini(retry_options=...)` — confirmed as a real, autonomous SDK mechanism, not a passthrough hint.
- Python's three configuration levels don't map 1:1: Go has no string-shorthand and no subclassing, so "centralizing production config" is just "one constructor function everyone calls," which this repo already had.
- Explicit, data-driven transient-vs-permanent status-code classification beats trusting an implicit SDK default — this repo sets `HTTPStatusCodes` itself, with each choice justified.
- No LiteLLM port is needed: this repo's `MODEL_TYPE` factory registry already delivers model-agnosticism, unchanged since module-2.
- Both backends now carry a retry policy, not just the one the Python lab asked for — `openai-go`'s more limited retry API means "parity" is attempt-count-and-delay-ceiling parity, not identical backoff math.
