# Module 11: Enterprise Integration with a Declarative Tool (Go)

## Theory

### Declaring a Tool Instead of Writing One

Every custom function tool so far has been a Go function you write by hand, wrapped with `functiontool.New`. That works well when you know the parameters at compile time. But what if you want to expose *any* REST operation as a tool, described by data rather than code — a name, a URL, a list of parameters — and get a working tool back without writing a wrapper function for each one? That's what this module builds.

### A Spec, Not a Function

`internal/infrastructure/openapitool.OperationSpec` describes one REST operation declaratively:

```go
var frankfurterSpec = openapitool.OperationSpec{
    OperationID: "get_latest_rates",
    Summary:     "Get latest currency exchange rates.",
    BaseURL:     "https://api.frankfurter.dev/v1",
    Path:        "/latest",
    Parameters: []openapitool.ParamSpec{
        {Name: "amount", Type: "number", Description: "The amount to convert", Required: true},
        {Name: "from", Type: "string", Description: "The 3-letter currency code to convert from", Required: true},
        {Name: "to", Type: "string", Description: "The 3-letter currency code to convert to", Required: true},
    },
}
```

`openapitool.NewToolset("frankfurter", frankfurterSpec)` turns that data into a working tool — no per-endpoint Go function to write.

### Building a Tool Without a Compile-Time Struct

`functiontool.New`'s generics need a Go struct for `TArgs`, known at compile time. A spec-driven tool doesn't have one — the parameter list is data, decided at runtime. So `openapitool` builds a tool by hand instead: implementing the same small method set every tool needs (`Name`, `Description`, `Declaration`, `Run`), with `Declaration` assembling its JSON Schema from the spec's `Parameters` list, and `Run` making the real HTTP call and decoding the response into a plain `map[string]any` — since, like the parameters, the response shape isn't known at compile time either.

The real call goes through [kawa](https://github.com/v8tix/kawa), a typed HTTP-call library already used in this repo's own module-7 bonus path. The response type here has to be a named `map[string]any`, not a struct — the whole point of this package is that a response's real shape depends on which API the caller points it at, so there's no fixed field set to declare up front. That turns out to be exactly the case kawa's strict decoding (which normally rejects an unrecognized JSON field on a struct target) doesn't apply to: a map has no fixed fields to be "unrecognized" against, so any shape decodes cleanly.

### One Value, Many Tools: `tool.Toolset`

`llmagent.Config` has a `Toolsets []tool.Toolset` field, separate from the plain `Tools` field you've used since module-9 — built for exactly this shape: one value that produces several tools. `openapitool.Toolset` implements it, so an agent attaches the whole thing in one line:

```go
toolset, _ := openapitool.NewToolset("frankfurter", frankfurterSpec)
llmagent.New(llmagent.Config{
    // ...
    Toolsets: []tool.Toolset{toolset},
})
```

Adding a second operation to the same toolset is just adding a second `OperationSpec` to the `NewToolset` call — no second wrapper function.

### Handling a Real API's Errors

A live REST API can fail in a few different ways, and they call for different responses. Confirmed against the real Frankfurter API: a bad currency code returns a normal HTTP response — `404` with `{"message":"not found"}` — which is exactly the kind of thing the model should read and explain, not something that should crash the agent. A genuine network failure is different: there's nothing for the model to reason about, so that becomes a real Go `error` instead. `openapitool`'s `Run` makes that split explicit, the same distinction module-9's divide-by-zero handling makes.

`Run` also checks for a missing required parameter *before* calling the real API at all — caught in review, since a local model occasionally omits one, and silently calling the API without it can produce a wrong answer instead of an obvious error (Frankfurter itself defaults a missing `from` to EUR rather than rejecting the request). That check produces the same kind of structured result as an API error, so the model has one consistent shape to read regardless of which of the two problems occurred.

### Key Takeaways
- A tool can be built from data (a spec) instead of a hand-written function — useful whenever the parameter set isn't known until runtime.
- The framework's tool-dispatch contract is just a small method set (`Name`, `Description`, `Declaration`, `Run`) — `functiontool.New` is one convenient way to satisfy it, not the only way.
- `llmagent.Config.Toolsets` attaches one value that produces many tools, distinct from `Tools`, which takes tools one at a time.
- A real API's HTTP-level error response belongs in a structured result the model can explain; a genuine network failure belongs in a Go `error`.
- A named map type (not a struct) sidesteps kawa's strict unrecognized-field decoding entirely — useful whenever a response's real shape isn't known until runtime.

<hr/>

> **Coming from Python?** `openapitool.NewToolset` plays the same role as Python's `OpenAPIToolset`: a spec goes in, working tools come out, no wrapper function per endpoint. There's no packaged equivalent in this Go SDK, though — confirmed by an exhaustive search of `google.golang.org/adk/v2` and its dependencies, this is a real, structural gap, not a naming difference. `openapitool` closes it using the same `tool.Tool`/`tool.Toolset` primitives the SDK already exposes, working from a small Go struct instead of a parsed OpenAPI JSON/YAML document — the same scope Python's own lab uses too, since it also builds its spec as a literal dict rather than loading a real spec file.
