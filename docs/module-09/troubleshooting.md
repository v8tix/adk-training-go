# Troubleshooting: Module 9 (Go) 🔧

### `400 INVALID_ARGUMENT` when mixing `geminitool.GoogleSearch{}` with a custom function tool

**Symptom:**

```
400 ... Please enable tool_config.include_server_side_tool_invocations
to use Built-in tools with Function calling.
```

**Cause:** attaching both a built-in tool (like `google_search`) and a custom function tool to the same agent's `Tools` list fails by default — a real restriction straight from the Gemini API itself, confirmed via `genai` source.

**Fix:** set `llmagent.Config.GenerateContentConfig.ToolConfig.IncludeServerSideToolInvocations = true`. Confirmed live: the same mixed-tool agent then works correctly, computing a real sum with both a built-in and a custom tool attached. ✅ This field is **Developer-API-only** (`genai`'s own doc comment says it straight: "This field is not supported in Vertex AI") — the same simpler `GOOGLE_AI_STUDIO_API_KEY` path this repo already uses, so no extra credential needed.
