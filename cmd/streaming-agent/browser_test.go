package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/cdproto/dom"
	"github.com/chromedp/cdproto/input"
	"github.com/chromedp/chromedp"
)

// agentPort must match main.go's own hardcoded addr (":9095") — the static
// client's own WS_BASE derives from window.location, so this is the only
// place the port needs to be kept in sync.
const agentPort = "9095"

// repoRoot resolves the repository root via `go env GOMOD`'s own directory —
// cwd-independent, same technique module-29's browser_test.go established.
func repoRoot(t *testing.T) string {
	t.Helper()
	out, err := exec.Command("go", "env", "GOMOD").Output()
	if err != nil {
		t.Fatalf("go env GOMOD: %v", err)
	}
	return filepath.Dir(strings.TrimSpace(string(out)))
}

// chromeAvailable reports whether a Chrome- or Chromium-family browser is
// installed — mirrors module-29's own copy exactly (each cmd/ program's
// browser_test.go is self-contained; there is no shared internal package
// for this across two `package main`s).
func chromeAvailable() bool {
	names := []string{
		"google-chrome", "google-chrome-stable", "chromium", "chromium-browser", "chrome",
	}
	for _, name := range names {
		if _, err := exec.LookPath(name); err == nil {
			return true
		}
	}
	if runtime.GOOS == "darwin" {
		paths := []string{
			"/Applications/Google Chrome.app/Contents/MacOS/Google Chrome",
			"/Applications/Chromium.app/Contents/MacOS/Chromium",
		}
		for _, p := range paths {
			if _, err := os.Stat(p); err == nil {
				return true
			}
		}
	}
	return false
}

// portFree reports whether addr can be bound right now.
func portFree(addr string) bool {
	l, err := net.Listen("tcp", addr)
	if err != nil {
		return false
	}
	l.Close()
	return true
}

// buildStreamingAgentBinary compiles cmd/streaming-agent to a real temp
// binary — not `go run` — so t.Cleanup can send it a deterministic Kill().
// See module-29's own buildUIAgentBinary for why `go run` doesn't work here.
func buildStreamingAgentBinary(t *testing.T, repoRoot string) string {
	t.Helper()
	bin := filepath.Join(t.TempDir(), "streaming-agent-test-binary")
	cmd := exec.Command("go", "build", "-o", bin, "./cmd/streaming-agent")
	cmd.Dir = repoRoot
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("building cmd/streaming-agent: %v\n%s", err, out)
	}
	return bin
}

// waitForHealth polls url until it returns 200 or the timeout elapses.
func waitForHealth(t *testing.T, url string, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		resp, err := http.Get(url)
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return
			}
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatalf("%s never became healthy within %s", url, timeout)
}

// pressAndHold dispatches real, CDP-level mouse events (Input.dispatchMouseEvent)
// at the center of sel's node — mousedown, a hold, then mouseup — rather than
// a synthetic element.dispatchEvent(new MouseEvent(...)) call. Confirmed live
// this distinction matters here: a JS-dispatched synthetic event does not
// count as a user gesture, and startTalking()'s own AudioContext silently
// never produces data without one; a real CDP-level event does.
func pressAndHold(sel string, hold time.Duration) chromedp.Action {
	return chromedp.ActionFunc(func(ctx context.Context) error {
		var nodes []*cdp.Node
		if err := chromedp.Nodes(sel, &nodes, chromedp.ByID).Do(ctx); err != nil {
			return err
		}
		if len(nodes) == 0 {
			return fmt.Errorf("node not found: %s", sel)
		}
		n := nodes[0]
		if err := dom.ScrollIntoViewIfNeeded().WithNodeID(n.NodeID).Do(ctx); err != nil {
			return err
		}
		boxes, err := dom.GetContentQuads().WithNodeID(n.NodeID).Do(ctx)
		if err != nil {
			return err
		}
		if len(boxes) == 0 {
			return fmt.Errorf("no content quads for %s", sel)
		}
		content := boxes[0]
		var x, y float64
		for i := 0; i < len(content); i += 2 {
			x += content[i]
			y += content[i+1]
		}
		x /= float64(len(content) / 2)
		y /= float64(len(content) / 2)

		if err := input.DispatchMouseEvent(input.MousePressed, x, y).
			WithButton(input.Left).WithClickCount(1).Do(ctx); err != nil {
			return err
		}
		if err := chromedp.Sleep(hold).Do(ctx); err != nil {
			return err
		}
		return input.DispatchMouseEvent(input.MouseReleased, x, y).
			WithButton(input.Left).WithClickCount(1).Do(ctx)
	})
}

