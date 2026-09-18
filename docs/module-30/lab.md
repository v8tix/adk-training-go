# Lab 30: Building a Custom Streaming Client (Go) 🎙️🔊

## Goal

Build a real push-to-talk voice client against `/run_live`, backed by a real Gemini Live API connection over Vertex AI — proving the full bidirectional audio round trip, not just reading about it.

### Prerequisites

A configured Vertex AI backend — `VERTEX_AI_API_KEY` (Express Mode) or `VERTEX_AI_PROJECT`+`VERTEX_AI_LOCATION` (Application Default Credentials), same as module 12. There is no local-only path for this module: `runner.RunLive` rejects every model this repo's Ollama default builds, checked at the Go interface level (see this module's own README). The committed browser test also needs Google Chrome or Chromium — see the top-level [README's Tooling section](../../README.md#-tooling) — and skips cleanly if it isn't found.

## Lab Tasks

### 1. Read `internal/infrastructure/llm/config.go` and `factory.go`

`VertexAILiveModel` (default `gemini-live-2.5-flash-native-audio`, env `VERTEX_AI_LIVE_MODEL`) and `ModelTypeVertexAILive` — a third model catalog alongside `GeminiModel`/`VertexAIModel`, reusing the same two Vertex auth paths module 12 built.

### 2. Read `internal/agents/streamingagent/agent.go`

A deliberately trivial, tool-less agent — this module's real lesson is the protocol the client speaks to it, not what the agent itself can do.

### 3. Read `cmd/streaming-agent/main.go`

This is **not** the usual two-`cmd/`-program, launcher-based shape from module 29. Its own doc comment explains why: `RunLiveHandler`'s WebSocket upgrader has no `CheckOrigin` override, so it rejects any cross-origin browser client outright — no CORS setting can fix that, since the check happens before CORS is even consulted. This program instead builds its own minimal `net/http` server (mirroring the SDK's own reference example, `google.golang.org/adk/v2/examples/bidi`), serving its static client and `/run_live` from the very same origin:

```bash
go run ./cmd/streaming-agent
```

Then open `http://localhost:9095` in a browser.

### 4. Read `cmd/streaming-agent/main_test.go`

Two real, live tests, each proving a different part of the protocol against a real Vertex AI Live connection (`adkrest.NewServer` built directly, not through the launcher — same pattern module 29 established):

- `TestRunLive_ReturnsRealAudioForATextTurn` — a plain JSON text turn is enough to get a real spoken (audio) reply back; no synthesized audio needed for this one, since a native-audio model's *input* isn't restricted to audio, only its *output* is.
- `TestRunLive_BinaryAudioTurnRequiresSilenceTailNotActivityEnd` — the module's central finding, proven twice in the test's own prose and once in its actual assertion: streaming `testdata/hello.wav`'s real speech followed by real silence produces a complete real response (transcribed input, transcribed output, real audio) — the mechanism this module's own client actually uses. Sending `{"activityEnd": {}}` instead (documented in the test, not re-asserted as a second test — a true negative result has no clean way to distinguish itself from a hung test) produced nothing at all, live, for the module's entire Build phase.

Run them (they skip cleanly without real Vertex AI credentials):

```bash
set -a && source .env && set +a  # only needed if your credentials live in .env, not the shell
go test ./cmd/streaming-agent/... -run TestRunLive -v
```

Real, confirmed output from this exact command:

```
=== RUN   TestRunLive_ReturnsRealAudioForATextTurn
--- PASS: TestRunLive_ReturnsRealAudioForATextTurn (2.03s)
=== RUN   TestRunLive_BinaryAudioTurnRequiresSilenceTailNotActivityEnd
--- PASS: TestRunLive_BinaryAudioTurnRequiresSilenceTailNotActivityEnd (2.87s)
```

### 5. Read `cmd/streaming-agent/static/index.html`

The real, hand-written push-to-talk client. Four things to notice against Python's own lab version:

- `WS_BASE` derives from `window.location` — same-origin by construction, not a hardcoded port to keep in sync (see Step 3).
- No separate session-creation call: this server's own controller is built with `AutoCreateSession: true`, confirmed live against `runner.Runner.getOrCreateSession` — unlike module 29's `/run_sse` client, which needs that extra step.
- Audio is sent as raw binary WebSocket frames (`sendAudioChunk`), not Base64-wrapped JSON.
- On button release, `sendSilenceTail()` sends ~1s of real silence — not an `{"activityEnd": {}}` marker. See this module's own README for why that's the one that actually works.

`static/js/audio-recorder.js`, `pcm-recorder-processor.js`, `audio-player.js`, and `pcm-player-processor.js` are ported as-is from the SDK's own reference client (`google.golang.org/adk/v2/examples/bidi`) — pure Web Audio API signal processing (Float32↔Int16 conversion, a ring-buffer player), not this module's own lesson.

### 6. Run it yourself 🎙️

```bash
go run ./cmd/streaming-agent
```

Open `http://localhost:9095`, hold the **Hold to Talk** button, say something, and release. You should hear a real spoken reply, and see both your own transcribed words and the model's transcribed reply appear as chat bubbles — a genuine bonus this Go server enables by default that Python's own lab (audio-only output, no text at all) doesn't have.

### 7. Read `cmd/streaming-agent/browser_test.go`

The permanent, full-stack proof for the *client* side, using a real headless Chrome (`github.com/chromedp/chromedp`) with Chrome's fake-audio-device flags standing in for a microphone. It asserts a real WebSocket connects and real binary audio frames are genuinely captured and sent — not that a real reply comes back. Its own doc comment explains why not: confirmed live, twice, Chrome's Web Audio API pipeline delivers only silence from a fake capture device in headless mode, even when that fake device is fed a real speech `.wav` file via `--use-file-for-fake-audio-capture` — the amplitude of every sample captured measured exactly zero. That's a genuine Chromium/headless limitation, not a bug in this module's own code; the real round-trip audio proof lives in Step 4's own Go-side test instead, which writes genuine non-silent bytes straight over the wire.

```bash
set -a && source .env && set +a
go test ./cmd/streaming-agent/... -run TestVoiceClient -v
```

Real, confirmed output from this exact command:

```
=== RUN   TestVoiceClient_ConnectsAndSendsRealAudio
--- PASS: TestVoiceClient_ConnectsAndSendsRealAudio (6.09s)
```

### Checkpoint

- [ ] `go test ./cmd/streaming-agent/... -race` passes (all three live tests, with real Vertex AI Live credentials configured)
- [ ] Manually confirmed: holding the talk button, speaking, and releasing produces a real spoken reply and real transcript bubbles in the browser

## Self-Reflection Questions 🤔
- `RunLiveHandler` gives you no way to disable automatic activity detection from outside the SDK. If it did, would you prefer manual `activityStart`/`activityEnd` control over a silence-tail heuristic for a real push-to-talk product? What would each cost you?
- This module's server bypasses `cmd/launcher` entirely, building its own minimal `net/http` server instead. What exactly would you lose by doing that for a module that *didn't* have `/run_live`'s same-origin constraint forcing the issue?
- `TestVoiceClient_ConnectsAndSendsRealAudio` deliberately does not assert a real spoken reply comes back. Is that a weaker test than one that does? What would you need to change about *how* the audio is fed to the browser to make that assertion meaningful again?

<hr/>

### Looking for the solution? 🔍

Hint: read `internal/agents/streamingagent/agent.go`, `cmd/streaming-agent/main.go`, and `cmd/streaming-agent/static/index.html` for the real mechanism — a trivial agent, a same-origin REST+WebSocket server, and a hand-written push-to-talk client talking to it over `/run_live`.
