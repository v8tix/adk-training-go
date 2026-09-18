package main

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"

	"github.com/v8tix/adk-training-go/internal/agents/streamingagent"
	"github.com/v8tix/adk-training-go/internal/infrastructure/llm"
	"google.golang.org/adk/v2/agent"
	"google.golang.org/adk/v2/server/adkrest"
	"google.golang.org/adk/v2/session"
)

// pcmFromWAV extracts the raw PCM payload from a WAV file's own "data"
// chunk, by walking the RIFF chunk list — not by assuming a fixed 44-byte
// canonical header. A real, confirmed bug found in Phase 5 review:
// testdata/hello.wav (produced by macOS's own `afconvert`) inserts a
// 4044-byte "FLLR" filler chunk between "fmt " and "data", pushing the real
// payload to byte 4096. Slicing at a hardcoded 44 silently fed the "data"
// chunk's own 8-byte header (the literal ASCII "data" plus its 4-byte
// little-endian size) into the PCM stream as four bogus samples, spliced
// right before the real speech — harmless to this particular test only by
// luck (VAD tolerates a few garbage samples plus ~4KB of leading silence),
// but wrong, and fragile against any change to how the fixture is
// regenerated.
func pcmFromWAV(wav []byte) ([]byte, error) {
	if len(wav) < 12 || string(wav[0:4]) != "RIFF" || string(wav[8:12]) != "WAVE" {
		return nil, fmt.Errorf("not a RIFF/WAVE file")
	}
	pos := 12
	for pos+8 <= len(wav) {
		id := string(wav[pos : pos+4])
		size := int(binary.LittleEndian.Uint32(wav[pos+4 : pos+8]))
		start := pos + 8
		if start+size > len(wav) {
			return nil, fmt.Errorf("chunk %q size %d overruns file", id, size)
		}
		if id == "data" {
			return wav[start : start+size], nil
		}
		pos = start + size
		if size%2 == 1 {
			pos++ // chunks are word-aligned; skip the pad byte
		}
	}
	return nil, fmt.Errorf("no \"data\" chunk found")
}

var testConfig llm.Config

func TestMain(m *testing.M) {
	testConfig = llm.LoadConfig()
	m.Run()
}

// skipIfNoVertexAILive skips unless one of Vertex AI's two real auth paths
// is configured — mirrors internal/agents/researchassistant's own
// skipIfNoVertexAI, applied to the Live model type this module forces
// (llm.ModelTypeVertexAILive), the only model type runner.RunLive accepts
// at all in this project.
func skipIfNoVertexAILive(t *testing.T) llm.Config {
	t.Helper()
	hasAPIKey := testConfig.VertexAIAPIKey != ""
	hasADC := testConfig.VertexAIProject != "" && testConfig.VertexAILocation != ""
	if !hasAPIKey && !hasADC {
		t.Skip("skipping: neither VERTEX_AI_API_KEY nor VERTEX_AI_PROJECT+VERTEX_AI_LOCATION are set")
	}
	cfg := testConfig
	cfg.ModelType = llm.ModelTypeVertexAILive
	return cfg
}

// newTestServer builds the real adkrest.Server directly (bypassing
// cmd/launcher's own CLI machinery) and wraps it in httptest.NewServer —
// the same pattern cmd/ui-agent/main_test.go established for /run_sse,
// applied here to /run_live.
func newTestServer(t *testing.T, cfg llm.Config) *httptest.Server {
	t.Helper()

	llmModel, _, err := llm.BuildModel(t.Context(), cfg)
	if err != nil {
		t.Fatalf("BuildModel() error = %v", err)
	}
	rootAgent, err := streamingagent.BuildRootAgent(llmModel)
	if err != nil {
		t.Fatalf("BuildRootAgent() error = %v", err)
	}

	srv, err := adkrest.NewServer(adkrest.ServerConfig{
		SessionService:  session.InMemoryService(),
		AgentLoader:     agent.NewSingleLoader(rootAgent),
		SSEWriteTimeout: 60 * time.Second,
	})
	if err != nil {
		t.Fatalf("adkrest.NewServer() error = %v", err)
	}

	ts := httptest.NewServer(srv)
	t.Cleanup(ts.Close)
	return ts
}

