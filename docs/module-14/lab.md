# Lab 14: Integrating a Third-Party Wikipedia Tool (Go)

## Goal

Build a "fact-finder" agent that looks up information on Wikipedia, using a real, independently-maintained third-party Go package — no cloud account beyond what earlier modules already need.

### Step 1: The Tool

`internal/agents/factfinder/tools.go`'s `lookupWikipedia` calls `github.com/trietmn/go-wiki`'s `Summary` function:

```go
func lookupWikipedia(_ agent.Context, args LookupWikipediaArgs) (LookupWikipediaResult, error) {
    summary, err := gowiki.Summary(args.Query, 5, -1, false, true)
    if err != nil {
        return LookupWikipediaResult{Status: "error", Error: err.Error()}, nil
    }
    return LookupWikipediaResult{Status: "success", Summary: summary}, nil
}
```

Notice the package's own `init()`, right above it — that's where the User-Agent fix from this module's README lives.

### Step 2: The Agent

`internal/agents/factfinder/agent.go`'s `BuildRootAgent` wraps it via `functiontool.New`, exactly like `calculator.BuildRootAgent` from module-9:

```go
wikipediaTool, err := functiontool.New(functiontool.Config{
    Name:        "lookup_wikipedia",
    Description: "Looks up a topic on Wikipedia and returns a short summary.",
}, lookupWikipedia)
```

### Step 3: Run It

```bash
go run ./cmd/fact-finder console
```

Real, confirmed output from this exact command (the local model's own reasoning text runs directly into its answer with no separator — `cmd/launcher/console` prints every text part unconditionally, without filtering `Thought` parts, a known SDK rendering quirk covered in earlier modules, not something this module's own code can fix):

```
📚 fact-finder using qwen3.8:27b

User -> Who was Marie Curie?
Agent -> The user is asking a factual, encyclopedia-style question about Marie Curie.
According to my instructions, I should use the lookup_wikipedia tool to answer this
rather than from memory alone.The tool returned a successful result with a summary
about Marie Curie. I can now use this information to answer the user's question.
**Marie Curie** (born Maria Salomea Skłodowska; 7 November 1867 – 4 July 1934) was a
Polish and naturalised-French physicist and chemist. Here are some highlights of her
life and achievements:

- **Nobel Prizes:** She won the 1903 Nobel Prize in Physics (shared with her husband,
  Pierre Curie) and the 1911 Nobel Prize in Chemistry.
- **Records:** She was the first woman to win a Nobel Prize, the first person to win
  two, and the only person to win in two different scientific fields.
```

No `GOOGLE_AI_STUDIO_API_KEY` needed — this runs entirely on the local Ollama default, confirmed live, since a plain function tool needs no built-in-tool capability. If you want to see the tool actually verify it searched rather than guessed, drive the agent directly through `runner.Run` instead of the console launcher, the way `agent_test.go`'s `askFactFinder` does — it filters `Thought` parts and reads the tool's own `FunctionResponse`, exactly the pattern this repo has used to test every agent since module-9.

### Step 4: Confirm the Tool Actually Ran

`internal/agents/factfinder/agent_test.go`'s `assertLooksUpWikipedia` checks the tool's own `FunctionResponse` — not just the final text — asserting `status == "success"` and the summary genuinely mentions the queried subject. This matters because a model could otherwise answer a well-known question like this one from its own training data without ever calling the tool.

### Troubleshooting

See [troubleshooting.md](./troubleshooting.md) if a step doesn't behave as expected.

### Lab Summary

You integrated a real, third-party Go package into an ADK agent using the exact same `functiontool.New` pattern every custom-tool module since module-9 already uses — no special adapter needed.

### Self-Reflection Questions
- Why does `functiontool.New` need no information about where `lookupWikipedia`'s logic comes from — your own code or a third-party package?
- `TestLookupWikipedia`'s "nonsense query" test asserts a structured error result, not a Go error. Why does that distinction matter for what the model can do with the outcome?
- What would change in `tools.go` if you wanted to swap `github.com/trietmn/go-wiki` for a different Wikipedia client library?

<hr/>

> **Coming from Python?** Python's lab has you instantiate a `WikipediaAPIWrapper`, wrap it in `WikipediaQueryRun`, then wrap *that* in `LangchainTool` — three layers, because LangChain's tool object needs translating into ADK's shape. This lab has one layer: a Go function that calls the third-party package directly, wrapped with the same `functiontool.New` every earlier module already uses.
