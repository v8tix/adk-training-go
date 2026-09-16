# Troubleshooting: Module 17 (Go) 🛠️

### The console shows the model's raw reasoning text mixed into `Agent ->` output 🤯

**Cause:** confirmed live on the local Ollama model used in this course (`qwen3.8:27b`, a reasoning-capable model): its chain-of-thought text isn't filtered out before the node's own output is printed, so `classify_and_route`'s classifier turn and each specialist's turn can show visible reasoning ahead of the actual JSON or analysis text. Gemini's output was clean by comparison in the same run — a real, confirmed cross-backend difference, not a bug in this module's own code.

**Fix:** switch to `MODEL_TYPE=gemini` if you want a clean transcript for demonstration purposes. The routing logic itself is unaffected either way — `classify_and_route` reads the classifier's structured `OutputSchema` result, not its visible text, so reasoning leakage in the console display doesn't change which specialist gets chosen. 👍

### `classify_and_route` returns an error like "classify_and_route: classifier returned no currency" ❌

**Cause:** the classifier's `OutputSchema`-constrained response didn't include a non-empty `currency` field — most likely because the underlying model quantization doesn't support JSON-schema-constrained decoding (module-4's own confirmed finding for *some* quantizations, returning `501 Not Implemented`).

**Fix:** confirm your local model supports `OutputSchema` requests before assuming this module's own code is at fault — try the same request against `MODEL_TYPE=gemini` to isolate whether it's a local-model limitation.
