# Lab 1 Challenge: Your First Interaction 🗺️

## Goal

No code today! This lab is all about getting your bearings: exploring the official ADK-for-Go docs and code repo so you know exactly where to look when you need answers later.

## Step 1: Navigate the Official Documentation 📖

The official ADK-for-Go docs are your #1 resource — tutorials, guides, API references, all of it.

1. **Open the docs:** head to `https://adk.dev/get-started/`.
2. **Check out "Get Started":** read `Installation` and the Go quickstart at `https://adk.dev/get-started/go/`. Note the `go get google.golang.org/adk/v2` command and the Go 1.25+ requirement.
3. **Browse Graph and Workflow sections:** `https://adk.dev/graphs/` and `https://adk.dev/workflows/` — this is the heart of ADK 2.0, showing how nodes and edges come together into a workflow.

💡 **Key takeaway:** make sure you're on the **Go** pages specifically. Go has its own `Runner`-equivalent API (package `runner`), separate from the higher-level `cmd/launcher` app shell the quickstart samples use.

## Step 2: Discover the Official Code Repository 💻

The Go SDK is open source with its own GitHub repo — source code, issue tracker, and a ton of examples.

1. **Find it:** `github.com/google/adk-go`.
2. **Check the version:** look for release tag `v2.4.0` or higher.
3. **Explore `examples`:** the closest official runnable example is `examples/workflow/basic`.
4. **Read the code:** open `examples/workflow/basic/main.go`. Try connecting what you see to what you skimmed in Step 1 — `workflow.NewFunctionNode`, `workflow.Chain`, `workflowagent.New`.

🕵️ **Scavenger Hunt!**
> Find the `examples/workflow/basic` sample and spot what actually runs the agent programmatically. Hint: this sample doesn't call `runner` directly — it goes through a higher-level app shell instead. Peek at the last few lines of `main()`.

💡 **Key takeaway:** the official examples are your best source of working code to learn from (and steal for your own projects 😉).

## Step 3: Find the Community 🙋

1. **Issues tab:** on `github.com/google/adk-go`, click "Issues" — bugs, feature requests, and a good read on the project's current state.
2. **Discussions tab:** if it's there, it's the spot for community Q&A and idea-sharing.

## Lab Summary 🎉

Nice work, lab 1 done! You now know:

* How to navigate the official ADK-for-Go docs.
* Where to find official, working Go examples in the GitHub repo.
* Where to look for community support and project updates.

Up next: setting up your own local Go dev environment so you're ready to build your first agent.

## Self-Reflection Questions 🤔

- Why does having solid official docs and examples matter so much for a framework as new as the Go SDK?
- Just from the file names in `examples`, what advanced capabilities do you think the Go SDK might have?
- How can GitHub Issues and Discussions speed up your own learning?

<hr/>

### Looking for the solution? 🔍

Hint: check `examples/workflow/basic/main.go` in `github.com/google/adk-go`, specifically the last few lines of `main()` — that's the `cmd/launcher` call you're after. (Want to run an agent directly from your own code instead of via a launcher-built app? Look at package `runner` — `runner.NewInMemory` + `Run`.)
