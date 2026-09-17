package observability

import (
	"errors"
	"testing"

	"google.golang.org/adk/v2/session"
)

// fakeTool is a minimal tool.Tool double — just enough for onToolError's
// own t.Name() call.
type fakeTool struct{ name string }

func (f fakeTool) Name() string        { return f.name }
func (f fakeTool) Description() string { return "" }
func (f fakeTool) IsLongRunning() bool { return false }

// finalEvent and nonFinalEvent build the two IsFinalResponse() shapes
// onEvent branches on: a plain zero-value event is final (no function
// calls/responses, not partial); one with Partial set is not.
func finalEvent() *session.Event { return &session.Event{} }
func nonFinalEvent() *session.Event {
	e := &session.Event{}
	e.LLMResponse.Partial = true
	return e
}

func TestAlertTracker_EscalatesAtThreshold(t *testing.T) {
	tracker := &alertTracker{}
	wantErr := errors.New("boom")

	for i := 1; i <= alertEscalationThreshold; i++ {
		if _, err := tracker.onToolError(nil, fakeTool{name: "risky_operation"}, nil, wantErr); err != nil {
			t.Fatalf("onToolError() call %d error = %v, want nil (a graceful result, not a propagated error)", i, err)
		}
	}

	if tracker.errorCount != alertEscalationThreshold {
		t.Errorf("errorCount = %d, want %d", tracker.errorCount, alertEscalationThreshold)
	}
	if !tracker.hadErrorThisTurn {
		t.Error("hadErrorThisTurn = false, want true after a real tool error")
	}
}

func TestAlertTracker_OnToolError_ReturnsGracefulResult(t *testing.T) {
	tracker := &alertTracker{}
	wantErr := errors.New("boom")

	got, err := tracker.onToolError(nil, fakeTool{name: "risky_operation"}, nil, wantErr)
	if err != nil {
		t.Fatalf("onToolError() error = %v, want nil", err)
	}
	if got["status"] != "error" {
		t.Errorf("result[\"status\"] = %v, want %q", got["status"], "error")
	}
	if got["message"] != wantErr.Error() {
		t.Errorf("result[\"message\"] = %v, want %q", got["message"], wantErr.Error())
	}
}

func TestAlertTracker_ResetsOnCleanFinalTurn(t *testing.T) {
	tracker := &alertTracker{errorCount: 2, hadErrorThisTurn: false}

	if _, err := tracker.onEvent(nil, finalEvent()); err != nil {
		t.Fatalf("onEvent() error = %v", err)
	}

	if tracker.errorCount != 0 {
		t.Errorf("errorCount = %d, want 0 after a clean final turn", tracker.errorCount)
	}
}

func TestAlertTracker_DoesNotResetAfterAnErroredTurn(t *testing.T) {
	tracker := &alertTracker{}
	if _, err := tracker.onToolError(nil, fakeTool{name: "risky_operation"}, nil, errors.New("boom")); err != nil {
		t.Fatalf("onToolError() error = %v", err)
	}

	if _, err := tracker.onEvent(nil, finalEvent()); err != nil {
		t.Fatalf("onEvent() error = %v", err)
	}

	if tracker.errorCount != 1 {
		t.Errorf("errorCount = %d, want 1 — a turn that had an error must not reset the count on its own final event", tracker.errorCount)
	}
	if tracker.hadErrorThisTurn {
		t.Error("hadErrorThisTurn = true after a final event, want it cleared for the next turn")
	}
}

func TestNewAlertingPlugin_Succeeds(t *testing.T) {
	p, err := NewAlertingPlugin("alerting_plugin")
	if err != nil {
		t.Fatalf("NewAlertingPlugin() error = %v", err)
	}
	if p == nil {
		t.Fatal("NewAlertingPlugin() returned a nil plugin with no error")
	}
}

func TestAlertTracker_OnEvent_NonFinalIsNoOp(t *testing.T) {
	tracker := &alertTracker{errorCount: 2, hadErrorThisTurn: true}

	if _, err := tracker.onEvent(nil, nonFinalEvent()); err != nil {
		t.Fatalf("onEvent() error = %v", err)
	}

	if tracker.errorCount != 2 {
		t.Errorf("errorCount = %d, want unchanged 2 — a non-final event must not reset anything", tracker.errorCount)
	}
	if !tracker.hadErrorThisTurn {
		t.Error("hadErrorThisTurn = false, want unchanged true — a non-final event must not clear it")
	}
}
