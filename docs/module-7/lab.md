# Lab 7: Building a Visual Product Catalog Analyzer (Go)

## Goal

Build a vision-capable agent that analyzes a product photo and writes a marketing description — this repo's mirror of Python's "Visual Product Catalog Analyzer," using the App/Runner pattern from module-6 plus one new piece: explicit session creation.

## Lab Tasks

### 1. Read `internal/agents/visualcatalog/agent.go`

Same shape as `internal/agents/supportanalyzer` (module-6) — `BuildRootAgent(llmModel)` resolves its own prompt internally — but simpler: no `OutputSchema`/`OutputKey`, since this agent returns plain marketing copy, not structured JSON.

### 2. Read `cmd/visual-catalog/main.go`

Three things to notice:

- **`cfg.ModelType = llm.ModelTypeGemini` is set in code, not left to `.env`.** Vision needs it — the local Ollama backend's Go client has no way to send an image (see the README for the confirmed finding). Setting this in code means the program can't accidentally hit that error path just because someone's `.env` defaults to `ollama`.
- **`runOnce`-style helper, but with an explicit session first.** `analyzeProduct` calls `sessionSvc.Create(...)` before `r.Run(...)` — unlike every prior module's `runner.NewInMemory`-based helper, which never needed this because `NewInMemory` auto-creates sessions.
- **The image itself:** `os.ReadFile(imagePath)` + `genai.NewPartFromBytes(imageBytes, "image/jpeg")`, combined with a text part via `genai.NewContentFromParts`.

### 3. Run it

```bash
go run ./cmd/visual-catalog
```

(Requires `GOOGLE_AI_STUDIO_API_KEY` in your environment or `.env` — see `.env.example`.)

Real, confirmed output from this exact command (abbreviated — the full descriptions are longer):

```
🎨 visual-catalog using gemini-3.5-flash

--- Analyzing Product: HEADPHONES-01 ---
📸 Sending image to Gemini...
✅ Description:
### HEADPHONES-01 — Premium Over-Ear Audiophile Headphones
Elevate your listening experience with the HEADPHONES-01, designed for
those who appreciate both exceptional sound quality and timeless design...

--- Analyzing Product: LAPTOP-02 ---
📸 Sending image to Gemini...
✅ Description:
# LAPTOP-02 — High-Performance Sleek Ultrabook
Elevate your daily productivity and creative workflows with the LAPTOP-02...
```

Two distinct, accurate descriptions, each correctly identifying the real product in its real photo — headphones described as headphones, a laptop described as a laptop, with specific visual details (the coiled cable, the wooden desk, the laptop's chassis) pulled from the actual images, not generic boilerplate.

### 4. See the session requirement fail, on purpose

Try commenting out the `sessionSvc.Create(...)` call in `analyzeProduct` and re-run. You should see the real, confirmed error this module is about:

```
session not found: "sess_HEADPHONES-01"
```

Put the `Create` call back before moving on — this is meant to be observed, not left broken.

### 5. Bonus (outside this course's ADK-SDK lesson): confirm the gap is in the SDK, not the model

```bash
go run ./cmd/visual-catalog-local
```

No `GOOGLE_AI_STUDIO_API_KEY` needed — this one runs entirely against the repo's shared local default. Real, confirmed output:

```
🎨 visual-catalog-local using qwen3.8:27b directly (no ADK agent/runner)

--- Analyzing Product: HEADPHONES-01 ---
✅ Description:
A pair of black over-ear headphones with brushed silver/metallic ear-cup rims...

--- Analyzing Product: LAPTOP-02 ---
✅ Description:
A silver laptop sits open at the center of a light wooden desk...
```

This isn't a second way to do the real lab — it bypasses `llmagent`/`runner` entirely, calling Ollama directly (via [kawa](https://github.com/v8tix/kawa), not the ADK SDK). It exists only to prove step 1's claim from the other side: the local model server can see images just fine; the gap really is in `model/openaimodel`.

## Self-Reflection Questions
- Why did `cmd/support-analyzer-runner` (module-6) never need an explicit session-creation call, but `cmd/visual-catalog` does?
- This module's local-inference limitation is specifically that the Go SDK's client can't *send* an image — the model server itself handles the same image fine when called directly. Why does that distinction matter if you were deciding whether to file a bug against the SDK versus against Ollama?
- If you wanted to analyze a PDF document instead of an image, what would you change in `analyzeProduct` — the `Part` construction, the MIME type, or both?
- `cmd/visual-catalog-local` reaches the same local model this repo already uses everywhere else, just without going through `llmagent`/`runner`. What would you lose by always doing this instead — what does the ADK agent/runner layer give you that a raw HTTP call doesn't?

<hr/>

### Looking for the solution?

Hint: read `internal/agents/visualcatalog/agent.go` and `cmd/visual-catalog/main.go`'s `analyzeProduct` function for the real lesson — that's the whole ADK-based mechanism, end to end. For the bonus, `internal/agents/visualcatalog/local_vision.go`'s `DescribeImageLocally` and `cmd/visual-catalog-local/main.go`.
