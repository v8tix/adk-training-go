# Troubleshooting: Module 19 (Go) 🛠️

### A task-mode sub-agent's response doesn't show up as its own event/author 🤔

**Symptom:** code inspecting the event stream (a test, a custom trace viewer) expects a `ModeTask`/`ModeSingleTurn` sub-agent's response to appear as a distinctly-authored, separately-final event — the way a `ModeChat` hand-off's transferred-to agent does — and finds it isn't there, or finds the coordinator itself credited as the author instead.

**Cause:** confirmed live this module — task/single-turn dispatch happens through a framework-injected function tool (`TaskAgentTool`/`SingleTurnTool`), not `transfer_to_agent`. The calling agent stays the one producing the conversation's visible output; the sub-agent's result is data it consumes, not a hand-off of "who's speaking." A sub-agent's own text can end up folded directly into the coordinator's own response within the same turn.

**Fix:** don't assert on `event.Author` or `event.IsFinalResponse()` to detect a task/single-turn sub-agent's contribution. Assert on the actual visible content instead — concatenate every event's non-thought text for the turn and check for the information you expect to be present, the way this module's own `agent_test.go` does (checking the final plan mentions the airline the user gave, rather than checking which event "belongs" to which agent).

### `OutputKey` on a task-mode agent gets overwritten before the task finishes ⚠️

**Symptom:** code checks `event.Actions.StateDelta[someOutputKey]` expecting it to populate only once the task-mode agent finishes (calls `finish_task`), but the key is already present after the agent's very first, still-incomplete response (e.g. its clarifying question).

**Cause:** confirmed live this module — `OutputKey` writes on every one of that agent's own responses, not just its terminal one. A task-mode agent's mid-conversation question counts as "its own" response for this purpose, the same as its eventual completed answer.

**Fix:** don't use `OutputKey` presence alone as a "did this task finish" signal. If you need to distinguish "still asking" from "done," check the actual value's content, or restructure the check around externally observable behavior (as in this module's own test) rather than key presence.
