package main

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/chromedp/chromedp"

	"github.com/v8tix/adk-training-go/internal/infrastructure/llm"
)

// agentPort must match static/index.html's own hardcoded API_BASE
// (http://localhost:9093/api) — the lab's own documented invocation uses
// this port, and the static page has no runtime override hook for it (a
// test-only override would mean the production markup carries test-only
// wiring, not worth it for one test). A random free port would silently
// make the browser's fetch calls go nowhere.
const agentPort = "9093"

// repoRoot resolves the repository root via `go env GOMOD`'s own directory
// — cwd-independent, unlike a relative path. Same technique
// internal/agents/mcpcart.RepoRoot() uses, established after module-27's
// own review caught a cwd-relative path silently breaking under `go test`.
func repoRoot(t *testing.T) string {
	t.Helper()
	out, err := exec.Command("go", "env", "GOMOD").Output()
	if err != nil {
		t.Fatalf("go env GOMOD: %v", err)
	}
	return filepath.Dir(strings.TrimSpace(string(out)))
}

// chromeAvailable reports whether a Chrome- or Chromium-family browser is
// installed, mirroring (not calling — it's unexported) chromedp's own
// findExecPath fallback list in allocate.go, so this test can skip cleanly
// instead of failing on a machine without one, the same discipline
// llm.OllamaReachable and npxReachable (module-27) already established for
// other external dependencies.
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

// portFree reports whether addr can be bound right now — used to skip
// cleanly instead of failing confusingly when agentPort is already taken
// by, say, a developer's own manually-running instance of this same
// program.
func portFree(addr string) bool {
	l, err := net.Listen("tcp", addr)
	if err != nil {
		return false
	}
	l.Close()
	return true
}

// buildUIAgentBinary compiles cmd/ui-agent to a real temp binary — not `go
// run` — specifically so t.Cleanup can send it a hard, deterministic Kill()
// at test end. `go run` spawns its own child build process that a Kill()
// of the "go run" wrapper doesn't reliably terminate; building once and
// exec'ing the real binary avoids that class of orphaned-process risk
// entirely.
func buildUIAgentBinary(t *testing.T, repoRoot string) string {
	t.Helper()
	bin := filepath.Join(t.TempDir(), "ui-agent-test-binary")
	cmd := exec.Command("go", "build", "-o", bin, "./cmd/ui-agent")
	cmd.Dir = repoRoot
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("building cmd/ui-agent: %v\n%s", err, out)
	}
	return bin
}

// waitForHealth polls url until it returns 200 or the timeout elapses —
// the agent binary needs a moment to start listening after being launched.
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

// TestChatUI_RendersStreamedAnswerWithoutLeakingThought_Ollama is the
// permanent, full-stack proof this module's lab only verified manually
// during Build: a real browser (headless Chrome, driven by
// github.com/chromedp/chromedp), loading the real static/index.html,
// talking to the real cmd/ui-agent binary launched through the real
// cmd/launcher CLI (not a direct adkrest.NewServer construction — this is
// the one test in this module that exercises the launcher's own /api
// mounting and CORS wiring for real), genuinely renders a streamed answer
// and never renders the model's own thought text.
func TestChatUI_RendersStreamedAnswerWithoutLeakingThought_Ollama(t *testing.T) {
	if !llm.SelectedTestBackend().IncludesOllama() {
		t.Skip("skipping: TEST_BACKEND excludes ollama")
	}
	if !llm.OllamaReachable(testConfig, 2*time.Second) {
		t.Skip("skipping: local Ollama server (" + testConfig.OllamaBaseURL + ") is not reachable")
	}
	if !chromeAvailable() {
		t.Skip("skipping: no Chrome/Chromium install found — see README.md's Tooling section to install one")
	}
	if !portFree("localhost:" + agentPort) {
		t.Skipf("skipping: port %s is already in use (static/index.html hardcodes this port, so the test can't use a different one)", agentPort)
	}

	root := repoRoot(t)

	// Serve the real static/index.html in-process — equivalent behavior to
	// cmd/ui-client-server, without a second subprocess to manage.
	staticDir := filepath.Join(root, "cmd", "ui-client-server", "static")
	staticServer := httptest.NewServer(http.FileServer(http.Dir(staticDir)))
	t.Cleanup(staticServer.Close)

	// Launch the REAL cmd/ui-agent binary through its own CLI, with
	// -webui_address pointed at the static server's own origin — this is
	// what actually exercises the launcher's CORS wiring; a browser will
	// reject the client's cross-origin fetch calls without it.
	bin := buildUIAgentBinary(t, root)
	agentCmd := exec.Command(bin,
		"web", "--port="+agentPort,
		"api", "-webui_address", staticServer.URL,
	)
	agentCmd.Dir = root
	agentCmd.Stderr = os.Stderr
	if err := agentCmd.Start(); err != nil {
		t.Fatalf("starting cmd/ui-agent: %v", err)
	}
	t.Cleanup(func() {
		if agentCmd.Process != nil {
			_ = agentCmd.Process.Kill()
			_ = agentCmd.Wait()
		}
	})

	waitForHealth(t, "http://localhost:"+agentPort+"/health", 15*time.Second)

	ctx, cancel := chromedp.NewContext(context.Background())
	defer cancel()
	ctx, cancelTimeout := context.WithTimeout(ctx, 45*time.Second)
	defer cancelTimeout()

	var assistantText string
	err := chromedp.Run(ctx,
		chromedp.Navigate(staticServer.URL),
		chromedp.WaitVisible(`#message-input`, chromedp.ByID),
		chromedp.SendKeys(`#message-input`, "Say hello in exactly three words.", chromedp.ByID),
		chromedp.Click(`#input-form button[type="submit"]`, chromedp.ByQuery),
		chromedp.Poll(
			`(() => {
				const msgs = document.querySelectorAll('.message.assistant');
				const last = msgs[msgs.length - 1];
				if (!last) return null;
				const t = last.textContent;
				return (t && t !== '...' && t.length > 0) ? t : null;
			})()`,
			&assistantText,
			chromedp.WithPollingTimeout(30*time.Second),
		),
	)
	if err != nil {
		t.Fatalf("chromedp run failed: %v", err)
	}
	if assistantText == "" {
		t.Fatal("no assistant text rendered in the DOM")
	}

	lower := strings.ToLower(assistantText)
	suspiciousMarkers := []string{"let me think", "the user wants", "let me count", "chain of thought"}
	for _, marker := range suspiciousMarkers {
		if strings.Contains(lower, marker) {
			t.Errorf("rendered assistant text = %q, contains a likely reasoning-trace marker %q — the client's thought filtering may have regressed", assistantText, marker)
		}
	}
}
