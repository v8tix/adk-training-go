# Troubleshooting: Module 13 (Go)

### The confirmation prompt never appears

**Cause:** `functiontool.Config.RequireConfirmation` isn't actually `true` on the tool passed to `Tools` — the LLM's own instructions have no effect on this; it's a framework-level gate, not something a prompt can request or skip.

**Fix:** confirm `RequireConfirmation: true` is set in the `functiontool.Config` passed to `functiontool.New`, not just documented in the tool's description.

### Approving a large investment never reaches `supervisor`

**Cause:** `finance_instruction.md` doesn't explicitly tell the model to call the transfer tool itself on an "escalated" status. Without that line, a model can narrate the escalation in text without ever actually calling `transfer_to_agent` — confirmed live during this module's own build.

**Fix:** make sure the finance agent's instruction explicitly says to call `transfer_to_agent` on an "escalated" result, not just describe that escalation should happen.
