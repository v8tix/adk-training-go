# Lab 25: Building an Alerting Plugin and Proving Real Tracing (Go) 🔭📊

## Goal

Build a Plugin that watches an agent for consecutive tool errors, escalating an alert and resetting on recovery — then prove the SDK's real, graph-aware OpenTelemetry tracing actually works, at two levels of rigor.

## Lab Tasks

### 1. Read `internal/agents/observability/tools.go`

`riskyOperation` returns a genuine Go `error` (`ErrSimulatedFailure`) when asked to fail — not a structured `{"status": "error"}` result like `calculator.divide`'s. `OnToolErrorCallback` only fires on a real error return, and this module is about observing exactly that path.

### 2. Read `internal/agents/observability/alerting_plugin.go`

`alertTracker` is a plain struct with two methods — `onToolError` (marks the turn errored, increments a count, logs an alert that escalates to CRITICAL at 3 consecutive failures, and returns a graceful result so the agent doesn't crash) and `onEvent` (resets the count once a turn finishes cleanly, via `event.IsFinalResponse()`). `newAlertingPlugin` wraps both via `plugin.New` — no interface to implement, just matching function signatures.

### 3. Read `internal/agents/observability/agent.go`

Notice the plugin is **not** attached here — `BuildRootAgent` only wires the tool. The plugin gets registered separately, at the Runner/launcher level, in `cmd/observability-agent/main.go`. This separation is the whole point of a Plugin System.

### 4. Run it — console mode, entirely locally 🖥️

```bash
go run ./cmd/observability-agent console
```

Real, confirmed output from a 4-turn conversation (three failures, then a clean recovery; thinking-model reasoning trimmed for readability):

```
🔭 observability-agent using qwen3.8:27b

User -> Please call risky_operation and make it FAIL.
⚠️  ALERT: tool "risky_operation" failed (1 consecutive so far): simulated failure
Agent -> The operation failed as requested. It returned a simulated failure error.

User -> Please call risky_operation and make it FAIL.
⚠️  ALERT: tool "risky_operation" failed (2 consecutive so far): simulated failure
Agent -> The operation failed as you requested. It returned a simulated failure error.

User -> Please call risky_operation and make it FAIL.
🚨 CRITICAL ALERT: tool "risky_operation" has failed 3 times consecutively (simulated failure)
Agent -> The operation failed as requested — it returned a "simulated failure" error.

User -> Please call risky_operation without making it fail.
Agent -> The operation completed successfully this time. No failure occurred.
```

Three escalating alerts, then silence on the clean turn — the plugin's own reset logic, observed live.

### 5. Read `internal/agents/observability/alerting_plugin_test.go`

Pure unit tests, no LLM: escalation at the threshold, the graceful (non-error) return value, resetting only after a clean final turn, and a non-final event being a no-op. `nil` is passed for the `agent.Context`/`agent.InvocationContext` parameters in these tests — both callback methods only use their `tool`/`event`/`error` arguments, so no test double is needed for the context types at all.

### 6. Read `internal/agents/observability/agent_test.go`

`TestAlertingPlugin_EscalatesOnConsecutiveErrors_{Ollama,Gemini}` proves the *real* Plugin mechanism, not just the unit-level callback logic already proven above: it builds a real `runner.New` with the real plugin wired into `PluginConfig.Plugins`, drives three separate `Run()` calls, then reads the plugin's own `alertTracker.errorCount` directly (via `newAlertingPlugin`'s tracker return value) to confirm the real interception happened — not parsed from console output or model text.

### 7. Read `internal/agents/observability/telemetry_test.go`

One test function, two subtests, sharing one telemetry setup (see the file's own top comment for why — calling `SetGlobalOtelProviders` twice in one process doesn't work the way you'd expect):

- `in-memory-attributes` — always runs, no Docker needed. Captures a real `invoke_agent` span via an in-memory exporter and checks its `gen_ai.agent.name` attribute.
- `real-otlp-export-to-jaeger` — starts a real `jaegertracing/all-in-one` container via Testcontainers, exports a real span to it over real OTLP gRPC, then queries Jaeger's own HTTP API to confirm it arrived. Skips cleanly if Docker isn't available.

Run both:

```bash
go test ./internal/agents/observability/... -run TestTelemetry_GraphAwareTracing -v
```

Real, confirmed output from this exact command (with Docker available):

```
=== RUN   TestTelemetry_GraphAwareTracing
=== RUN   TestTelemetry_GraphAwareTracing/in-memory-attributes
=== RUN   TestTelemetry_GraphAwareTracing/real-otlp-export-to-jaeger
--- PASS: TestTelemetry_GraphAwareTracing (14.53s)
    --- PASS: TestTelemetry_GraphAwareTracing/in-memory-attributes (0.00s)
    --- PASS: TestTelemetry_GraphAwareTracing/real-otlp-export-to-jaeger (0.00s)
PASS
```

**A real bug this test caught while being written:** the in-memory exporter's own `Shutdown` method clears its recorded spans as a side effect — calling `providers.Shutdown()` (needed to flush the *other*, OTLP exporter) before reading the in-memory one's spans silently wiped them. The fix was to read the in-memory spans first, then shut down. See the test file's own comments for the full story.

## Self-Reflection Questions 🤔
- Why does `alertTracker.onToolError` return a non-nil result instead of the error it received? What would happen to the agent's run if it returned the error unchanged instead?
- `agent_test.go`'s live test reads `alertTracker.errorCount` directly rather than checking the console output for "🚨 CRITICAL ALERT" text. Why is that a stronger proof?
- Both telemetry subtests share one `telemetry.New`/`SetGlobalOtelProviders()` call instead of each having its own. What would go wrong if they didn't?
- `-otel_to_cloud=true` is documented but never actually used in this module's own tests. Why not, and what would you need to actually try it?

<hr/>

### Looking for the solution? 🔍

Hint: read `internal/agents/observability/alerting_plugin.go` and `agent.go` for the real mechanism — a plain struct's two methods, wrapped by `plugin.New`, registered separately from the agent itself in `cmd/observability-agent/main.go`.
