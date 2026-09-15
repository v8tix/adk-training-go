# Lab 1 Challenge: Your First Interaction

## Goal

The goal of this first lab is to familiarize yourself with the available resources for learning and using the Google Agent Development Kit (ADK) for Go. You won't be writing any code yet; instead, you'll be exploring the official documentation and code repository to understand where to find information and inspiration.

## Step 1: Navigate the Official Documentation

The official ADK-for-Go documentation is the most important resource you will use. It's your primary source for tutorials, guides, and API references.

1. **Open the Documentation:** In your web browser, go to `https://adk.dev/get-started/`.
2. **Explore the "Get Started" Section:** Read through `Installation` and the Go-specific quickstart at `https://adk.dev/get-started/go/`. Note the `go get google.golang.org/adk/v2` install command and the Go 1.25+ requirement.
3. **Browse the Graph and Workflow Sections:** Look at `https://adk.dev/graphs/` and `https://adk.dev/workflows/` — this is the heart of ADK 2.0, showing how you compose nodes and edges into a workflow.

**Key Takeaway:** The documentation is your best friend. Ensure you are looking at the **Go** pages specifically — Go has its own `Runner`-equivalent API (package `runner`), separate from the higher-level `cmd/launcher` app shell the quickstart samples use.

## Step 2: Discover the Official Code Repository

The Go SDK is open source, with its own repository on GitHub. This repository contains the source code, issue tracker, and, most importantly, a wealth of examples.

1. **Find the Repository:** Go to `github.com/google/adk-go`.
2. **Verify the Version:** Check the latest release tags. You should see `v2.4.0` or higher.
3. **Explore the `examples` directory:** The closest official runnable example is `examples/workflow/basic`.
4. **Examine the Code:** Open `examples/workflow/basic/main.go` and read through it. Try to connect what you see to the concepts you skimmed in Step 1 (`workflow.NewFunctionNode`, `workflow.Chain`, `workflowagent.New`).

***Scavenger Hunt!***
> Find the `examples/workflow/basic` sample and identify what actually runs the agent programmatically. Hint: this particular sample doesn't call the `runner` package directly — it goes through a higher-level app shell instead. Look at the last few lines of `main()`.

**Key Takeaway:** The official examples are the best place to find working code that you can learn from and adapt for your own projects.

## Step 3: Understand the Community and Support Channels

1. **Issues Tab:** On `github.com/google/adk-go`, click on the "Issues" tab. This is where developers report bugs and request new features. Browsing through the issues can give you insight into the current state of the project and common problems users face.
2. **Discussions Tab:** If available, the "Discussions" tab is a place for community conversations, questions, and sharing ideas.

## Lab Summary

Congratulations, you've completed your first lab! You now know:

* How to navigate the official ADK-for-Go documentation to find guides and references.
* Where to find official, working Go code examples in the GitHub repository.
* Where to look for community support and project updates.

In the next module, you will use this knowledge to set up your own local Go development environment and prepare for building your first agent.

## Self-Reflection Questions

- Why is it important to have official documentation and code examples for a framework like the ADK, especially for an SDK as new as Go's?
- Based on the file names in the `examples` directory, what are some of the advanced capabilities you think the Go SDK might have?
- How can community support channels like GitHub Issues and Discussions accelerate your learning process?

<hr/>

### Looking for the solution?

Hint: look at `examples/workflow/basic/main.go` in `github.com/google/adk-go`, specifically the last few lines of `main()` — that's the `cmd/launcher` call you're looking for. (For running an agent directly from your own code instead of via a launcher-built app, see package `runner` — `runner.NewInMemory` + `Run`.)
