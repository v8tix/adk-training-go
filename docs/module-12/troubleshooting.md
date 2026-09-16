# Troubleshooting: Module 12 (Go)

### `400 INVALID_ARGUMENT` mentioning `include_server_side_tool_invocations`

**Cause:** you've built one agent with both `geminitool.GoogleSearch{}` and a custom tool, without setting that flag.

**Fix:** either set `GenerateContentConfig.ToolConfig.IncludeServerSideToolInvocations = true` (the combined-agent approach from Step 3), or split back into two agents (the sequential-composition approach from Steps 1-2) — never both a built-in and a custom tool in one agent's `Tools` with no `ToolConfig`.

### The local Ollama backend rejects the request

**Cause:** this lab requires `MODEL_TYPE=gemini` — `cmd/research-assistant/main.go` forces this in code, since `google_search` only works on Gemini 2.0+ models.

**Fix:** check your `GOOGLE_AI_STUDIO_API_KEY` is actually set if you see an authentication error instead of the expected local-backend rejection.
