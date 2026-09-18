# Module 28: Building a Custom MCP Tool (Go) 🛠️🔌

## Theory

### From Consumer to Provider

Module 27 made your agent an MCP **client** — connecting to a server someone else built. This module flips the direction: you build the **server**. Once your own service speaks MCP, it's usable by any MCP-compliant client, not just your own ADK agents — a genuine, standalone, AI-addressable component.

### A Server Needs Two Things: a Tool Menu, and a Way to Run Them

`github.com/modelcontextprotocol/go-sdk/mcp` is the same package module-27's `mcptoolset` already depends on — it's a real, complete SDK for both sides of MCP, client and server. Building a server means three calls:

```go
server := mcp.NewServer(&mcp.Implementation{Name: "shopping_cart_mcp_server"}, nil)

mcp.AddTool(server, &mcp.Tool{
    Name:        "add_item_to_cart",
    Description: "Adds an item to the shopping cart.",
}, addItem)

if err := server.Run(context.Background(), &mcp.StdioTransport{}); err != nil {
    log.Fatal(err)
}
```

`mcp.NewServer` builds the server. `mcp.AddTool` registers one typed tool handler — this is the direct equivalent of Python's `@app.list_tools()` + `@app.call_tool()` pair, but as a single call instead of two decorated functions. `server.Run` with `&mcp.StdioTransport{}` serves over stdin/stdout — the server-side counterpart to `mcp.CommandTransport`, which module-27 already used from the client side.

### A Real Go Win: No Hand-Written Schema

Python's lab hand-writes each tool's `inputSchema` as a raw JSON-Schema dict. Go's `AddTool` needs none of that — confirmed in its own doc comment: the input (and output) JSON Schema is *inferred automatically* from your handler's own argument and return struct types, using `jsonschema` struct tags for property descriptions:

```go
type AddItemArgs struct {
    Item string `json:"item" jsonschema:"the item to add to the cart"`
}

func addItem(ctx context.Context, req *mcp.CallToolRequest, args AddItemArgs) (*mcp.CallToolResult, AddItemResult, error) {
    // ...
}
```

This is the exact same mechanism `functiontool.New` has used for regular ADK tools since module-9 — one typed struct, one inferred schema, no duplication between the schema and the code.

### Returning Structured Output, Automatically

A handler can return `nil` for `*mcp.CallToolResult` and a real, typed `Out` value instead — `CallToolResult`'s own doc comment confirms the framework auto-populates `Content` with that value's JSON text (and sets `StructuredContent` too). No manual `json.Marshal` + wrapping in a `TextContent`, unlike Python's lab.

### State Lives on the Server, Across Every Client Call

This module's server holds a shopping cart — a plain `[]string`, guarded by a mutex from the start (the same discipline this repo's shared-state trackers have followed since module-25; a real server can field concurrent calls from more than one session). Every `add_item_to_cart` call mutates it; every `view_cart` call reads it back. That's the whole point of building a *stateful* tool this way: the state genuinely lives in the server process, independent of whatever client (or how many clients) connect to it.

### A Real Go-Specific Gotcha: Stderr Isn't Free

`mcp.CommandTransport` wires a subprocess's stdin/stdout to the MCP protocol itself — confirmed by reading its own source, it never touches `Stderr`. Left unset, `exec.Cmd` discards it silently, meaning every `log.Printf` your server does vanishes into nothing for whoever launches it as a subprocess. `internal/agents/mcpcart.BuildRootAgent` sets `serverCmd.Stderr = os.Stderr` explicitly — without it, a learner would never see the server's own console logs Python's lab explicitly tells you to check.

### Going Further: Wrapping a Whole Agent as an MCP Tool (Confirmed Absent)

Python's own module closes with an experimental feature: `to_mcp_server(agent)`, wrapping an entire agent — its model loop and all its own tools — as a single MCP tool any host can call. Searched directly: no equivalent exists anywhere in the pinned `google.golang.org/adk/v2` source. The only "MCP server" concept the Go SDK's own examples reference is Google Cloud's separate **Agent Registry** product — a governed catalog client, not a way to wrap an agent as a tool. A confirmed scoping gap, matching Python's own "Experimental, preview" framing for the same feature — Theory-only here too, nothing to build against yet.

### Key Takeaways ✅
- `mcp.NewServer` + `mcp.AddTool` + `server.Run(ctx, &mcp.StdioTransport{})` are the real, complete way to build an MCP server in Go — the server side of the exact package module-27 already used as a client.
- `AddTool` infers JSON Schema automatically from your handler's own typed arguments and struct tags — no hand-written schema, unlike Python's lab.
- A handler can return `(nil, typedOutput, nil)` and let the framework build the response's `Content`/`StructuredContent` automatically.
- Server state (a mutex-guarded cart here) lives in the server process, shared across every client call — the actual meaning of "stateful tool."
- `mcp.CommandTransport` never forwards a launched subprocess's `Stderr` — forward it yourself (`serverCmd.Stderr = os.Stderr`) if you want to see the server's own logs.
- No Go equivalent of Python's experimental `to_mcp_server(agent)` exists in this SDK version — confirmed absent, Theory-only.

<hr/>

> **Coming from Python?** 🐍 Python's `@app.list_tools()`/`@app.call_tool()` decorator pair maps to Go's single `mcp.AddTool` call — same two responsibilities (advertise a schema, execute the logic), one function instead of two. Python hand-writes each tool's `inputSchema`; Go infers it from your own argument struct. Both sides mark `to_mcp_server`/wrapping-an-agent-as-a-tool as experimental/preview — this course's Go mirror doesn't build against it either, for the same reason.
