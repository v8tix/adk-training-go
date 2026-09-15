# Module 8: Introduction to Tools (Go)

## Theory

### From Talking to Acting

Every prior module's agent could only reason over what its model already knew from training. This module gives an agent its first way to break out of that box: a **Tool** — a capability an agent can invoke to do something beyond generating text, like searching the web for information that didn't exist when the model was trained.

### `tool.Tool` and `llmagent.Config.Tools`

In the ADK Go SDK, a tool is anything implementing `tool.Tool` (`Name()`, `Description()`, `IsLongRunning()`). Attaching one to an agent is exactly one field on the config already used since module-2:

```go
llmagent.New(llmagent.Config{
    Name:        "researcher_agent",
    Model:       llmModel,
    Instruction: instruction,
    Tools:       []tool.Tool{geminitool.GoogleSearch{}},
})
```

`geminitool.GoogleSearch{}` is a **built-in tool**: unlike a custom function tool (module 9), it runs entirely inside the Gemini model itself — the ADK framework never calls any local code for it. Its `ProcessRequest` method just attaches `&genai.Tool{GoogleSearch: &genai.GoogleSearch{}}` to the outgoing request; Gemini decides when to search and folds the results into its own reasoning before responding.

### A Real, Confirmed Local-Inference Gap — the Same Category as Module 7's

This module found the same class of gap module-7 found for images, in a different SDK code path:

1. **`google_search` is a Gemini-native built-in tool, not a function the ADK calls locally.** Confirmed via source: `tool/geminitool/google_search.go`'s `ProcessRequest` only ever sets `genai.Tool.GoogleSearch` — there's no local `Run` method to execute.
2. **This repo's Go client for the local Ollama backend rejects it outright.** `model/openaimodel/tools.go`'s `ensureFunctionToolOnly` explicitly checks for and rejects `GoogleSearch` (along with every other non-function built-in: `Retrieval`, `GoogleMaps`, `CodeExecution`, etc.) with `"openai: non-function tools are not supported (tool %d)"`. Confirmed live: building a `researcher_agent` with this tool against the Ollama backend and running it fails immediately with exactly that error, before any network call.

**The precise, correct claim is: this SDK's local-model client can't send any non-function built-in tool — not that Ollama or the underlying model has no search capability at all.** `cmd/researcher` therefore requires `MODEL_TYPE=gemini`, hardcoded in its own `main()`, same as `cmd/visual-catalog` in module-7.

### `google_search` Needs No New Setup Either

Confirmed live: the same `GOOGLE_AI_STUDIO_API_KEY` path every module since module-2 already uses is enough for `google_search`, too — no separate project/location configuration needed.

```
$ go run ./temp/module-8/probe2   # a researcher_agent, MODEL_TYPE=gemini, existing GOOGLE_AI_STUDIO_API_KEY
Using model: gemini-3.5-flash
GroundingMetadata present: true
WebSearchQueries: [...]
Answer: <a real, current, grounded answer>
```

No `GOOGLE_GENAI_USE_VERTEXAI`, `GOOGLE_CLOUD_PROJECT`, or `GOOGLE_CLOUD_LOCATION` set anywhere — the same `GOOGLE_AI_STUDIO_API_KEY` path every module since module-2 already uses was sufficient.

### Proving the Tool Actually Fired: `event.GroundingMetadata`

`session.Event` (returned by every `runner.Run` call) embeds `model.LLMResponse`, which carries `GroundingMetadata *genai.GroundingMetadata` directly — non-nil exactly when a real search happened. That gives you an automated, structural way to assert a tool actually fired, instead of only being able to check by eye in the Dev UI's Trace view (which is still there, and still useful — see the lab).

### Key Takeaways
- A **built-in tool** like `google_search` runs inside the model itself; a **custom function tool** (next module) is your own code the ADK calls locally. `llmagent.Config.Tools []tool.Tool` is the attachment point for both.
- The local Ollama backend's Go client can't send any non-function built-in tool (confirmed via the exact source check and a live, reproduced error) — a real, precisely-scoped SDK limitation, not a claim about Ollama's or the model's own capabilities.
- The plain `GOOGLE_AI_STUDIO_API_KEY` path already supports `google_search`, confirmed live — no new setup needed.
- `event.GroundingMetadata` gives a real, automatable way to verify a built-in tool fired, instead of relying only on manual Trace-view inspection.

<hr/>

> **Coming from Python?** Python's lab asks for a full Vertex AI setup for `google_search` (`GOOGLE_GENAI_USE_VERTEXAI`, a project ID, a location) because the tool "requires an Agent Platform configuration." This repo's simpler `GOOGLE_AI_STUDIO_API_KEY` path handles it just fine, confirmed live — the second time this course has found a Python-stated Vertex AI requirement unnecessary here (module-7 found the same for vision). And where Python's lab verifies the tool fired by manually reading the Dev UI's Trace view, `event.GroundingMetadata` lets you check that automatically instead.
