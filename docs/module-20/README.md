# Module 20: Cyclic Workflows — Iteration and Self-Correction (Go)

## Theory

### Iteration Is Just a Loop

Module-18 established `workflow.NewDynamicNode` and `workflow.RunNode` for imperative, code-driven orchestration — a dynamic node's body is ordinary Go code, free to call other nodes in whatever order it decides. Iteration needs nothing new on top of that: a plain `for` loop inside the same kind of function body, calling `RunNode` repeatedly, is the entire mechanism. There's no separate "Loop" construct to learn.

```go
refinementWorkflow := workflow.NewDynamicNode("refinement_workflow",
    func(ctx agent.Context, topic string, emit func(*session.Event) error) (string, error) {
        currentStory, err := workflow.RunNode[string](ctx, writerNode, fmt.Sprintf("Topic: %s", topic))
        if err != nil {
            return "", err
        }

        for i := 0; i < maxIterations; i++ {
            feedback, err := workflow.RunNode[string](ctx, criticNode, currentStory)
            if err != nil {
                return "", err
            }
            if strings.Contains(feedback, "APPROVED") {
                break
            }

            currentStory, err = workflow.RunNode[string](ctx, refinerNode,
                fmt.Sprintf("WORK:\n%s\n\nFEEDBACK:\n%s", currentStory, feedback))
            if err != nil {
                return "", err
            }
        }

        return currentStory, nil
    },
    workflow.NodeConfig{},
)
```

Every capability a loop needs — a hard iteration cap, an early-exit condition, ordinary error handling via Go's own `if err != nil` — is just Go, not a framework feature to configure.

### A Named Safety Cap

`const maxIterations = 3` — a model-driven exit condition (the critic's own judgment) needs a hard backstop, since nothing guarantees the critic ever says "APPROVED." Confirmed live: this probe's own run converged in 2 iterations well under the cap, but the cap exists specifically for the runs that wouldn't.

### A Genuinely Important, Confirmed Nuance: Where the Loop's Real Answer Lives

The loop's `return currentStory, nil` becomes the dynamic node's own output — but that value doesn't show up the way a chat response does. Confirmed live by inspecting every event in a real run directly: the writer, critic, and refiner's own turns each produce ordinary chat-content events (`Author: "writer"`/`"critic"`/`"refiner"`, real `Content`, `Output: nil`). The loop's actual return value arrives on a *different* event entirely — one authored by the root workflow agent's own name (`"EssayRefiner"`, this package's `workflowagent.Config.Name`), with `Content: nil` and `Output` set to the final story.

This matters concretely: in a real run, the *last* chat-content event was the critic's own final `"APPROVED"` reply — not the story. Code that wants the clean final result (a test, a programmatic caller) needs to read `Output` from the event authored by the workflow's own name, not assume "the last piece of visible text" is the answer.

### How This Plays Out on the Console

Confirmed by reading `cmd/launcher/console/console.go` directly: the console launcher prints every chat-content event's text as it streams, and *separately* falls back to rendering a content-less event's `Output` only when nothing was ever printed as chat text at all. Since this loop always produces real chat content (the draft, each verdict, each rewrite), that fallback path never fires — the polished final essay is visible earlier in the same continuous stream, not specially called out as a distinct "final answer." A console session genuinely shows the whole iteration unfold in real time, which is exactly the point of this lesson — just don't expect the very last line to be the answer.

### The Graph's Shape

```mermaid
flowchart TD
    START([START]) --> workflow{{refinement_workflow}}
    workflow -.->|"imperative RunNode call"| writer[writer]
    workflow -.->|"imperative RunNode call,<br/>looped up to maxIterations"| critic[critic]
    workflow -.->|"imperative RunNode call,<br/>looped up to maxIterations"| refiner[refiner]
```

One real edge; the loop itself lives entirely inside `refinement_workflow`'s own function body, the same dashed-arrow convention module-18 used for the same reason. This is also this course's first case of `RunNode` calling the *same* node repeatedly within one dynamic node's body, rather than routing to a different node each time (module-18 only ever called each node once) — confirmed by reading the scheduler directly that repeated calls to the same child node are handled correctly, each tracked by its own auto-incrementing run id, with no collision.

### What Happens If the Critic Never Approves

If the loop reaches `maxIterations` without the critic ever replying "APPROVED," it simply exits and returns whatever the refiner's last rewrite was — there's no separate error or flag distinguishing "approved" from "gave up at the cap." That's a deliberate simplification for this lab, not an oversight: a caller that needs to tell the two cases apart would return an additional status value alongside the story.

### Key Takeaways
- Iteration needs no new SDK construct: a plain Go `for`/`while` loop inside a `workflow.NewDynamicNode`'s body, calling `workflow.RunNode` repeatedly, is the entire mechanism.
- Always cap a model-driven loop with a named safety constant — nothing guarantees the exit condition fires on its own.
- The loop's real return value arrives on the terminal event authored by the root workflow agent's own name, with `Output` set and `Content` nil — not on whatever chat-content event happens to come last.
- The console launcher streams every intermediate chat turn live; it only falls back to rendering a content-less event's `Output` when nothing was ever streamed as text, which doesn't happen for a loop like this one.

<hr/>

> **Coming from Python?** Python's `for i in range(5): ... await ctx.run_node(...)` inside an `@node(rerun_on_resume=True)` function maps directly onto Go's `for i := 0; i < maxIterations; i++ { ... workflow.RunNode(...) }` inside a `workflow.NewDynamicNode` — the same mechanism module-18 already established, with `rerun_on_resume`'s Go equivalent already set automatically for `LlmAgent` nodes (confirmed in module-19). Python's lab notes `ctx.run_node()`'s second argument is positional, not a keyword; Go's `RunNode` has no keyword-argument mechanism at all, so this specific caution simply doesn't apply.
