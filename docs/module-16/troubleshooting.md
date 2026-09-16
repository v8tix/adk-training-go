# Troubleshooting: Module 16 (Go)

### The local model refuses to invent headlines

**Cause:** neither research agent has a real search tool, so a local Ollama model asked to "find headlines" with nothing to search with may honestly decline rather than fabricate a plausible-sounding answer — a real, confirmed behavior, not a bug.

**Fix:** switch to `MODEL_TYPE=gemini` if you want a more filled-in example — Gemini tends to answer confidently from its own training data instead of declining. See this module's README for the full, confirmed cross-backend difference.

### The summarizer's instruction shows literal `{tech_news}` instead of real content

**Cause:** the `OutputKey` set on a researcher agent doesn't exactly match the placeholder name used in `summarizer_instruction.md` — the substitution is matched by exact string, not fuzzy.

**Fix:** confirm the two `OutputKey`s exactly match the placeholder names in `summarizer_instruction.md`.
