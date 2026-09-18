# Lab 28: Building a "Shopping Cart" MCP Server (Go) 🛠️🔌

## Goal

Build your own standalone MCP server from scratch — a stateful shopping cart — then connect an ADK agent to it as a client, using the exact same `mcptoolset`/`mcp.CommandTransport` mechanism module-27 already proved against a third-party server.

### Prerequisites

None beyond what module-27 already needed — this lab runs entirely on the local Ollama default, no cloud credentials required. Node.js/`npx` aren't needed either: this module's server is a Go program, not a third-party Node package.

## Lab Tasks

### 1. Read `cmd/cart-mcp-server/main.go`

A standalone MCP server with **zero dependency on `google.golang.org/adk/v2`** — matching Python's own `cart_server.py` being a plain script, not an ADK program. `cart{mu sync.Mutex, items []string}` holds the state; `addItem`/`view` are its two tool handlers, registered via `mcp.AddTool(server, &mcp.Tool{...}, handler)`. Notice `AddItemArgs`/`AddItemResult` are just plain Go structs with `jsonschema` tags — no hand-written schema anywhere, unlike Python's lab.

### 2. Read `cmd/cart-mcp-server/main_test.go`

Pure unit tests directly on `cart.addItem`/`cart.view` — no MCP protocol involved at all, just Go function calls — proving items accumulate in order, a fresh cart starts empty, and concurrent `addItem` calls are genuinely race-safe (`-race` is what would actually catch a missing lock here).

### 3. Read `internal/agents/mcpcart/agent.go`

`BuildRootAgent(llmModel, repoRoot)` launches `cmd/cart-mcp-server` (via `go run`, so no separate build step) as its MCP server subprocess, wired through `mcptoolset.New` + `mcp.CommandTransport` — the exact mechanism module-27's own `mcpfilesystem` established. `RepoRoot()` resolves the repository root via `go env GOMOD`'s own directory, not a relative path guess — a direct lesson from a real bug module-27's own review caught (a cwd-relative path silently broke under `go test`, which runs with a different working directory than `go run`).

### 4. Run it — console mode, entirely locally 🖥️

```bash
go run ./cmd/shopping-agent console
```

No `.env`, no API key needed. Real, confirmed output from this exact command (thinking-model reasoning trimmed for readability) — notice the server's own `[Server]: ...` log lines interleaved with the conversation, forwarded via `serverCmd.Stderr = os.Stderr`:

```
🛒 shopping-agent using qwen3.8:27b

User -> Please add milk to my cart.
[Server]: Starting Shopping Cart MCP Server...
[Server]: Waiting for a client to connect...
[Server]: added "milk" to the cart
Agent -> Milk has been added to your cart! Let me know if there's anything else you'd like to add.

User -> Also add eggs.
[Server]: added "eggs" to the cart
Agent -> Eggs have been added to your cart! Your cart now contains:

1. Milk
2. Eggs

Anything else you'd like to add?

User -> What is in my shopping cart?
[Server]: client viewed the cart (2 item(s))
Agent -> Your shopping cart currently contains:

1. Milk
2. Eggs

Would you like to add or remove anything?
```

That state — "milk" and "eggs" both showing up in the final `view_cart` call — lives entirely in the server subprocess, genuinely carried across three separate turns of the conversation, not remembered by the agent itself.

### 5. Read `internal/agents/mcpcart/agent_test.go`

`TestBuildRootAgent_Constructs` proves construction never launches the subprocess (MCP sessions connect lazily — the same finding module-27 already confirmed). `TestShoppingCart_AddsAndViewsItems_{Ollama,Gemini}` drives the exact three-turn conversation above through a real `runner.New`, asserting both real item names appear in the final answer — the real, structural proof of cross-process state, not just eyeballing console output.

## Self-Reflection Questions 🤔
- `cart`'s state is a plain in-memory `[]string` in one server process. What would break if two learners ran their own `shopping-agent` at the same time, each launching their own server subprocess? What would break differently if they somehow shared *one* server instance?
- `mcp.AddTool`'s automatic schema inference means your Go struct *is* the schema. What would you need to change on the Go side if you wanted to rename a JSON field without changing your Go field name?
- Why does `RepoRoot()` use `go env GOMOD` instead of a relative path like `filepath.Join(thisFileDir, "..", "..", "..")`?
- This server has no way to *remove* an item from the cart. If you added a `remove_item_from_cart` tool, would `mcp.AddTool`'s automatic schema inference need anything from you beyond a new args struct and handler?

<hr/>

### Looking for the solution? 🔍

Hint: read `cmd/cart-mcp-server/main.go` and `internal/agents/mcpcart/agent.go` for the real mechanism — one standalone server with two typed tool handlers, and one client agent launching it as a subprocess.
