# Module 26: Callbacks and Guardrails — Agent Safety and Monitoring (Go) 🚦🧯

## Theory

### Six Hooks Into an Agent's Own Execution

A **callback** intercepts one specific stage of one specific agent's run — before or after the agent runs, before or after it calls the model, before or after it calls a tool. `llmagent.Config` exposes all six as plain function slots, each a plural slice (you can register more than one callback per stage — Go supports this natively, unlike a single-callback-per-slot design):

```go
llmagent.Config{
    BeforeAgentCallbacks: []agent.BeforeAgentCallback{cache.beforeAgentCallback},
    AfterAgentCallbacks:  []agent.AfterAgentCallback{cache.afterAgentCallback},
    BeforeModelCallbacks: []llmagent.BeforeModelCallback{beforeModelCallback},
    AfterModelCallbacks:  []llmagent.AfterModelCallback{afterModelCallback},
    BeforeToolCallbacks:  []llmagent.BeforeToolCallback{beforeToolCallback},
    AfterToolCallbacks:   []llmagent.AfterToolCallback{afterToolCallback},
}
```

### Returning Non-Nil Means "Use This Instead"

Every "before" callback can short-circuit the step it guards by returning a non-nil result: `BeforeAgentCallback` returning `*genai.Content` skips the agent's own run entirely (the caching mechanism this module builds); `BeforeModelCallback` returning `*model.LLMResponse` skips the real model call (the input guardrail); `BeforeToolCallback` returning a result map skips the tool call (argument validation). The "after" callbacks work the same way in reverse — a non-nil return replaces what already happened, letting `AfterModelCallback` redact a real model response and `AfterToolCallback` redact a real tool result after the fact.

### Callbacks vs. Plugins: Two Different Scopes, Both Real in Go

Modules 25 and 25.5 already built two real Plugins (`AlertingPlugin`, `PIIGuardrailPlugin`) — both registered once at the Runner/launcher level via `PluginConfig`, observing or intercepting *every* agent that plugin is attached to. Callbacks are the opposite scope: registered directly on *one* agent's own `llmagent.Config`, as part of that agent's own logic. If you want one guardrail protecting every agent in an app, reach for a Plugin (module 25.5's own `PIIGuardrailPlugin` is a real example). If you want to change how *this one* agent caches, validates, or filters, a callback — wired directly into its own config, right here — is the right tool.

### A Real Constraint Found Building This Module: Callback Contexts Don't Support `Session()`

Python's own lab reads `callback_context.session.events` directly to find an agent's last real answer, walked in reverse. The first version of this module's own `afterAgentCallback` tried the same thing in Go — and panicked the instant it ran for real: `agent.Context.Session()`, when the context is a callback context, is deliberately restricted. Confirmed by reading the SDK's own source (`callback_context_wrapper.go`): it logs `"Session() is not supported for callback context"` and returns `nil`, which panics the moment anything calls a method on it.

The real, sanctioned Go mechanism is different, and arguably cleaner: `llmagent.Config.OutputKey`. Its own doc comment names this exact use case — *"Extracts agent reply for later use, such as in tools, callbacks, etc."* Set it once, and the framework itself writes the agent's real final-answer text into session state for you, already correctly skipping any `Thought` parts (confirmed by reading `maybeSaveOutputToState`'s own implementation) — the same lesson module-25.5 had to learn by hand, here already solved by the framework:

```go
llmagent.Config{
    OutputKey: outputKey,
    // ...
}

func (c *responseCache) afterAgentCallback(ctx agent.Context) (*genai.Content, error) {
    val, err := ctx.State().Get(outputKey)
    // ... save val under this turn's own cache key ...
}
```

### A Second Real Bug: Don't Check the Whole Conversation History

The first version of this module's own `beforeModelCallback` checked every entry in `llmRequest.Contents` for a blocked word — but `Contents` carries the *entire* conversation history sent to the model, not just the current turn. A live, multi-turn test caught the real consequence immediately: one blocked-word attempt early in a session permanently refused *every later turn*, including perfectly ordinary follow-up questions. An input guardrail should judge the current input, not something the user already tried and moved on from — fixed by checking only `Contents`' own last entry, the current turn's own message.

### Key Takeaways ✅
- Six callback slots, all plural, cover before/after agent, model, and tool — direct matches for Python's own six, plus native support for registering more than one per stage.
- A non-nil return from a "before" callback means "use this instead"; from an "after" callback, it replaces what already happened.
- Callbacks (node-scoped, part of one agent's own logic) and Plugins (app-scoped, cross-cutting) are genuinely different tools — modules 25/25.5's own shipped Plugins are the real comparison point, not a hypothetical one.
- A callback context doesn't support `Session()` at all — `OutputKey` is the real, sanctioned way to read an agent's own last real answer inside a callback, confirmed live, not assumed from Python's parallel approach.
- An input guardrail must check only the *current* turn, not the whole conversation history — confirmed the hard way, by watching a real multi-turn session get permanently blocked.

<hr/>

> **Coming from Python?** 🐍 Python's own module covers the same six callbacks, the same "return non-nil to override" contract, and the same Callbacks-vs-Plugins distinction. Two things worth flagging directly: Python's `CallbackContext` *does* expose `.session.events` for the reverse-walk this module's own lab describes — Go's equivalent context deliberately doesn't, and `OutputKey` is the real substitute, not a workaround. Separately, Go ships a real, working port of Python's `ReflectAndRetryToolPlugin` (`google.golang.org/adk/v2/plugin/retryandreflect`, its own doc comment links to the Python source it mirrors) but has no shipped equivalent of `ReflectAndRetryModelPlugin`, nor `on_agent_error_callback`/`on_run_error_callback` (confirmed absent by reading every method `plugin.Plugin` exposes) — both are Theory-only asides in Python's own material too, so this mirrors Python's own lab scope exactly, not a gap introduced here.
