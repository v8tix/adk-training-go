# Troubleshooting: Module 12 (Go) 🔧

### `400 INVALID_ARGUMENT` mentioning `include_server_side_tool_invocations`

**Cause:** you've built one agent with both `geminitool.GoogleSearch{}` and a custom tool, without setting that flag.

**Fix:** either set `GenerateContentConfig.ToolConfig.IncludeServerSideToolInvocations = true` (the combined-agent approach from Step 3), or split back into two agents (the sequential-composition approach from Steps 1-2) — never both a built-in and a custom tool in one agent's `Tools` with no `ToolConfig`.

### The local Ollama backend rejects the request

**Cause:** this lab requires `MODEL_TYPE=gemini` — `cmd/research-assistant/main.go` forces this in code, since `google_search` only works on Gemini 2.0+ models.

**Fix:** double-check your `GOOGLE_AI_STUDIO_API_KEY` is actually set if you're seeing an authentication error instead of the expected local-backend rejection.

### `401 CREDENTIALS_MISSING: API keys are not supported by this API` (Vertex AI bonus)

**Cause:** `VERTEX_AI_API_KEY` is set to a plain Google Cloud API key (the classic `AIzaSy...` format, generated from the normal Cloud Console credentials page) instead of a key generated through Vertex AI Studio's own **Express Mode** enrollment flow. The `aiplatform.googleapis.com` PredictionService rejects a non-Express-enrolled key outright — confirmed live, this is a real Google Cloud auth restriction, not a bug in this repo's code.

**Fix:** either (a) generate a real Express Mode key at `console.cloud.google.com/vertex-ai/generative/express` and use that instead — not every existing Google Cloud account can access this enrollment flow — or (b) clear `VERTEX_AI_API_KEY` and use Application Default Credentials instead: set `VERTEX_AI_PROJECT` + `VERTEX_AI_LOCATION`, then run `gcloud auth application-default login`.

### `404 NOT_FOUND: Publisher model ... was not found` (Vertex AI bonus)

**Cause:** `VertexAIModel` (or a manual override of it) names a model that isn't in Vertex AI's publisher model catalog for the given project/region — confirmed live: `gemini-3.5-flash` (this repo's own `GeminiModel` default, used by the public Gemini API) doesn't exist on Vertex, even though the model family sounds the same. Vertex AI's own model catalog uses different, often-lagging identifiers than the public Gemini API.

**Fix:** use `VertexAIModel`'s own default (`gemini-2.5-flash`, confirmed live to work), or check which models are actually available for your project/region before overriding `VERTEX_AI_MODEL` to something else.
