package observability

import (
	"fmt"
	"sync"

	"google.golang.org/adk/v2/agent"
	"google.golang.org/adk/v2/plugin"
	"google.golang.org/adk/v2/session"
	"google.golang.org/adk/v2/tool"
)

// alertEscalationThreshold is the consecutive-error count at which
// alertTracker escalates from a plain alert to a CRITICAL one, matching
// Python's own lab threshold of 3.
const alertEscalationThreshold = 3

// alertTracker counts consecutive tool errors across turns and escalates
// once alertEscalationThreshold is reached, resetting on the next clean
// turn. It's a plain struct with two methods rather than a bespoke
// interface — plugin.Config's callback fields are just function values, so
// bound methods satisfy them directly with no adapter needed.
//
// mu guards both fields: one alertTracker is shared for the lifetime of
// the process once registered via PluginConfig, and two real conditions
// make that concurrent, not just theoretical — (1) the SDK runs a single
// turn's multiple tool calls concurrently by default (confirmed in
// internal/llminternal/base_flow.go), so two parallel risky_operation
// failures in one turn would race on errorCount/hadErrorThisTurn without a
// lock; (2) under the web/api launcher serving concurrent users, every
// session's tool errors and resets hit this same tracker, so the mutex
// also keeps one user's turn from reading a half-updated state written by
// another's. Confirmed race-free with `go test -race`.
type alertTracker struct {
	mu               sync.Mutex
	errorCount       int
	hadErrorThisTurn bool
}

// onToolError marks the current turn as errored, increments the running
// count, logs an alert (escalating to CRITICAL at the threshold), and
// returns a graceful result instead of the error — the direct equivalent
// of Python's own on_tool_error_callback returning a dict "so the agent
// recovers gracefully instead of crashing the run."
func (a *alertTracker) onToolError(_ agent.Context, t tool.Tool, _ map[string]any, err error) (map[string]any, error) {
	a.mu.Lock()
	a.hadErrorThisTurn = true
	a.errorCount++
	count := a.errorCount
	a.mu.Unlock()

	if count >= alertEscalationThreshold {
		fmt.Printf("🚨 CRITICAL ALERT: tool %q has failed %d times consecutively (%v)\n", t.Name(), count, err)
	} else {
		fmt.Printf("⚠️  ALERT: tool %q failed (%d consecutive so far): %v\n", t.Name(), count, err)
	}

	return map[string]any{"status": "error", "message": err.Error()}, nil
}

// onEvent resets the tracker to a clean state once a turn finishes with no
// error — a clean turn means the agent has recovered. Non-final events are
// a no-op.
func (a *alertTracker) onEvent(_ agent.InvocationContext, event *session.Event) (*session.Event, error) {
	if event.IsFinalResponse() {
		a.mu.Lock()
		if !a.hadErrorThisTurn {
			a.errorCount = 0
		}
		a.hadErrorThisTurn = false
		a.mu.Unlock()
	}
	return event, nil
}

// newAlertingPlugin builds a *plugin.Plugin around a fresh alertTracker,
// registered at the Runner/launcher level via PluginConfig — a
// cross-cutting concern kept deliberately separate from BuildRootAgent's
// own per-agent callback slots (llmagent.Config's BeforeToolCallbacks etc.),
// matching the real architectural split the SDK itself enforces between
// agent-level callbacks and Runner-level plugins.
//
// The underlying *alertTracker is also returned (a *plugin.Plugin itself
// exposes no way to reach back into it) so a caller — this package's own
// live test included — can inspect the real error count directly instead
// of parsing console output or model text. Production callers (see
// cmd/observability-agent/main.go) only need the *plugin.Plugin and can
// discard it.
func newAlertingPlugin(name string) (*plugin.Plugin, *alertTracker, error) {
	tracker := &alertTracker{}
	p, err := plugin.New(plugin.Config{
		Name:                name,
		OnToolErrorCallback: tracker.onToolError,
		OnEventCallback:     tracker.onEvent,
	})
	return p, tracker, err
}

// NewAlertingPlugin is newAlertingPlugin's public entrypoint for production
// callers (see cmd/observability-agent/main.go) that only need the
// *plugin.Plugin to register and have no reason to inspect its tracker.
func NewAlertingPlugin(name string) (*plugin.Plugin, error) {
	p, _, err := newAlertingPlugin(name)
	return p, err
}
