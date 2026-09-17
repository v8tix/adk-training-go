# Lab 23: Building a Document Processing Pipeline (Go) 📄🖼️

## Goal

Build a **Document Processor** agent that runs a four-step pipeline — extract, summarize, chart, report — saving each step's output as a versioned artifact, chaining each step's output into the next.

## Lab Tasks

### 1. Read `internal/agents/documentprocessor/tools.go`

Four handlers, each showing a different part of the Artifact system:

- `extractText` — saves cleaned text as `"{document_name}_extracted.txt"` via `ctx.Artifacts().Save`. Call it twice on the same document and you get versions 1, then 2 — never an overwrite.
- `summarizeDocument` — loads the extracted-text artifact first. If it doesn't exist yet (`errors.Is(err, fs.ErrNotExist)`), it returns a normal, successful result with a helpful message instead of a Go error — the model can react to that in conversation rather than crashing.
- `generateChart` — this module's one binary artifact: a hardcoded dummy PNG saved via `genai.NewPartFromBytes(pngBytes, "image/png")`.
- `createReport` — lists every artifact (`ctx.Artifacts().List`), filters to this document's own files, and branches on `part.InlineData.MIMEType` to tell the chart image apart from the text artifacts while compiling the final report.

### 2. Read `internal/agents/documentprocessor/agent.go` and its prompt

Same `functiontool.New` + `Tools` shape as every prior module's agent. Its instruction (`prompts/document_processor_instruction.md`) tells the model to run the four steps strictly in order, since each one reads what the previous one saved.

### 3. Run it — console mode, entirely locally 🖥️

```bash
go run ./cmd/document-processor console
```

No `.env`, no API key needed. Real, confirmed output from this exact command (thinking-model reasoning trimmed for readability):

```
📄 document-processor using qwen3.8:27b

User -> Process the document named 'AnnualReport'.
Agent -> All done! Here's a summary of the processing for **AnnualReport**:

1. **Extracted text** → saved as `AnnualReport_extracted.txt` (version 1)
2. **Summary** → saved as `AnnualReport_summary.txt` (version 1)
3. **Stats chart** → saved as `AnnualReport_chart.png` (version 1)
4. **Final report** → saved as `AnnualReport_FINAL_REPORT.md` (version 1)

All four files were generated successfully and are ready for your use. Let me know if you'd like anything else!
```

One user message, four chained tool calls, each saving a real versioned artifact — confirmed structurally, not just by this transcript, in `internal/agents/documentprocessor/agent_test.go`.

**Note on the `ArtifactService` wiring:** unlike every prior module's `cmd/`, this one's `main.go` sets `ArtifactService: artifact.InMemoryService()` explicitly in `launcher.Config`. That's required, not decorative — `runner.New` never defaults a nil `ArtifactService` the way `web` mode's own internal wiring does; skip it, and any tool calling `ctx.Artifacts().Save(...)` panics on a nil interface. If you're extending this lab into a new `cmd/` program of your own that touches artifacts, don't forget this line.

### 4. Read `internal/agents/documentprocessor/tools_test.go`

Pure unit tests, no LLM — a new `fakeArtifacts` double (map-backed, 1-indexed versions, matching the real backends exactly) plus `erroringArtifacts` for propagation tests. Covers version incrementing, the not-found-vs-real-error distinction, MIME type correctness, and — a real edge case worth noticing — `createReport` excluding its own prior version when run twice on the same document.

### 5. Read `internal/agents/documentprocessor/artifacts_test.go`

Three standalone tests against the *real* `artifact.InMemoryService()`, no LLM: versions genuinely start at 1, a `user:`-prefixed artifact genuinely crosses sessions, and — the negative case — a plain filename genuinely doesn't. This is what actually proves the scoping and versioning claims, not just the fake double module 4 relies on for its own unit tests.

### 6. Read `internal/agents/documentprocessor/agent_test.go`

`TestDocumentProcessor_BuildsVersionedPipeline_{Ollama,Gemini}` drives the whole pipeline through one `Run()` call, then reads the artifact store directly afterward: all four files exist, the extracted text is genuinely version 1, the chart's MIME type is genuinely `image/png`, and the final report's own content genuinely references the chart filename — the real, structural proof the four steps chain correctly.

## Self-Reflection Questions 🤔
- Why does `summarizeDocument` return a normal result (not a Go error) when the extracted-text artifact is missing, while a genuine service failure still propagates as a real error?
- What would happen if `createReport` didn't exclude its own filename from the list it processes? Try removing that check and re-running `TestCreateReport_ExcludesItsOwnPriorVersion` to see.
- If you wanted a chart to persist across every session a user starts (not just this one), what single-character change to its filename would you make?
- Why does `console` mode need `ArtifactService` set explicitly in `main.go`, while `web` mode doesn't?

<hr/>

### Looking for the solution? 🔍

Hint: read `internal/agents/documentprocessor/tools.go` and `agent.go` for the real mechanism — four tool functions, each touching `ctx.Artifacts()`, wrapped the same way every prior module's tools were.
