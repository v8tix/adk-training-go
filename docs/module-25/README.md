# Module 25: Advanced Observability with Plugins (Go) 🔭📊

## Theory

### Watching an Agent Without Touching Its Logic

A tool that can fail is only half the story — you also need to know *when* it failed, how often, and whether the agent recovered. Bolting that logic directly into every tool function works, but it clutters business logic with cross-cutting concerns and has to be repeated per tool. A **Plugin** solves this by sitting at the Runner level, watching every tool call and every event across the whole agent, without either the tool or the agent needing to know it exists.

### A Plugin Is a Set of Callbacks, Not a Class to Subclass

`google.golang.org/adk/v2/plugin` builds a plugin from a config of named callback fields — no interface to implement, no base type to embed:

```go
func newAlertingPlugin(name string) (*plugin.Plugin, *alertTracker, error) {
	tracker := &alertTracker{}
	return plugin.New(plugin.Config{
		Name:                name,
		OnToolErrorCallback: tracker.onToolError,
		OnEventCallback:     tracker.onEvent,
	})
}
```

`OnToolErrorCallback` fires whenever a tool returns a genuine error — not a "soft" structured failure like `{"status": "error"}`, but an actual Go `error` return. Returning a non-nil result from the callback (instead of the error) tells the framework "I've handled this, let the agent recover gracefully" — the exact same recovery contract this module's `riskyOperation` tool relies on:

```go
func (a *alertTracker) onToolError(_ agent.Context, t tool.Tool, _ map[string]any, err error) (map[string]any, error) {
	a.hadErrorThisTurn = true
	a.errorCount++
	// ... log an alert, escalating past a threshold ...
	return map[string]any{"status": "error", "message": err.Error()}, nil
}
```

`OnEventCallback` fires on every event the agent produces; checking `event.IsFinalResponse()` — the same method every prior module's tests have used to find a turn's real answer — lets a plugin tell "this turn just finished" from "this is an intermediate step," so `alertTracker` only resets its count once a turn completes cleanly.

### Registered at the Runner, Not the Agent

A Plugin is deliberately kept separate from an agent's own callback slots (`llmagent.Config`'s `BeforeToolCallbacks`/`AfterToolCallbacks`/`OnToolErrorCallbacks`) — those are per-agent, this is cross-cutting. It's registered via `runner.Config.PluginConfig`/`launcher.Config.PluginConfig`, which both `console` and `web` sub-launchers already wire straight into the real `Runner`:

```go
config := &launcher.Config{
	AgentLoader:  agent.NewSingleLoader(rootAgent),
	PluginConfig: runner.PluginConfig{Plugins: []*plugin.Plugin{alertingPlugin}},
}
```

### Every Node Already Has Real, Graph-Aware Tracing

Beyond custom plugins, the SDK has native OpenTelemetry support wired in — and it's genuinely graph-aware, not a generic wrapper. `internal/telemetry/node_tracing.go` starts a real span named `invoke_agent <name>` (or `invoke_workflow`/`invoke_node` for a graph's own nodes), tagged with a `gen_ai.agent.name` attribute — confirmed live this session by capturing a real span with an in-memory exporter and reading its name and attributes directly. Its own doc comment states it deliberately mirrors `adk-python`'s own `node_tracing` module's semantic conventions.

### It's Already Wired Into Every `cmd/` Program

Both `console` and `web` launchers expose a `-otel_to_cloud` flag, wired straight to `google.golang.org/adk/v2/telemetry`, with no code change needed:

```bash
go run ./cmd/observability-agent console -otel_to_cloud=false   # default: no cloud credentials needed
go run ./cmd/observability-agent console -otel_to_cloud=true    # needs real Google Application Default Credentials
```

This module's own lab and tests only ever use the safe default — proving the mechanism works via a real OTLP export to a local, disposable Jaeger container instead of a real Google Cloud project.

### A Real Gotcha #1: the Global Provider Binds Once Per Process

`SetGlobalOtelProviders()` registers your `*telemetry.Providers` with OTel's global registry — but the SDK's own span-emitting code (`internal/telemetry`) doesn't look up that registry fresh on every span. It resolves a `Tracer` handle from the global registry *once*, into a package-level variable, the first time any code installs a real provider in the process. OTel's Go API has a delegating mechanism specifically so a handle obtained before a real provider exists still works once one is installed — but that delegate only binds *once*: a *second* `SetGlobalOtelProviders()` call later in the same process doesn't redirect an already-bound handle. Building this module's own telemetry test hit this directly: splitting the "in-memory exporter" check and the "real Jaeger export" check into two separate top-level test functions, each doing its own `telemetry.New`/`SetGlobalOtelProviders()` setup, silently broke the second one — its spans never reached its own exporter. The fix: one process gets one real provider-install call. If you need multiple exporters, attach them all to the *same* provider before installing it, not to two providers installed one after another.

### A Real Gotcha #2, Confirmed by Actually Hitting It

Building this module's own telemetry test surfaced a second thing worth knowing: `go.opentelemetry.io/otel/sdk/trace/tracetest.InMemoryExporter`'s `Shutdown` method calls `Reset()` internally — it silently clears every span it's holding. Calling `Shutdown` on a shared `*telemetry.Providers` to flush a *different* exporter (a real OTLP one, in this case) will also wipe an in-memory exporter attached to the same provider, if you read its spans afterward. The fix: read what you need from an in-memory exporter *before* calling `Shutdown`, not after — confirmed the hard way, not assumed.

### Key Takeaways ✅
- A Plugin observes an agent's real tool calls and events without touching either's own code — registered at the Runner/launcher level, not per-agent.
- A Plugin instance is shared for the whole process once registered — guard any mutable state it keeps (like a running error count) with a mutex, since the SDK runs a turn's multiple tool calls concurrently by default, and a multi-user launcher shares one plugin across every session.
- `OnToolErrorCallback` fires only on a genuine Go `error` return; returning a non-nil result instead lets the agent recover gracefully.
- `OnEventCallback` + `event.IsFinalResponse()` is how a plugin tells a completed turn from an intermediate step.
- Graph-aware OTel tracing is real and already running — `invoke_agent`/`invoke_workflow`/`invoke_node` spans with real `gen_ai.*` attributes, confirmed live via a captured span, not just cited from docs.
- Every `cmd/` program already has a `-otel_to_cloud` flag; the safe default needs no cloud credentials at all.
- A real OTel provider only binds once per process — install every exporter you need on one provider, not on two providers installed one after another.
- `tracetest.InMemoryExporter.Shutdown()` clears its own recorded spans — read before you shut down, not after.

<hr/>

> **Coming from Python?** 🐍 Python's own module covers the same Plugin System (`BasePlugin`, `on_tool_error_callback`, `on_event_callback`, `App(plugins=[...])`) and the same native OTel/Cloud Trace integration (`get_gcp_exporters`, `maybe_set_otel_providers`, every event carrying a `node_info` field). Go's version matches closely in spirit — a config of callback functions instead of a subclassed `BasePlugin`, the same recovery contract, the same graph-aware tracing goal — just expressed as real OTel span attributes rather than a `node_info` object copied onto the event itself.
