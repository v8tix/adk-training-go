# Troubleshooting: Module 18 (Go) 🛠️

### `workflow.RunNode` fails with "output type ... does not satisfy expected ..." ❌

**Symptom:** calling `workflow.RunNode[SomeStruct](ctx, classifierNode, input)` against a node built with `llmagent.Config.OutputSchema` fails at runtime with an error like `workflow.RunNode: child "classifier" output type map[string]interface {} does not satisfy expected main.SomeStruct`.

**Cause:** confirmed live this module — `workflow.RunNode`'s implementation does a plain Go type assertion on the child node's raw output, with no schema-aware JSON conversion (unlike `workflow.NewFunctionNode`'s input coercion, which does convert). An `OutputSchema`-constrained agent's structured result always arrives at `RunNode` as `map[string]any`, never as a typed struct.

**Fix:** declare the call as `workflow.RunNode[map[string]any](ctx, child, input)` and index the result by key (e.g. `result["sentiment"]`), rather than declaring a matching Go struct as `OUT`.

### The classifier misclassifies a borderline message 🤔

**Cause:** confirmed live this module — a genuinely ambiguous message ("My internet is down, help!") was classified "angry" by both Ollama and Gemini in this course's own probe, when a human reader might call it neutral. This is a model-quality limitation of the classifier prompt and the underlying model, not a bug in the routing code — the `if`/`else` correctly followed whatever the classifier actually returned.

**Fix:** use unambiguous test messages (clearly angry, clearly happy) when verifying the routing logic itself. If classification accuracy on borderline messages matters for your own use case, that's a prompt-engineering or model-choice problem to solve separately from the orchestration pattern this module teaches.
