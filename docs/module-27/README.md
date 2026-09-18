# Module 27: Introduction to MCP & Stateful Tools (Go) 🔌🗂️

## Theory

### A Function Tool Forgets Everything Between Calls

Every custom tool since module-9 has been a Go function: it runs, returns a value, and forgets it ever ran. That's fine for `add(a, b)`, but it falls short the moment a tool needs to hold onto real state across a conversation — a live database connection, an open file handle, a multi-step booking flow.

The **Model Context Protocol (MCP)** is an open standard for exactly this: a client-server protocol letting an agent talk to *external, stateful* tools instead of only in-process functions. Instead of writing a bespoke integration for every service, your agent connects to a pre-built MCP server and gets its tools for free — discovered at runtime, not declared in your own code.

### `mcptoolset`: ADK's Real MCP Client

`google.golang.org/adk/v2/tool/mcptoolset` is a real, complete Go MCP client — confirmed directly in its own source, not assumed from a getting-started page. `mcptoolset.New(mcptoolset.Config{...})` returns a `tool.Toolset` you drop straight into `llmagent.Config.Toolsets`:

```go
toolset, err := mcptoolset.New(mcptoolset.Config{
    Transport: &mcp.CommandTransport{
        Command: exec.Command("npx", "-y", "@modelcontextprotocol/server-filesystem", sandboxDir),
    },
})

llmagent.New(llmagent.Config{
    Name:     "filesystem_agent",
    Model:    llmModel,
    Toolsets: []tool.Toolset{tool.FilterToolset(toolset, tool.AllowedToolsPredicate([]string{"list_directory", "read_file"}))},
})
```

Note `Toolsets`, not `Tools` — a separate field from the single-tool slice every module since 9 has used. A `tool.Toolset` doesn't declare its own tools at compile time; it *discovers* them from a live server when `.Tools()` is called.

### Three Real Transports, One Toolset

`mcptoolset` wraps `github.com/modelcontextprotocol/go-sdk/mcp`, the official, independently-maintained Go MCP SDK. That package ships three real transport types, confirmed by reading `cmd.go`, `streamable.go`, and `sse.go` directly:

| Go type | What it connects to |
|---|---|
| `mcp.CommandTransport` | A local server, launched as a subprocess, talking over stdin/stdout |
| `mcp.StreamableClientTransport` | A remote server, over bidirectional HTTP streaming |
| `mcp.SSEClientTransport` | A remote server, over Server-Sent Events |

This lab exercises the first two: a local filesystem server over stdio, then a remote GitHub server over StreamableHTTP.

### The Connection Is Lazy

`mcptoolset.New`'s own doc comment states the session is created lazily, on the first request to the LLM — confirmed by reading `set.go`: `New` only builds a transport struct; nothing dials or launches a subprocess until `.Tools()` is actually called. This has a genuinely useful consequence: building an agent around an MCP toolset is a pure, side-effect-free construction step, testable without a live server at all — see `agent_test.go`'s `TestBuildRootAgent_Constructs`, which succeeds even for a sandbox directory that doesn't exist yet.

### Security: Sandbox What You Launch

`mcp.CommandTransport` runs a real subprocess with your program's own privileges — `npx -y @modelcontextprotocol/server-filesystem <path>` can read and write anywhere `<path>` points, and nothing else stops it. The sandboxing here is entirely the directory argument you pass in — not an OS-level jail the SDK provides for you. Always point a stdio MCP server at the narrowest directory the lab actually needs, never at a home directory or the repo root.

### Filtering the Server's Own Tool Surface

An MCP server can expose more tools than your agent should use. `tool.FilterToolset(toolset, predicate)` restricts what the LLM ever sees — this lab uses `tool.AllowedToolsPredicate([]string{"list_directory", "read_file"})` to allow exactly two of the filesystem server's tools, nothing else. (Two older names do the same thing but are both marked deprecated: `mcptoolset.Config.ToolFilter` in `FilterToolset`'s favor, and `tool.StringPredicate` in `AllowedToolsPredicate`'s favor — use the current pairing.)

### Confirmed Live: No MCP Session Timeout on a Cold `npx` Cache

Cleared this machine's `npx` execution cache for the filesystem server package and re-ran the console demo from scratch: no session timeout, no retry needed — it worked on the first attempt. Noted here because it's a genuine, tested finding, not carried over from anywhere else.

### Key Takeaways ✅
- Standard function tools are stateless; MCP is the real, standard way to give an agent access to *stateful* external tools.
- `mcptoolset.New` builds a `tool.Toolset` from any of three real transports (`CommandTransport`, `StreamableClientTransport`, `SSEClientTransport`) — go into `llmagent.Config.Toolsets`, a separate field from `Tools`.
- The MCP session connects lazily — building an agent around a toolset never dials anything itself, making construction fully unit-testable.
- `tool.FilterToolset` + `tool.AllowedToolsPredicate` restrict which of a server's tools the LLM actually sees.
- A stdio-launched MCP server runs with your program's own privileges — sandboxing is your responsibility, expressed as the directory path you pass it, not something the SDK enforces for you.

<hr/>

> **Coming from Python?** 🐍 Python's own module covers the same three connection types (`StdioConnectionParams`, `SseConnectionParams`, `StreamableHTTPConnectionParams`) and the same `McpToolset` mental model — Go's naming differs (`mcp.CommandTransport` vs. `StdioConnectionParams`, `tool.FilterToolset` vs. `tool_filter=[...]`) but the architecture and the security warning about sandboxing are identical. One real difference: Python's own lab warns that the very first request may hit an MCP session timeout while `npx` downloads the server package for the first time, advising a simple retry. Tested that exact scenario live in Go (cleared the npx execution cache, re-ran from scratch) and did not reproduce a timeout on this machine — worth knowing about if you hit it, but not something this Go mirror can promise never happens on every machine or network.
