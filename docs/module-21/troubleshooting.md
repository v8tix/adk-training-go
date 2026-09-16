# Troubleshooting: Module 21 (Go) 🛠️

### The orchestrator can't reach the specialist 📡

**Symptom:** `go run ./cmd/a2a-orchestrator console` fails, hangs, or the model reports it couldn't complete the research request.

**Cause:** `cmd/research-specialist-server` isn't running, was started without the right arguments (`web --port 8001 a2a -a2a_agent_url http://localhost:8001`), or is running on a different address than `cmd/a2a-orchestrator` expects (`RESEARCH_SPECIALIST_URL`, default `http://localhost:8001`).

**Fix:** confirm the server is up first, independently of the orchestrator: `curl -s http://localhost:8001/.well-known/agent-card.json` should return real JSON (an `AgentCard` with a `name` and `supportedInterfaces`). If it doesn't, start the server in its own terminal (with the full `web --port ... a2a -a2a_agent_url ...` arguments) before starting the orchestrator in another. If you changed the server's `--port`/`-a2a_agent_url`, set a matching `RESEARCH_SPECIALIST_URL` for the orchestrator.

### An `event.Author` check alone doesn't prove a remote call succeeded ⚠️

**Symptom:** a test or trace-inspection helper asserts only `event.Author == "research_specialist"` and reports success even when the specialist was unreachable.

**Cause:** confirmed live this module — every failure path in `remoteagent/v2` (an unresolvable agent card, a failed RPC, a timeout) synthesizes its own error event stamped with the same local wrapper agent's name. `event.Author` matching doesn't tell you "the remote agent really answered" apart from "the call failed and this is the resulting error event."

**Fix:** also check `event.ErrorMessage == ""` and that `event.Content` carries real, non-empty text. See `internal/agents/a2aorchestrator/agent_test.go`'s `assertDelegatesToRemoteSpecialist` for the corrected pattern, and `TestA2AOrchestrator_UnreachableSpecialist_Gemini` for a test that proves the fix actually discriminates.

### `cmd/research-specialist-server` won't build — module not found or `go.mod` errors 🔨

**Cause:** this module introduces this repo's first direct import of `github.com/a2aproject/a2a-go/v2` (`a2a`, `a2asrv` packages) — previously only present as an indirect dependency pulled in by the pinned SDK.

**Fix:** run `go mod tidy` from the repo root; it'll promote the dependency to direct and update `go.sum` accordingly. This is a one-time fix already applied in this module's own commit.

### A test using a real HTTP server flakes with a "port already in use" error 🎲

**Cause:** the test starts its own real TCP listener to test the genuine A2A round trip (not a mock), and a hardcoded port could collide with another process, including a previous test run that didn't clean up.

**Fix:** confirmed in this lab's own `agent_test.go`: always bind with `net.Listen("tcp", "127.0.0.1:0")` (port `0` asks the OS for any free port) rather than a fixed port number, and read the actual assigned address back from `lis.Addr()`. This is what the shipped test already does — if you write your own variant, keep this pattern. 👍
