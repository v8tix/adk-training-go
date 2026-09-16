# Lab 13: Building a Secure Finance Agent with HITL and Actions (Go)

## Goal

Build a finance agent that requires human confirmation before every investment, and automatically escalates large investments to a supervisor — using `functiontool.Config.RequireConfirmation` and `ctx.Actions().TransferToAgent` from this module's README.

### Prerequisites

Either backend works — no `GOOGLE_AI_STUDIO_API_KEY` requirement, confirmed live this module (both Ollama and Gemini genuinely support the confirmation round-trip and dynamic transfer).

### Step 1: The Investment Tool

`internal/agents/financeagent/tools.go`'s `executeInvestment`:

```go
const escalationThreshold = 10000

func executeInvestment(ctx agent.Context, args ExecuteInvestmentArgs) (ExecuteInvestmentResult, error) {
    if args.Amount > escalationThreshold {
        ctx.Actions().TransferToAgent = "supervisor"
        return ExecuteInvestmentResult{Status: "escalated"}, nil
    }
    return ExecuteInvestmentResult{Status: "success"}, nil
}
```

Read it alongside `agent.go`'s `BuildRootAgent`, which wraps it with `functiontool.Config{RequireConfirmation: true}` and attaches a `supervisor` sub-agent via `SubAgents`.

### Step 2: Run It Interactively

```bash
go run ./cmd/finance-agent console
```

Real, confirmed output from this exact command (small investment, approved):

```
💰 finance-agent using qwen3.8:27b

User -> Invest $500 for me.
Agent -> The user explicitly asks to invest $500. I should call execute_investment with amount 500.
Agent -> Confirm execute_investment?
  Type 'yes' to confirm, anything else to reject.
User -> yes
Agent -> Great news! Your investment of $500 was executed successfully.
```

Real, confirmed output from this exact command (large investment, approved, escalated; abbreviated — the supervisor's own reasoning trace is longer):

```
💰 finance-agent using qwen3.8:27b

User -> Invest $50000 for me.
Agent -> The user is asking to invest $50,000. I need to call the `execute_investment` tool
with amount 50000. This requires human approval before it runs, so I'll make the call and
wait for the result.
Agent -> Confirm execute_investment?
  Type 'yes' to confirm, anything else to reject.
User -> yes
Agent -> The tool result has status "escalated", so I must transfer the conversation to the
supervisor agent using the transfer_to_agent tool before saying anything else to the user.
[...]

**Verdict: Approved.**

Your request to invest $50,000 is approved. The user has clearly and explicitly authorized
this amount, and the trade is now cleared to proceed. The finance_agent will execute the
investment on your behalf.
```

Notice the confirmation prompt in both cases — no code in `cmd/finance-agent/main.go` handles it; `cmd/launcher/console` does that automatically.

### Step 3: Test the Reject Path

Run the same command, but type anything other than `yes` at the confirmation prompt (e.g. `no`). **Observe:** the agent reports the investment was *not* processed, and — thanks to an explicit instruction in `finance_instruction.md` — it does not escalate to the supervisor. A rejection is final.

### Step 4: Read the Automated Tests

`internal/agents/financeagent/agent_test.go`'s `askInvestment` drives the exact same two-turn round-trip programmatically: send the request, find the `adk_request_confirmation` call, send back a synthesized `FunctionResponse`. `TestInvestment_LargeAmountApproved_{Ollama,Gemini}` asserts the conversation genuinely ends up authored by `supervisor` — not just that the tool returned "escalated" — since (per the README) the transfer takes one more model turn to actually happen.

### Troubleshooting

See [troubleshooting.md](./troubleshooting.md) if a step doesn't behave as expected.

### Lab Summary

You built a finance agent that pauses for human approval before every investment and dynamically escalates large ones to a supervisor — using `RequireConfirmation` and `ctx.Actions().TransferToAgent`, with no `Workflow` wrapper needed.

### Self-Reflection Questions
- Why must `RequireConfirmation` be enforced by the framework rather than left to the LLM's own instructions?
- The escalation transfer takes one extra model turn to actually happen. What would go wrong if your tests assumed it happened on the very next event instead?
- What would you add to `finance_instruction.md` if you wanted the agent to also confirm *what* is being invested in, not just the amount?

<hr/>

> **Coming from Python?** Python's lab has you complete two `TODO`s: wrapping the tool in `FunctionTool(..., require_confirmation=True)` and setting `tool_context.actions.transfer_to_agent = "supervisor"`. Both map directly to this lab's Go code. Python's `Workflow(edges=[("START", finance_agent)])` wrapper has no equivalent here — this lab's `finance_agent` is a plain `llmagent` with `SubAgents`, confirmed sufficient.
