# Module 13: Advanced Interactions — Actions & HITL (Go)

## Theory

### A Tool That Pauses for a Human

Every custom tool so far has run to completion the moment the model calls it. Some actions — moving money, deleting data, sending a message — shouldn't run just because a model decided to call a function. `functiontool.Config` has a field for exactly this:

```go
investmentTool, err := functiontool.New(functiontool.Config{
    Name:                "execute_investment",
    Description:         "Executes a long-term investment.",
    RequireConfirmation: true,
}, executeInvestment)
```

With `RequireConfirmation: true`, the framework intercepts the very first call: it never runs `executeInvestment` yet. Instead it emits a special event — a `FunctionCall` named `adk_request_confirmation` — and returns a "requires confirmation" result to the model. Confirmed live: your real handler's first line of code never executes until a human answers.

### Answering the Confirmation

The calling application answers that special event with a `FunctionResponse` of the same name, carrying `{"confirmed": true}` or `{"confirmed": false}`:

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

Send that as the next turn, and the framework resumes the *original* `execute_investment` call — this time your real handler runs, with the exact same arguments the model originally supplied. Reject it (`"confirmed": false`), and the handler never runs at all — the call fails with `tool.ErrConfirmationRejected` instead, confirmed live: a rejected confirmation is enforced by the framework itself, not something your own handler code has to check for.

If you're building an interactive CLI, you don't need to write any of this by hand: `cmd/launcher/console` already recognizes `adk_request_confirmation` events and prompts for yes/no automatically — confirmed by reading its own `hitl.go` source. `cmd/finance-agent` in this module uses the plain launcher, exactly like every prior module's `cmd/` program, and gets a working confirmation prompt for free.

### Steering the Runtime with `ctx.Actions()`

`agent.Context.Actions()` returns the current event's `*session.EventActions` — a way for a tool to influence what happens next, beyond just its return value:

```go
func executeInvestment(ctx agent.Context, args ExecuteInvestmentArgs) (ExecuteInvestmentResult, error) {
    if args.Amount > escalationThreshold {
        ctx.Actions().TransferToAgent = "supervisor"
        return ExecuteInvestmentResult{Status: "escalated"}, nil
    }
    return ExecuteInvestmentResult{Status: "success"}, nil
}
```

`SkipSummarization` (used internally by the confirmation mechanism above) tells the framework not to let the model rewrite a tool's result before showing it to the user. `TransferToAgent` hands the conversation to a different agent — `supervisor` here needs no special wrapper: it's a plain `llmagent`, attached via the finance agent's own `SubAgents: []agent.Agent{supervisorAgent}`. No `workflow`/`Workflow` package is required for this — confirmed live, a plain agent-with-sub-agents is sufficient on its own.

### A Confirmed, Non-Obvious Interaction: Confirmation + Transfer on the Same Call

Setting `ctx.Actions().TransferToAgent` from an *unconditional* tool (no confirmation gate) switches the active agent immediately — the very next event is authored by the new agent, with no model decision involved. But `executeInvestment` above is *also* wrapped with `RequireConfirmation: true`, and that combination behaves differently: confirmed live, the transfer does **not** happen on the same turn the confirmed call resolves. Instead, the model gets one more turn inside the original agent, sees the "escalated" status, and must itself call the framework's own auto-injected `transfer_to_agent` tool — only *then* does the active agent actually change. Both Ollama and Gemini reliably did this once instructed to, but it took an explicit prompt line to make it reliable (see the Constraints below) — without it, a "thinking" model can just narrate the escalation in text instead of actually calling the tool.

A related, real prompt-design finding: a rejected confirmation, if the agent's own instruction doesn't say what to do next, can send a model into escalating anyway and looping (reject → escalate → supervisor sends it back → retry → reject again). `internal/agents/financeagent/prompts/finance_instruction.md` explicitly tells the model a rejection is final — this is a genuine lesson from building this module: the instruction file needed an explicit line for it, since a generic instruction alone left the model to improvise.

### Reading Uploaded Files: `ctx.Artifacts().Load`

The third `ToolContext` capability from this module's theory — reading a file a user uploaded — has a direct construction, though this module doesn't build or test it — it's included here for reference only:

```go
func analyzeLogs(ctx agent.Context, args AnalyzeLogsArgs) (AnalyzeLogsResult, error) {
    resp, err := ctx.Artifacts().Load(ctx, args.FileName)
    if err != nil {
        return AnalyzeLogsResult{}, err
    }
    // resp holds the artifact's content as a *genai.Part.
}
```

### Key Takeaways
- `functiontool.Config{RequireConfirmation: true}` pauses a tool for human approval — the real handler never runs until a `FunctionResponse` named `toolconfirmation.FunctionCallName` confirms it, and a rejection is enforced by the framework, not by handler code.
- `cmd/launcher/console` already speaks this confirmation protocol — an interactive CLI needs no custom driving code for it.
- `ctx.Actions().TransferToAgent` hands the conversation to a `SubAgents` entry — no `Workflow` wrapper required.
- Combining confirmation and transfer on the same tool call takes one extra model turn before the transfer actually happens — a real, confirmed sequence, not an assumption.
- `ctx.Artifacts().Load` is the direct equivalent for reading uploaded files, a real construction even where a lab doesn't build one.

<hr/>

> **Coming from Python?** `functiontool.Config.RequireConfirmation` plays the same role as Python's `FunctionTool(fn, require_confirmation=True)`; `ctx.Actions()` is the same object as `tool_context.actions`, with the same `transfer_to_agent`/`skip_summarization` fields; `ctx.Artifacts().Load` is `tool_context.load_artifact`. Python's own README presents its `Workflow` container as a stylistic choice, not a requirement, for this lab — confirmed true in Go as well, a plain `SubAgents`-equipped `llmagent` is enough.
