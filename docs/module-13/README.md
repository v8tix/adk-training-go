# Module 13: Advanced Interactions — Actions & HITL (Go) 🛑

## Theory

### A Tool That Pauses for a Human 🧍

Every custom tool so far has run to completion the second the model calls it. But some actions — moving money, deleting data, sending a message — really shouldn't fire just because a model *decided* to. `functiontool.Config` has a field made exactly for this:

```go
investmentTool, err := functiontool.New(functiontool.Config{
    Name:                "execute_investment",
    Description:         "Executes a long-term investment.",
    RequireConfirmation: true,
}, executeInvestment)
```

Set `RequireConfirmation: true`, and the framework intercepts the very first call — `executeInvestment` doesn't run yet. Instead it emits a special event (a `FunctionCall` named `adk_request_confirmation`) and tells the model "this needs confirmation." Confirmed live: your real handler's first line of code never runs until an actual human answers. 🔒

### Answering the Confirmation ✅❌

The calling app answers that special event with a `FunctionResponse` of the same name, carrying `{"confirmed": true}` or `{"confirmed": false}`:

```go
confirmResp := &genai.Content{
    Role: string(genai.RoleUser),
    Parts: []*genai.Part{{
        FunctionResponse: &genai.FunctionResponse{
            Name:     toolconfirmation.FunctionCallName, // "adk_request_confirmation"
            ID:       confirmCallID,                     // the confirmation call's own ID
            Response: map[string]any{"confirmed": true},
        },
    }},
}
```

Send that on the next turn and the framework resumes the *original* `execute_investment` call — your real handler runs now, with the exact same arguments the model gave the first time. Reject it (`"confirmed": false`), and the handler never runs at all — the call fails with `tool.ErrConfirmationRejected`. Confirmed live: a rejection is enforced by the framework itself, not something your handler code needs to check for. 👍

Building an interactive CLI? You get this for free: `cmd/launcher/console` already knows how to recognize `adk_request_confirmation` events and prompts for yes/no automatically (confirmed by reading its own `hitl.go` source). `cmd/finance-agent` in this module uses the plain launcher, same as every prior module's `cmd/` program, and it just... works. No extra code needed. ✨

### Steering the Runtime with `ctx.Actions()` 🎛️

`agent.Context.Actions()` hands you the current event's `*session.EventActions` — a way for a tool to steer what happens next, beyond its own return value:

```go
func executeInvestment(ctx agent.Context, args ExecuteInvestmentArgs) (ExecuteInvestmentResult, error) {
    if args.Amount > escalationThreshold {
        ctx.Actions().TransferToAgent = "supervisor"
        return ExecuteInvestmentResult{Status: "escalated"}, nil
    }
    return ExecuteInvestmentResult{Status: "success"}, nil
}
```

`SkipSummarization` (used internally by the confirmation mechanism above) tells the framework "don't let the model rewrite this result before showing it to the user." `TransferToAgent` hands the whole conversation to a different agent — `supervisor` here needs no special wrapper, it's a plain `llmagent`, attached via the finance agent's own `SubAgents: []agent.Agent{supervisorAgent}`. No `workflow`/`Workflow` package required — confirmed live, a plain agent-with-sub-agents does the job on its own. 🙌

### A Real Gotcha We Confirmed: Confirmation + Transfer on the Same Call 🔍

Set `ctx.Actions().TransferToAgent` from a plain, unconditional tool (no confirmation gate), and the active agent switches immediately — the very next event comes from the new agent, no model decision involved. But `executeInvestment` above is *also* wrapped with `RequireConfirmation: true`, and that combo behaves differently. Confirmed live: the transfer does **not** happen on the same turn the confirmed call resolves. Instead, the model gets one more turn inside the original agent, sees the "escalated" status, and has to call the framework's own auto-injected `transfer_to_agent` tool itself — *only then* does the active agent actually change. Both Ollama and Gemini pulled this off reliably once instructed to, but it took an explicit prompt line to make it stick (see the Constraints below) — without it, a "thinking" model can just narrate the escalation in text instead of actually calling the tool. Sneaky! 😅

Here's another real one from building this module: a rejected confirmation, if the agent's own instruction doesn't spell out what happens next, can send a model into a weird loop — reject → escalate → supervisor bounces it back → retry → reject again, round and round. `internal/agents/financeagent/prompts/finance_instruction.md` explicitly tells the model a rejection is final, full stop. That line wasn't optional — a generic instruction alone left the model to improvise, and it improvised badly.

### Reading Uploaded Files: `ctx.Artifacts().Load` 📎

The third `ToolContext` capability from this module's theory — reading a file a user uploaded — has a direct construction, even though this module doesn't build or test it (reference only):

```go
func analyzeLogs(ctx agent.Context, args AnalyzeLogsArgs) (AnalyzeLogsResult, error) {
    resp, err := ctx.Artifacts().Load(ctx, args.FileName)
    if err != nil {
        return AnalyzeLogsResult{}, err
    }
    // resp holds the artifact's content as a *genai.Part.
}
```

### Key Takeaways ✅
- `functiontool.Config{RequireConfirmation: true}` pauses a tool for human approval — the real handler never runs until a `FunctionResponse` named `toolconfirmation.FunctionCallName` confirms it, and a rejection is enforced by the framework, not your handler code.
- `cmd/launcher/console` already speaks this confirmation protocol — no custom driving code needed for an interactive CLI.
- `ctx.Actions().TransferToAgent` hands the conversation to a `SubAgents` entry — no `Workflow` wrapper required.
- Combining confirmation and transfer on the same tool call takes one extra model turn before the transfer actually happens — a real, confirmed sequence, not a guess.
- `ctx.Artifacts().Load` is the direct equivalent for reading uploaded files — a real construction even where a lab doesn't build one.

<hr/>

> **Coming from Python?** 🐍 `functiontool.Config.RequireConfirmation` plays the same role as Python's `FunctionTool(fn, require_confirmation=True)`; `ctx.Actions()` is the same object as `tool_context.actions`, with the same `transfer_to_agent`/`skip_summarization` fields; `ctx.Artifacts().Load` is `tool_context.load_artifact`. Python's own README presents its `Workflow` container as a stylistic choice, not a requirement, for this lab — confirmed true in Go too: a plain `SubAgents`-equipped `llmagent` is all you need.
