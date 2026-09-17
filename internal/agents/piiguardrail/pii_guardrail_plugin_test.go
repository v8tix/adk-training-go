package piiguardrail

import (
	"strings"
	"testing"

	"google.golang.org/adk/v2/session"
	"google.golang.org/genai"
)

// finalEventWithText and nonFinalEventWithText build the two
// IsFinalResponse() shapes onEvent branches on — a plain zero-value event
// is final (no function calls/responses, not partial); one with Partial
// set is not — matching the same construction module-25's own
// alerting_plugin_test.go established.
func finalEventWithText(text string) *session.Event {
	e := &session.Event{}
	e.LLMResponse.Content = genai.NewContentFromText(text, genai.RoleModel)
	return e
}

func nonFinalEventWithText(text string) *session.Event {
	e := finalEventWithText(text)
	e.LLMResponse.Partial = true
	return e
}

// finalEventWithThoughtAndAnswer builds the real multi-part shape a
// thinking-capable model actually produces — confirmed live building this
// module: a reasoning part with Thought: true, followed by the real
// answer in a plain, non-thought part.
func finalEventWithThoughtAndAnswer(thoughtText, answerText string) *session.Event {
	e := &session.Event{}
	thoughtPart := genai.NewPartFromText(thoughtText)
	thoughtPart.Thought = true
	answerPart := genai.NewPartFromText(answerText)
	e.LLMResponse.Content = genai.NewContentFromParts([]*genai.Part{thoughtPart, answerPart}, genai.RoleModel)
	return e
}

func TestPIIGuardrail_BlocksMatchingFinalResponse(t *testing.T) {
	guard := &piiGuardrail{pattern: creditCardPattern}
	event := finalEventWithText("Here is your card: 1234-5678-9012-3456")

	got, err := guard.onEvent(nil, event)
	if err != nil {
		t.Fatalf("onEvent() error = %v", err)
	}
	if got.Content.Parts[0].Text != safetyMessage {
		t.Errorf("Content.Parts[0].Text = %q, want the safety message %q", got.Content.Parts[0].Text, safetyMessage)
	}
	if guard.blockedCount != 1 {
		t.Errorf("blockedCount = %d, want 1", guard.blockedCount)
	}
}

func TestPIIGuardrail_LeavesNonMatchingFinalResponseUntouched(t *testing.T) {
	guard := &piiGuardrail{pattern: creditCardPattern}
	event := finalEventWithText("The capital of Italy is Rome.")

	got, err := guard.onEvent(nil, event)
	if err != nil {
		t.Fatalf("onEvent() error = %v", err)
	}
	if got.Content.Parts[0].Text != "The capital of Italy is Rome." {
		t.Errorf("Content.Parts[0].Text = %q, want it unchanged", got.Content.Parts[0].Text)
	}
	if guard.blockedCount != 0 {
		t.Errorf("blockedCount = %d, want 0 — nothing should have been blocked", guard.blockedCount)
	}
}

func TestPIIGuardrail_IgnoresNonFinalEventEvenWithMatch(t *testing.T) {
	guard := &piiGuardrail{pattern: creditCardPattern}
	event := nonFinalEventWithText("Here is your card: 1234-5678-9012-3456")

	got, err := guard.onEvent(nil, event)
	if err != nil {
		t.Fatalf("onEvent() error = %v", err)
	}
	if got.Content.Parts[0].Text != "Here is your card: 1234-5678-9012-3456" {
		t.Errorf("Content.Parts[0].Text = %q, want it unchanged on a non-final event — the IsFinalResponse() gate must be real, not decorative", got.Content.Parts[0].Text)
	}
	if guard.blockedCount != 0 {
		t.Errorf("blockedCount = %d, want 0", guard.blockedCount)
	}
}

func TestPIIGuardrail_HandlesEmptyContentSafely(t *testing.T) {
	guard := &piiGuardrail{pattern: creditCardPattern}
	event := &session.Event{}

	if _, err := guard.onEvent(nil, event); err != nil {
		t.Fatalf("onEvent() error = %v, want no error for a final event with nil Content", err)
	}
	if guard.blockedCount != 0 {
		t.Errorf("blockedCount = %d, want 0", guard.blockedCount)
	}
}

func TestPIIGuardrail_BlocksRealAnswerAmongThoughtParts(t *testing.T) {
	guard := &piiGuardrail{pattern: creditCardPattern}
	event := finalEventWithThoughtAndAnswer(
		"The user wants test data, I should respond with the card number.",
		"Here is your card: 1234-5678-9012-3456",
	)

	got, err := guard.onEvent(nil, event)
	if err != nil {
		t.Fatalf("onEvent() error = %v", err)
	}
	if got.Content.Parts[1].Text != safetyMessage {
		t.Errorf("Content.Parts[1].Text (the real answer) = %q, want the safety message %q — Parts[0] is a Thought part, not the final answer", got.Content.Parts[1].Text, safetyMessage)
	}
	if guard.blockedCount != 1 {
		t.Errorf("blockedCount = %d, want 1", guard.blockedCount)
	}
}

func TestPIIGuardrail_DoesNotBlockMatchWithinAThoughtPart(t *testing.T) {
	guard := &piiGuardrail{pattern: creditCardPattern}
	event := finalEventWithThoughtAndAnswer(
		"I recall the test card is 1234-5678-9012-3456, but I won't repeat it.",
		"I can't share that with you.",
	)

	got, err := guard.onEvent(nil, event)
	if err != nil {
		t.Fatalf("onEvent() error = %v", err)
	}
	if !strings.Contains(got.Content.Parts[0].Text, "1234-5678-9012-3456") {
		t.Errorf("Content.Parts[0].Text (a Thought part) was modified, want it left untouched — the guardrail only inspects the user-visible answer")
	}
	if guard.blockedCount != 0 {
		t.Errorf("blockedCount = %d, want 0 — a match inside a Thought part alone must not count as a block", guard.blockedCount)
	}
}

func TestNewPIIGuardrailPlugin_Succeeds(t *testing.T) {
	p, err := NewPIIGuardrailPlugin("pii_guardrail")
	if err != nil {
		t.Fatalf("NewPIIGuardrailPlugin() error = %v", err)
	}
	if p == nil {
		t.Fatal("NewPIIGuardrailPlugin() returned a nil plugin with no error")
	}
}