// createSession mirrors the JS client's own ensureSession() — a POST to the
// session-management endpoint with an empty JSON body. /run_live has no
// AutoCreateSession path of its own; the session must already exist.
func createSession(t *testing.T, baseURL, appName, userID, sessionID string) {
	t.Helper()
	url := baseURL + "/apps/" + appName + "/users/" + userID + "/sessions/" + sessionID
	resp, err := http.Post(url, "application/json", strings.NewReader("{}"))
	if err != nil {
		t.Fatalf("creating session: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("creating session: status = %d, want %d", resp.StatusCode, http.StatusOK)
	}
}

// dialLive opens a real WebSocket connection to /run_live for the given
// session — shared connection-setup boilerplate between this file's two live
// tests, which otherwise differ in every protocol detail that actually
// matters (text vs. binary audio, extra transcription assertions).
func dialLive(t *testing.T, baseURL, appName, userID, sessionID string) *websocket.Conn {
	t.Helper()
	wsURL := "ws" + strings.TrimPrefix(baseURL, "http") +
		"/run_live?appName=" + appName + "&userId=" + userID + "&sessionId=" + sessionID
	conn, resp, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("dialing %s: %v", wsURL, err)
	}
	if resp != nil {
		defer resp.Body.Close()
	}
	t.Cleanup(func() { conn.Close() })
	return conn
}

// liveEvent is the subset of the server's own Event JSON this test needs —
// camelCase fields per google.golang.org/adk/v2/server/adkrest/internal/
// models/event.go's own json tags. Note inlineData's own mimeType field is
// camelCase here (the response side) — unlike the client-request blob
// wrapper's mime_type (see server/adkrest/internal/models/runtime.go),
// which is the real, confirmed inconsistency this module's docs cover.
type liveEvent struct {
	Content *struct {
		Parts []struct {
			InlineData *struct {
				MIMEType string `json:"mimeType"`
				Data     []byte `json:"data"`
			} `json:"inlineData"`
		} `json:"parts"`
	} `json:"content"`
	TurnComplete bool `json:"turnComplete"`
}

// TestRunLive_ReturnsRealAudioForATextTurn is the real, live proof this
// module's own server-side protocol works end to end: a plain JSON text
// Content message (no synthesized audio needed — a native-audio model's
// *input* isn't restricted to audio, only its output is) sent over a real
// WebSocket to /run_live produces real streamed PCM audio bytes back,
// confirmed against a real Vertex AI Live connection.
//
// Sending Content alone is sufficient to complete the turn: internal/
// llminternal/googlellm/live_connection.go's own SendContent always passes
// TurnComplete: true to the underlying genai session (confirmed by reading
// that source directly) — no separate end-of-turn marker is needed for a
// text turn at all. The binary-audio path is a different story — see
// TestRunLive_BinaryAudioTurnRequiresSilenceTailNotActivityEnd below for the
// real, live-confirmed mechanism that path actually needs.
func TestRunLive_ReturnsRealAudioForATextTurn(t *testing.T) {
	cfg := skipIfNoVertexAILive(t)
	ts := newTestServer(t, cfg)

	const (
		appName   = "streaming_agent"
		userID    = "test_user"
		sessionID = "test_session"
	)
	createSession(t, ts.URL, appName, userID, sessionID)
	conn := dialLive(t, ts.URL, appName, userID, sessionID)

	reqBody, err := json.Marshal(map[string]any{
		"content": map[string]any{
			"role":  "user",
			"parts": []map[string]string{{"text": "Say the word hello."}},
		},
	})
	if err != nil {
		t.Fatalf("marshaling live request: %v", err)
	}
	if err := conn.WriteMessage(websocket.TextMessage, reqBody); err != nil {
		t.Fatalf("sending live request: %v", err)
	}

	if err := conn.SetReadDeadline(time.Now().Add(30 * time.Second)); err != nil {
		t.Fatalf("SetReadDeadline() error = %v", err)
	}

	var totalAudioBytes int
	var sawTurnComplete bool
	for !sawTurnComplete {
		_, data, err := conn.ReadMessage()
		if err != nil {
			t.Fatalf("reading live event (totalAudioBytes so far = %d): %v", totalAudioBytes, err)
		}
		var ev liveEvent
		if err := json.Unmarshal(data, &ev); err != nil {
			t.Fatalf("unmarshaling live event %q: %v", data, err)
		}
		if ev.Content != nil {
			for _, part := range ev.Content.Parts {
				if part.InlineData == nil {
					continue
				}
				if !strings.HasPrefix(part.InlineData.MIMEType, "audio/") {
					t.Errorf("inlineData.mimeType = %q, want an audio/* MIME type", part.InlineData.MIMEType)
				}
				totalAudioBytes += len(part.InlineData.Data)
			}
		}
		if ev.TurnComplete {
			sawTurnComplete = true
		}
	}

	if totalAudioBytes == 0 {
		t.Fatal("no audio bytes received before turnComplete — want at least one real inlineData chunk")
	}
}

