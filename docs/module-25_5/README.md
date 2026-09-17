# Module 25.5: Responsible AI (RAI) & Safety Plugins (Go) 🛡️🚫

## Theory

### Fail-Closed: Check Before It Reaches the User

Module 25's Plugin System *observed* an agent's behavior. This module *controls* it. The **Fail-Closed** pattern means every response passes a mandatory safety check before the user ever sees it — intercept the final answer, evaluate it against a policy, and if it violates that policy, block it and substitute a safe reply. You can't rely solely on a model's own internal filters in an enterprise setting; you need a programmatic layer that enforces your own rules regardless of what the model itself decides to say.

### The Same Hook, Used to Rewrite Instead of Just Observe

Module 25's `alertTracker.onEvent` used `OnEventCallback` to *watch* final-response events. This module uses the identical hook to *change* one:

```go
func (g *piiGuardrail) onEvent(_ agent.InvocationContext, event *session.Event) (*session.Event, error) {
	if !event.IsFinalResponse() || event.Content == nil {
		return event, nil
	}
	for _, part := range event.Content.Parts {
		if part.Thought || part.Text == "" || !g.pattern.MatchString(part.Text) {
			continue
		}
		part.Text = safetyMessage
		// ...
	}
	return event, nil
}
```

Mutating `part.Text` in place and returning the same event pointer genuinely changes what the user sees — confirmed by reading `runner.go`'s own `fromPlugin` function, not assumed: its doc comment states plainly, *"a plugin that mutates the event in place and returns nil, or returns the same pointer, is the ordinary way to write this hook."* The mutation reaches every code path that yields events back to the caller.

### A Real Bug This Module's Own Build Caught: Don't Trust `Parts[0]`

Building this module's live test surfaced something this repo has documented since Module 3, but had never actually broken code before now: the default Ollama model is thinking-capable, and its response genuinely splits into multiple `Parts` — an earlier part carries the reasoning trace (`Thought: true`), and the real, user-visible answer lands in a *later*, non-thought part. The first version of this guardrail checked only `Parts[0].Text` and blocked the reasoning trace while letting the real leaked card number straight through in `Parts[1]`, completely unnoticed until the live test's own assertion caught it. The fix: check every part, skip the ones marked `Thought`, and only ever inspect (and, if needed, replace) the genuinely user-visible text.

### Why a Plugin, Not an Instruction

You could just tell the model "never reveal credit card numbers" — but that's not enforcement, it's a request the model can ignore, misunderstand, or be prompted around. A plugin is code: it runs on every single response, for every agent it's registered on, regardless of what any individual agent's own instruction says. That's also why it belongs at the Runner/launcher level (`PluginConfig`), the same placement module-25 established — one guardrail plugin can protect every agent in an organization without touching any of their own prompts.

### Key Takeaways ✅
- **Fail-Closed** means intercepting and evaluating a response *before* the user sees it, not filtering after the fact.
- `OnEventCallback` can rewrite an event, not just observe it — mutate a part's `Text` in place and return the same event, confirmed to genuinely change the user-facing response.
- A thinking model's response is multiple `Parts`; a safety check that only inspects `Parts[0]` can miss the real answer entirely — checked live, not assumed.
- A safety plugin, registered once at the Runner/launcher level, protects every agent that plugin is attached to — no per-agent instruction to maintain or trust.

<hr/>

> **Coming from Python?** 🐍 Python's own module covers the same Fail-Closed pattern and the same `on_event_callback` interception (`BasePlugin`, `event.content.parts[0].text = "..."`). The mechanism maps directly — the one thing worth flagging: Python's own example checks only `parts[0]` too, which works fine against Gemini's plain (non-thinking) output but would need the same "check every non-thought part" fix this module made if it ever ran against a thinking-capable model.
