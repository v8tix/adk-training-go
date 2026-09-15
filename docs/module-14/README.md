# Module 14: Integrating Third-Party Tools (Go)

## Theory

### A Third-Party Tool Is Just a Function

Every custom tool since module-9 has been a Go function wrapped with `functiontool.New`. Nothing changes when the function's own logic happens to call into a package someone else wrote and maintains, instead of code you wrote yourself:

```go
import gowiki "github.com/trietmn/go-wiki"

func lookupWikipedia(_ agent.Context, args LookupWikipediaArgs) (LookupWikipediaResult, error) {
    summary, err := gowiki.Summary(args.Query, 5, -1, false, true)
    if err != nil {
        return LookupWikipediaResult{Status: "error", Error: err.Error()}, nil
    }
    return LookupWikipediaResult{Status: "success", Summary: summary}, nil
}
```

```go
wikipediaTool, err := functiontool.New(functiontool.Config{
    Name:        "lookup_wikipedia",
    Description: "Looks up a topic on Wikipedia and returns a short summary.",
}, lookupWikipedia)
```

That's the entire integration. `github.com/trietmn/go-wiki` is a real, independently-maintained Go package (not part of the ADK SDK or this course) that does the actual Wikipedia API work — the tool wrapping it looks exactly like `calculator`'s `add` from module-9.

### Why No Dedicated Wrapper Class Exists

`tool.Tool` is a plain interface: `Name`, `Description`, `IsLongRunning`, plus `Declaration`/`Run` for a runnable one. `functiontool.New` builds a `tool.Tool` from any function whose shape matches `func(agent.Context, TArgs) (TResults, error)` — it has no idea, and no need to know, whether that function's body is five lines you wrote or a single call into someone else's library. There's nothing to inspect or translate beyond the function's own argument and result types, which Go's type system already describes precisely.

This is a real, confirmed finding, not an assumption: the pinned `google.golang.org/adk/v2` SDK has no wrapper type for adapting a third-party library's own tool-object model into ADK's shape, and none is needed, because a Go function from a third-party package presents the exact same shape `functiontool.New` already handles.

### A Real Gotcha, Confirmed Live

`github.com/trietmn/go-wiki`'s underlying Wikipedia API calls fail with a real error (`unable to fetch the results`) if you don't set a distinctive User-Agent — Wikimedia rate-limits the package's generic default, since it's shared by every user of the package. The fix is one line, called once before any request:

```go
func init() {
    gowiki.SetUserAgent("your-app-name/1.0 (contact-info)")
}
```

`internal/agents/factfinder/tools.go` calls this in its own package `init()`, confirmed live to be honored by later calls made from a different function — the same "set once, anywhere before first use" pattern a package-level fix like this always needs.

### `google_search` Still Can't Mix With It

The same restriction from modules 9 and 12 applies here unchanged: a wrapped or hand-written custom function tool still counts as a function tool for the Gemini API's "no mixing built-in and custom tools" rule. `lookup_wikipedia` can sit in the same `Tools` list as any other function tool, but not alongside `geminitool.GoogleSearch{}` without also setting `IncludeServerSideToolInvocations` (module-12) — the same two options apply: combine them via that flag, or use sequential composition.

### Looking Further: MCP

If the goal is publishing a tool for *other* frameworks to consume too — not just this one Go codebase — `google.golang.org/adk/v2/tool/mcptoolset` is the real, SDK-supported path: MCP (Model Context Protocol) is an open, framework-agnostic standard for exposing tools to any MCP-aware client, closer in spirit to LangChain's "one tool, many consumers" model than the direct function-wrapping this module uses. Building one is this course's own later lesson (modules 27-28) — worth knowing it exists, not worth building here.

### Key Takeaways
- A third-party Go package's function wraps into a tool exactly the same way a hand-written one does — `functiontool.New` draws no distinction.
- No dedicated adapter class exists in the Go SDK for this, and none is needed — Go's tool model has no separate object shape to translate from.
- A package-level fix like a custom User-Agent belongs in the package's own `init()`, confirmed live to be honored by handler-context calls made later.
- The built-in-tool mixing restriction from modules 9/12 applies unchanged to a wrapped third-party tool.
- `tool/mcptoolset` is the real path for publishing a tool to a wider audience than one codebase — covered in this course's own later modules.

<hr/>

> **Coming from Python?** Python needs `google.adk.integrations.langchain.LangchainTool` specifically because a LangChain tool isn't a plain function — it's its own object model (a `BaseTool` subclass with `name`/`description`/`args_schema` attributes) that has to be inspected and translated into ADK's shape. Go has no such translation problem: a third-party Go package's exported function already has the exact shape `functiontool.New` expects, so the wrapper class Python needs simply has nothing to do here.
