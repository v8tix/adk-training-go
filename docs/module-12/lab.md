# Lab 12: Building a Research Assistant with Web Search (Go) 🔎

## Goal

Build a research assistant that grounds itself in live web results, then hands those results to a second agent that formats them into a report — the sequential-composition pattern from this module's README, built end-to-end.

### Prerequisites

This lab needs a real `GOOGLE_AI_STUDIO_API_KEY` in your `.env` — `google_search` only works on Gemini 2.0+ models, and the local Ollama backend rejects it outright before ever reaching the network (confirmed in module-8). There's no local-only path through this one, sorry! 😅

### Step 1: The Two Agents

`internal/agents/researchassistant/agent.go` defines two agents:

```go
func BuildResearchAgent(llmModel model.LLM) (agent.Agent, error) {
    // Only geminitool.GoogleSearch{} in Tools.
}

func BuildFormatterAgent(llmModel model.LLM) (agent.Agent, error) {
    // Only extract_key_facts and format_research_notes in Tools — never google_search.
}
```

The two custom tools the formatter agent calls live in `tools.go`:

- `extractKeyFacts(ctx, ExtractKeyFactsArgs{Text, NumFacts}) (ExtractKeyFactsResult, error)` — splits `Text` on `.`, keeps sentences longer than 10 characters, up to `NumFacts` of them.
- `formatResearchNotes(ctx, FormatResearchNotesArgs{Topic, Findings}) (FormatResearchNotesResult, error)` — builds a Markdown-style report with a generated timestamp.

Read both agent builders and both tool functions before moving on — this lab's about how they're orchestrated, not about writing new tool logic.

### Step 2: Run the Pipeline 🏃

`cmd/research-assistant/main.go`'s `runResearchPipeline` does the orchestration:

```go
findings, err := runAgent(ctx, researchAgent, "research_app", "Research this topic: "+topic)
report, err := runAgent(ctx, formatterAgent, "formatter_app", "Topic: "+topic+"\n\nFindings: "+findings)
```

`runAgent` is a small helper around `runner.NewInMemory` + `runner.Run`, returning the agent's final non-thought text — the same programmatic-execution shape from module-6 and module-10, just called twice with two different agents.

Run it:

```bash
go run ./cmd/research-assistant "the latest AI developments from Google"
```

Real, confirmed output from this exact command (abbreviated — the research findings continue with three more bulleted developments, and the report's Findings section has three more facts):

```
🔎 research-assistant using gemini-3.5-flash
--- RESEARCH FINDINGS ---
In 2026, Google's artificial intelligence developments reflect a major
industry-wide shift from passive chat queries toward proactive "agentic AI"—
systems designed to execute complex, multi-step workflows autonomously.

Major developments include:

*   **The Gemini 3.5 Family & Next-Gen Models:** Google launched the
    **Gemini 3.5** model line, with **Gemini 3.5 Flash** (and the
    later-released **Gemini 3.8 Flash**) optimized for fast, autonomous
    execution loops. [...]
*   **Gemini Omni:** Developed by Google DeepMind, **Gemini Omni** is a
    unified multimodal model capable of generating high-quality video and
    native audio from any combination of inputs. [...]
[...]

--- FINAL REPORT ---
# Research Report: the latest AI developments from Google
Generated: 2026-09-15 09:27:29

## Findings
- In 2026, Google's artificial intelligence developments reflect a major
  industry-wide shift from passive chat queries toward proactive "agentic
  AI"—systems designed to execute complex, multi-step workflows autonomously.
- Google launched the Gemini 3.5 model line, with Gemini 3.5 Flash and
  Gemini 3.8 Flash optimized for fast, autonomous execution loops, with
  rumors of a Gemini 4 model in late 2026.
- Developed by Google DeepMind, Gemini Omni is a unified multimodal model
  capable of generating high-quality video and native audio, including
  progressive video editing.
[...]
```

Notice: the research agent's findings reflect real, current events beyond any training cutoff — proof `google_search` genuinely ran — and the formatter agent never touched the web at all, only the findings text it was handed. Neat separation of concerns! 🎯

### Step 3 (Bonus): One Agent, Both Tool Types 🎁

`BuildCombinedAgent` in the same package shows the alternative this module's README covers: instead of two agents, one agent with `IncludeServerSideToolInvocations` set can use `google_search` and your custom tools together. Read `TestCombinedAgent_UsesSearchAndCustomTool_Gemini` in `agent_test.go` to see how that's proven — it checks for real `GroundingMetadata` *and* a real `FunctionResponse` from `format_research_notes` in the same conversation.

### Troubleshooting

Hit a snag? See [troubleshooting.md](./troubleshooting.md).

### Lab Summary 🎉

You built a two-agent research pipeline using the ADK's built-in `google_search` tool, learned exactly why the Gemini API rejects mixing it with custom tools by default, and saw both ways around that: splitting into two agents, or setting one flag to combine them in one. Solid work!

### Self-Reflection Questions 🤔
- Why does `google_search` running "inside the model" make it a fundamentally different kind of tool than `extract_key_facts`, which runs in your own Go process?
- `TestMixedTools_WithoutServerSideFlag_Fails_Gemini` deliberately builds an invalid agent to prove the restriction is real. Why's a test that expects failure just as valuable as one that expects success?
- Now that you know `IncludeServerSideToolInvocations` exists, when would you still pick sequential composition (two agents) over the combined single-agent approach?

<hr/>

> **Coming from Python?** 🐍 Python's lab only builds the sequential-composition version — `formatter_agent` is left as a `TODO` for you to complete, then `main.py`'s `run_agent` helper does the same two-call orchestration `runAgent` does here. This Go lab throws in Step 3 as bonus content with no Python equivalent in this course.
