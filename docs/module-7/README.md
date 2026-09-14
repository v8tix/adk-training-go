# Module 7: Multimodal and Image Processing (Go)

## Theory

### The Building Block: `genai.Part`, Confirmed Identical in Shape to Python's

Python's `types.Part(inline_data=types.Blob(data=image_bytes, mime_type=...))` has a direct, confirmed Go equivalent — `genai.Part.InlineData *Blob`, with a convenience constructor:

```go
imagePart := genai.NewPartFromBytes(imageBytes, "image/jpeg")
msg := genai.NewContentFromParts([]*genai.Part{
    genai.NewPartFromText("What is in this picture?"),
    imagePart,
}, genai.RoleUser)
```

`genai.NewContentFromParts` is Go's equivalent of Python's `types.Content(role=..., parts=[...])` — both build the same multi-part message shape.

### A Real, Confirmed Local-Inference Gap — Precisely Scoped, Not Overstated

This module found something genuinely worth being precise about, confirmed with two separate live tests, not assumed either direction:

1. **The model server can see images just fine.** Sending the same real photo directly to Ollama's OpenAI-compatible endpoint (`curl .../v1/chat/completions` with an `image_url` content part) returns an accurate description — the model itself has real vision capability.
2. **This repo's Go client for that backend cannot send one.** `google.golang.org/adk/v2/model/openaimodel`'s request-builder (`convertContents`) only has cases for text, function-call, and function-response parts — an image `Part` falls through to `default: return nil, fmt.Errorf("openai: unsupported content part %T", part)`. Confirmed live: sending the identical image through this repo's own `llmagent`+`runner` code (not curl) fails immediately with exactly that error, regardless of which model is loaded.

**The precise, correct claim is: this SDK's local-model client can't send images yet — not that Ollama or the underlying model can't receive them.** `cmd/visual-catalog` therefore requires `MODEL_TYPE=gemini`, hardcoded in its own `main()` rather than left to the shared `.env` default, so it can never silently hit this confirmed error path.

### Why Gemini via `GOOGLE_AI_STUDIO_API_KEY`, Not Vertex AI

Python's lab asks for a Vertex AI setup (`GOOGLE_GENAI_USE_VERTEXAI=1`, `GOOGLE_CLOUD_PROJECT`, `GOOGLE_CLOUD_LOCATION`). This repo's existing Gemini path — the same `internal/infrastructure/llm.BuildModel` every module since module-2 already uses, authenticated with `GOOGLE_AI_STUDIO_API_KEY` — was tested directly with a real image and returned an accurate description with no Vertex-specific configuration at all. No new backend, no new env vars, no new setup for this module.

### The Explicit-Session Lesson, Confirmed Live

Every prior module's `cmd/` program used `runner.NewInMemory`, which sets `AutoCreateSession: true` — sessions appear automatically the first time you `Run` against a new `sessionID`. This module's `cmd/visual-catalog` uses `runner.New(Config{...})` directly instead, where `AutoCreateSession` defaults to `false`. Confirmed by probing both states:

```go
// AutoCreateSession left false, no session created first:
// Run(...) → "session not found: \"s1\""

// After sessionService.Create(ctx, &session.CreateRequest{AppName, UserID, SessionID}):
// Run(...) → succeeds
```

Exactly Python's lesson — `run_async` needs an explicit `create_session` call; `run_debug` (module-6) hid that step for you.

> **Going Further:** `cmd/visual-catalog-local` proves the local-inference-gap claim above from the other side, live — it sends the exact same two product photos directly to the local Ollama server, bypassing `llmagent`/`runner` entirely, and gets back correct descriptions. This is **not** a second way to do this module's real lesson (that's still `cmd/visual-catalog` — the point is that path *needs* Gemini because of a confirmed SDK gap) — it's a bonus demonstrating that the gap is specifically in `model/openaimodel`, this repo's ADK Go SDK client for the local backend, not in Ollama or the model. It uses [kawa](https://github.com/v8tix/kawa) (a typed HTTP-call library with built-in retry) as its HTTP layer instead of the ADK SDK, since this path is explicitly outside the framework this course teaches. See `internal/agents/visualcatalog/local_vision.go` and `cmd/visual-catalog-local/main.go`.

### Key Takeaways
- `genai.NewPartFromBytes`/`NewContentFromParts` are Go's direct, confirmed equivalents of Python's `types.Part`/`types.Content` for multimodal input.
- The local Ollama backend's model server can see images (confirmed via a direct API call); this SDK's Go client for that backend currently can't send them (confirmed via the exact source location and error). Keep these two facts separate — conflating them either direction would be wrong.
- Gemini via the existing `GOOGLE_AI_STUDIO_API_KEY` path handles vision correctly, with no Vertex AI setup needed.
- `runner.New` (unlike `runner.NewInMemory`) requires an explicit `session.Service.Create` call before the first `Run` against a new session — confirmed by reproducing the failure and the fix live.
- Bonus, outside the ADK lesson: `cmd/visual-catalog-local` confirms the local model server itself handles the same images correctly, entirely locally and for free, by calling Ollama directly — the gap really is in this SDK's client, not the backend it talks to.