// TestVoiceClient_ConnectsAndSendsRealAudio is the permanent, full-stack
// proof of this module's own client-side lesson: a real browser (headless
// Chrome, via chromedp, with Chrome's fake-audio-device flags standing in
// for a real microphone) loads the real static/index.html — served by the
// real cmd/streaming-agent binary itself, same origin as the WebSocket it
// talks to (see main.go's own doc comment for why that's required, not
// stylistic) — connects a real WebSocket to /run_live, and genuinely
// captures and streams real binary PCM frames over it via a real
// (CDP-level, not synthetic) press-and-hold gesture on the talk button.
//
// This test does NOT assert a real spoken reply comes back, unlike
// cmd/streaming-agent/main_test.go's own
// TestRunLive_BinaryAudioTurnRequiresSilenceTailNotActivityEnd. Confirmed
// live, twice, during Build: Chrome's Web Audio API pipeline delivers only
// silence from a fake capture device in headless mode — verified both with
// the default synthetic device and with a real speech WAV fed via
// --use-file-for-fake-audio-capture (this test's own PCM samples measured
// exactly zero amplitude throughout in every case tried). Real speech never
// actually reaches the server in a headless run, so the server's own
// automatic voice-activity detection has nothing to detect and never
// responds — correct behavior on the server's part, not a bug this test
// should paper over by asserting on it anyway. The real audio-response
// round trip is proven instead by the Go-side test named above, which
// writes genuine non-silent PCM bytes straight over the wire.
//
// Requires real Vertex AI Live credentials — Application Default
// Credentials or an Express Mode API key — exported into the shell (not
// just present in .env, since this test's own skip check reads them via
// llm.LoadConfig() before any subprocess runs; the launched binary itself
// calls godotenv.Load() and would pick up a real .env regardless, but the
// skip check needs to already know the answer). Run with, e.g.:
//
//	set -a && source .env && set +a && go test ./cmd/streaming-agent/... -run TestVoiceClient
func TestVoiceClient_ConnectsAndSendsRealAudio(t *testing.T) {
	skipIfNoVertexAILive(t)
	if !chromeAvailable() {
		t.Skip("skipping: no Chrome/Chromium install found — see README.md's Tooling section to install one")
	}
	if !portFree("localhost:" + agentPort) {
		t.Skipf("skipping: port %s is already in use (index.html derives its own WebSocket URL from window.location, so the test can't use a different port)", agentPort)
	}

	root := repoRoot(t)

	bin := buildStreamingAgentBinary(t, root)
	agentCmd := exec.Command(bin)
	agentCmd.Dir = root
	agentCmd.Stderr = os.Stderr
	if err := agentCmd.Start(); err != nil {
		t.Fatalf("starting cmd/streaming-agent: %v", err)
	}
	t.Cleanup(func() {
		if agentCmd.Process != nil {
			_ = agentCmd.Process.Kill()
			_ = agentCmd.Wait()
		}
	})

	agentURL := "http://localhost:" + agentPort
	waitForHealth(t, agentURL+"/health", 15*time.Second)

	// use-fake-device-for-media-stream/use-fake-ui-for-media-stream stand in
	// for a real microphone and auto-grant the getUserMedia permission
	// prompt. headless=old (overriding DefaultExecAllocatorOptions' own
	// plain "headless") is required, not optional: confirmed live, Chrome's
	// newer "headless=new" mode never resolves the recorder's own
	// getUserMedia() call at all under these fake-device flags — it hangs
	// forever, neither resolving nor rejecting — while legacy headless
	// resolves it correctly.
	allocOpts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("use-fake-device-for-media-stream", true),
		chromedp.Flag("use-fake-ui-for-media-stream", true),
		chromedp.Flag("headless", "old"),
	)
	allocCtx, cancelAlloc := chromedp.NewExecAllocator(context.Background(), allocOpts...)
	defer cancelAlloc()

	ctx, cancel := chromedp.NewContext(allocCtx)
	defer cancel()
	ctx, cancelTimeout := context.WithTimeout(ctx, 45*time.Second)
	defer cancelTimeout()

	var wsConnected bool
	var audioBytesSent float64
	err := chromedp.Run(ctx,
		chromedp.Navigate(agentURL),
		chromedp.WaitVisible(`#talk-button`, chromedp.ByID),
		chromedp.Poll(`window.__wsConnected === true`, &wsConnected,
			chromedp.WithPollingTimeout(15*time.Second)),
		pressAndHold("talk-button", 4*time.Second),
		chromedp.Evaluate(`window.__audioBytesSent`, &audioBytesSent),
	)
	if err != nil {
		t.Fatalf("chromedp run failed: %v", err)
	}
	if !wsConnected {
		t.Error("window.__wsConnected never became true — the client never reported a real WebSocket connection")
	}
	if audioBytesSent == 0 {
		t.Error("window.__audioBytesSent stayed 0 — no real audio frame was captured and sent over the WebSocket")
	}
}
