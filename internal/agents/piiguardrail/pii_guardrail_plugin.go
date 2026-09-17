package piiguardrail

import (
	"fmt"
	"regexp"
	"sync"

	"google.golang.org/adk/v2/agent"
	"google.golang.org/adk/v2/plugin"
	"google.golang.org/adk/v2/session"
)

// safetyMessage replaces a blocked response's own text — the canned,
// policy-safe reply the user sees instead of the real (leaked) content,
// matching Python's own lab's replacement text in spirit.
const safetyMessage = "I'm sorry, but I can't share that information."

// creditCardPattern matches a plain XXXX-XXXX-XXXX-XXXX credit-card-shaped
// string — the same simple regex Python's own lab uses; a real production
// guardrail would layer on more sophisticated PII detection, but the
// mechanism this module teaches (intercept, evaluate, block) doesn't
// depend on the detector's own sophistication.
var creditCardPattern = regexp.MustCompile(`\b\d{4}-\d{4}-\d{4}-\d{4}\b`)

// piiGuardrail implements the Fail-Closed pattern: every final-response
// event is checked against creditCardPattern before it reaches the user,
// and blocked (its text overwritten) if it matches. mu guards
// blockedCount for the same reason module-25's alertTracker needed a
// mutex — this plugin is shared for the whole process's lifetime once
// registered, and the SDK may invoke callbacks concurrently.
type piiGuardrail struct {
	mu           sync.Mutex
	pattern      *regexp.Regexp
	blockedCount int
}

// onEvent mutates the matching part's Text in place and returns the same
// pointer — confirmed live (module-25.5's own Phase 1 spec) that
// runner.go's own fromPlugin function supports exactly this idiom, the
// direct equivalent of Python's own event.content.parts[0].text = "...".
// Only a final response with real text is ever inspected; every other
// event (intermediate steps, tool calls, empty content) passes through
// completely unchanged.
//
// Every part is checked, not just Parts[0]: the default Ollama model is
// thinking-capable, and a response genuinely splits into multiple parts —
// the reasoning trace lands in an earlier part with Thought: true, the
// real answer in a later, non-thought part (confirmed live building this
// module: Parts[0] held the reasoning, Parts[1] the actual leaked card
// number — a plugin that only inspected Parts[0] would block the
// harmless reasoning text and let the real leak straight through). This
// is the same "don't assume Parts[0] is the final answer" finding
// module-3's own README already documents for reading events; it applies
// exactly as much to a plugin writing one.
func (g *piiGuardrail) onEvent(_ agent.InvocationContext, event *session.Event) (*session.Event, error) {
	if !event.IsFinalResponse() || event.Content == nil {
		return event, nil
	}

	var blocked bool
	for _, part := range event.Content.Parts {
		if part.Thought || part.Text == "" || !g.pattern.MatchString(part.Text) {
			continue
		}
		part.Text = safetyMessage
		blocked = true
	}
	if !blocked {
		return event, nil
	}

	g.mu.Lock()
	g.blockedCount++
	count := g.blockedCount
	g.mu.Unlock()

	fmt.Printf("🛑 [SAFETY] Response blocked due to policy violation (invocation %q, block #%d)\n", event.InvocationID, count)

	return event, nil
}

// newPIIGuardrailPlugin builds a *plugin.Plugin around a fresh
// piiGuardrail, registered at the Runner/launcher level via PluginConfig —
// the same "cross-cutting concern, not agent logic" placement module-25's
// own AlertingPlugin established. The underlying *piiGuardrail is also
// returned so a caller — this package's own live test included — can read
// blockedCount directly instead of parsing console output or model text.
func newPIIGuardrailPlugin(name string) (*plugin.Plugin, *piiGuardrail, error) {
	guard := &piiGuardrail{pattern: creditCardPattern}
	p, err := plugin.New(plugin.Config{
		Name:            name,
		OnEventCallback: guard.onEvent,
	})
	return p, guard, err
}

// NewPIIGuardrailPlugin is newPIIGuardrailPlugin's public entrypoint for
// production callers (see cmd/pii-guardrail/main.go) that only need the
// *plugin.Plugin to register and have no reason to inspect its counter.
func NewPIIGuardrailPlugin(name string) (*plugin.Plugin, error) {
	p, _, err := newPIIGuardrailPlugin(name)
	return p, err
}
