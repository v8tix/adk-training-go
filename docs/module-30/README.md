# Module 30: Building a Custom Streaming Client (Go) 🎙️🔊

## Theory

### From Text Streaming to Bidirectional Audio

Module 29's `/run_sse` streams *text* one way: the client sends one message, the server streams the reply back over one HTTP response. This module's `/run_live` is a different shape entirely — a genuine WebSocket, open for the life of a conversation, with audio flowing in *and* out at the same time. The client can be mid-recording while the model is still speaking its previous reply. That's what "bidirectional" buys you, and it's why this module needs a new transport, not just a new endpoint.

### `runner.RunLive` Is Real — But Gemini-Only, Enforced at the Interface Level

`google.golang.org/adk/v2/runner.Runner.RunLive` and its REST wrapper, `RunLiveHandler` (`server/adkrest/controllers/runtime.go`), are fully implemented — this module is not a gap to work around. But `internal/llminternal/base_flow.go`'s own `Flow.RunLive` does this, verbatim:

```go
clientProvider, ok := f.Model.(interface{ Client() *genai.Client })
if !ok {
    return nil, nil, fmt.Errorf("model does not support live connection")
}
```

Only `model/gemini`'s own model type implements that interface. `model/openaimodel` — what every other module in this course runs against Ollama through — does not. This is the first module in the whole series where the local-first default has no seat at the table at all, confirmed by reading the SDK's own source, not inferred from missing docs.

### `VertexAILiveModel`: a Third, Distinct Model Catalog

`internal/infrastructure/llm.Config` already had `GeminiModel` (public Gemini API) and `VertexAIModel` (Vertex's own regular catalog, module-12). This module adds `VertexAILiveModel`, because the Live API draws from a *third*, separate catalog — confirmed live: `VertexAIModel`'s own default (`gemini-2.5-flash`) cannot open a Live connection at all, while `gemini-live-2.5-flash-native-audio` can, on the very same Vertex project and credentials. Same two auth paths as always (Express Mode API key, or Project+Location for ADC) — `newVertexAILiveModel` just reuses `vertexAIClientConfig` with a different model name.

### The Protocol, As Actually Implemented (Not As Assumed)

Every claim below was confirmed against a real, running server — either by reading `RunLiveHandler`'s own source or by a live WebSocket exchange during this module's own Build.

- **Query params**: `appName`/`userId`/`sessionId` (camelCase, snake_case accepted as a fallback).
- **Two ways to send audio**: a raw **binary** WebSocket frame, auto-wrapped server-side as `audio/pcm;rate=16000` with *no* envelope and *no* Base64 — the path this module's own client uses — or a JSON text message shaped `{"blob": {"mime_type": "...", "data": "<base64>"}}`. The binary path is a genuine Go-specific efficiency win: Python's own lab has no such option.
- **A real, confirmed inconsistency**: that JSON `blob` wrapper's own field is `mime_type` (snake_case) — the only snake_case field in an otherwise all-camelCase request shape (`content`, `blob`, `activityStart`, `activityEnd`, `close`). Confirmed in `server/adkrest/internal/models/runtime.go`.
- **Transcription is on by default** — `RunLiveHandler`'s own hardcoded `agent.LiveRunConfig` sets both `InputAudioTranscription` and `OutputAudioTranscription`. A native-audio model's own generated *reply* is audio-only, but the server still hands back real transcript text alongside it — a genuine bonus this Go mirror can show that Python's own lab (no text output at all) can't.

### The Single Most Important Finding: Automatic Activity Detection Owns Turn-Ending, Not `activityEnd`

`genai.ActivityEnd{}` is a real, documented mechanism for manually marking the end of a spoken turn — and it is exactly what this module's client used in its first draft, on button release. It was wrong, and only live testing caught it.

`RunLiveHandler` never sets `agent.LiveRunConfig.RealtimeInputConfig`, so the Live API stays in its own default mode: **automatic activity detection**, where the *server* watches the raw audio stream itself and decides where speech ends. `activityEnd` only has defined behavior once automatic detection is explicitly turned off — a knob `RunLiveHandler` gives callers no way to reach from outside the SDK. Confirmed live, twice, with a real Vertex AI Live connection and the exact same real speech audio:

| Client behavior on release | Result |
| :--- | :--- |
| Send `{"activityEnd": {}}` | **Nothing.** No response, no error, silently, indefinitely. |
| Send ~1s of real trailing silence instead | A complete, real response — transcribed input, transcribed output, real streamed audio. |

Both are "correct" client code by the API's own documented spec. Only one is trusted by *this server's own hardcoded configuration* — and there was no way to know which without actually sending real audio and watching what came back. This module's client (`sendSilenceTail`, in `cmd/streaming-agent/static/index.html`) sends trailing silence, not `activityEnd`.

### A Same-Origin Requirement, Confirmed Live

`RunLiveHandler` builds its own `websocket.Upgrader{ReadBufferSize: 1024, WriteBufferSize: 1024}` with no `CheckOrigin` override, so gorilla/websocket falls back to its own default: reject the upgrade unless the request's `Origin` matches its `Host`. A real browser always sends `Origin`. This means **no CORS header, `-webui_address` setting, or anything else can make a cross-origin browser client work against `/run_live`** — the check happens before any of that is even consulted. Module 29's own two-process split (a separate agent server and a separate static-file server) is fundamentally incompatible with this endpoint. This module's `cmd/streaming-agent` serves its own static client from the same `net/http` server it registers `/run_live` on — one process, one origin, by construction — the same shape the SDK's own reference example (`google.golang.org/adk/v2/examples/bidi`) uses, for the same reason.

A second, independent, and — once you avoid the launcher for this reason — moot gotcha, confirmed along the way: the launcher's default `/api` path-prefix mount (`cmd/launcher/web/api/api.go`) wraps every request in its own `redirectRewriter`, which embeds `http.ResponseWriter` as an interface field and so does *not* implement `http.Hijacker` even though the real underlying writer does. gorilla/websocket's `Upgrade` does a plain `w.(http.Hijacker)` type assertion, which fails under that wrapper — every `/run_live` upgrade through a mounted `/api` prefix 500s before a connection is ever made. `/run_sse` (module 29) was never affected, because SSE only needs `http.Flusher`, which the wrapper does implement.

### Key Takeaways ✅
- `runner.RunLive` is real and fully working — but Gemini-only, checked at the Go interface level, with zero fallback for this course's local-first Ollama default.
- The binary-frame audio-send path is a genuine, Go-specific simplification over Python's Base64-only approach.
- Turn-ending for a real-time audio stream is decided by the server's own automatic activity detection watching for real silence — not by a client-sent `activityEnd` marker, confirmed live by testing both and seeing only one produce a response.
- `/run_live` requires a same-origin client by construction (no override available); a mounted `/api` prefix breaks it entirely for an unrelated reason (a `http.Hijacker` gap in the launcher's own CORS wrapper).

<hr/>

> **Coming from Python?** 🐍 Python's own lab sends audio as Base64-wrapped JSON exclusively and signals end-of-turn with `{"audio_stream_end": true}`. This Go mirror's client sends raw binary frames (simpler and more efficient) and signals nothing explicit at all on release — it just stops streaming real audio and lets a second of real silence do the talking, because that's what this server's own default configuration actually listens for. Python's lab doesn't hit the launcher's `/api`-mount issue because its reference server isn't built the same way; this module's own single-process, same-origin server (mirroring the Go SDK's own official example) sidesteps it by never using that mount for `/run_live` at all.