// TestRunLive_BinaryAudioTurnRequiresSilenceTailNotActivityEnd is the real,
// live proof of this module's single most important (and least obvious)
// finding: RunLiveHandler never configures agent.LiveRunConfig's
// RealtimeInputConfig, so the Live API stays in its own default automatic
// activity detection mode — the server, not the client, decides where a
// spoken turn ends, by watching the audio stream itself for real silence.
//
// Confirmed live during Build, twice: (1) streaming real speech audio
// (testdata/hello.wav) followed immediately by {"activityEnd": {}} — this
// module's own first draft, and the literal mechanism genai.ActivityEnd{}
// exists for — produced ZERO response and ZERO error, silently, for the
// full length of a 30s read deadline; (2) the exact same real speech audio
// followed instead by ~1s of real trailing silence (no activityEnd sent at
// all) produced a complete, real response: transcribed input, transcribed
// output, and real streamed audio bytes. Both runs are equally "correct"
// client behavior on paper — only one is trusted by this server's own
// hardcoded config, and only live testing surfaced which. Skipped: sending
// an unpaired activityEnd is a firm negative result already captured above
// in prose, not re-asserted as its own test (asserting "nothing happens" is
// indistinguishable from a hung test and no cheap timeout describes the
// server's own quiet 30s+ window)); this test is the positive proof for the
// one path this module's own client (static/index.html's sendSilenceTail)
// actually uses.
func TestRunLive_BinaryAudioTurnRequiresSilenceTailNotActivityEnd(t *testing.T) {
	cfg := skipIfNoVertexAILive(t)
	ts := newTestServer(t, cfg)

	const (
		appName   = "streaming_agent"
		userID    = "test_user"
		sessionID = "test_session_audio"
	)
	createSession(t, ts.URL, appName, userID, sessionID)
	conn := dialLive(t, ts.URL, appName, userID, sessionID)

	wav, err := os.ReadFile("testdata/hello.wav")
	if err != nil {
		t.Fatalf("reading testdata/hello.wav: %v", err)
	}
	pcm, err := pcmFromWAV(wav)
	if err != nil {
		t.Fatalf("parsing testdata/hello.wav: %v", err)
	}

	const (
		sampleRate    = 16000
		bytesPerFrame = 2 // 16-bit mono
		chunkBytes    = 3200
	)
	sendPCM := func(data []byte) {
		t.Helper()
		for i := 0; i < len(data); i += chunkBytes {
			end := min(i+chunkBytes, len(data))
			if err := conn.WriteMessage(websocket.BinaryMessage, data[i:end]); err != nil {
				t.Fatalf("sending audio chunk: %v", err)
			}
		}
	}

	sendPCM(pcm)
	// ~1s of real trailing silence — the mechanism this module's own client
	// actually relies on (see static/index.html's own sendSilenceTail).
	sendPCM(make([]byte, sampleRate*bytesPerFrame))

	if err := conn.SetReadDeadline(time.Now().Add(30 * time.Second)); err != nil {
		t.Fatalf("SetReadDeadline() error = %v", err)
	}

	var totalAudioBytes int
	var sawInputTranscription, sawTurnComplete bool
	for !sawTurnComplete {
		_, data, err := conn.ReadMessage()
		if err != nil {
			t.Fatalf("reading live event (totalAudioBytes so far = %d): %v", totalAudioBytes, err)
		}
		var ev struct {
			liveEvent
			InputTranscription *struct {
				Text string `json:"text"`
			} `json:"inputTranscription"`
		}
		if err := json.Unmarshal(data, &ev); err != nil {
			t.Fatalf("unmarshaling live event %q: %v", data, err)
		}
		if ev.Content != nil {
			for _, part := range ev.Content.Parts {
				if part.InlineData != nil {
					totalAudioBytes += len(part.InlineData.Data)
				}
			}
		}
		if ev.InputTranscription != nil && ev.InputTranscription.Text != "" {
			sawInputTranscription = true
		}
		if ev.TurnComplete {
			sawTurnComplete = true
		}
	}

	if totalAudioBytes == 0 {
		t.Error("no audio bytes received before turnComplete — the silence-tail mechanism should have produced a real spoken response")
	}
	if !sawInputTranscription {
		t.Error("no inputTranscription text received — want the server to have transcribed testdata/hello.wav's real speech")
	}
}
