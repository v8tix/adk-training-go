# Lab 27: Connecting an Agent to a Stateful File System Tool (Go) 🔌🗂️

## Goal

Build an agent whose only tools come from a real, external Model Context Protocol server — a filesystem server launched as a local subprocess — then prove it can list and read real files it never defined tools for itself. A bonus section swaps the local subprocess for a remote, network-hosted MCP server.

### Prerequisites

- **Node.js and `npx`**: the filesystem MCP server (`@modelcontextprotocol/server-filesystem`) is a third-party Node.js package. Install Node.js (which includes `npx`) from [nodejs.org](https://nodejs.org/) if you don't already have it. This is genuinely required — MCP is protocol-agnostic to the client's language, but the pre-built server this lab connects to happens to be a Node package.

## Lab Tasks

### 1. Read `internal/agents/mcpfilesystem/agent.go`

`BuildRootAgent(llmModel, sandboxDir)` wires an `mcptoolset.New` toolset (via `mcp.CommandTransport`, launching `npx -y @modelcontextprotocol/server-filesystem <sandboxDir>` as a subprocess) into `llmagent.Config.Toolsets`, filtered down to `list_directory`/`read_file` with `tool.FilterToolset`. `sandboxDir` is an explicit parameter, not a hidden package-level path — this is both more idiomatic Go and what makes the live test below fully self-contained.

### 2. Read `internal/agents/mcpfilesystem/agent_test.go`

`TestBuildRootAgent_Constructs` proves construction never dials anything — the MCP session is lazy, confirmed in the README's Theory section. `TestFilesystemMCP_ListsAndReadsFile_{Ollama,Gemini}` builds its own isolated `t.TempDir()` sandbox with a known fixture file, drives a real `runner.New` through two turns, and asserts the real file name and real file content both appear in the agent's own answers. `npxReachable()` makes both live variants skip cleanly (not fail) when Node.js isn't installed — the same discipline `llm.OllamaReachable` already established for a different kind of external dependency.

### 3. Run it — console mode, entirely locally 🖥️

```bash
go run ./cmd/mcp-filesystem console
```

No `.env`, no API key needed. Real, confirmed output from this exact command (thinking-model reasoning trimmed for readability):

```
📂 mcp-filesystem using qwen3.8:27b
Sandboxed to /path/to/adk-training-go/cmd/mcp-filesystem/test_files

User -> What files are in my directory?
Agent -> Your directory contains one file:

- **hello.txt** (file)

Would you like me to read the contents of `hello.txt`?

User -> Great, can you read the content of hello.txt for me?
Agent -> Here's the content of `hello.txt`:

```
Hello from the MCP world!
```

Is there anything else you'd like me to do?
```

Both `list_directory` and `read_file` were **discovered at runtime** from the real MCP server — neither is defined anywhere in this repo's own code.

### Checkpoint

- [ ] Real captured output shows the agent listing `hello.txt` by name and reading its real content back
- [ ] `go test ./internal/agents/mcpfilesystem/... -race` passes
- [ ] `go test ./internal/agents/mcpgithub/... -race` passes (header-logic tests always run; the live network test skips cleanly without `GITHUB_TOKEN`)
- [ ] `go test ./cmd/mcp-filesystem/... -race` passes (sandbox-directory resolution and creation logic)

### Bonus: Connecting to a Remote MCP Server 🐙

So far the agent talked to a *local* MCP server launched as a subprocess. Most real-world integrations instead connect to a server already running somewhere else — `internal/agents/mcpgithub` does exactly that, over `mcp.StreamableClientTransport`, against GitHub's own hosted MCP server.

#### 4. Read `internal/agents/mcpgithub/agent.go`

`githubAuthTransport` is a ~15-line custom `http.RoundTripper` setting two headers (`Authorization: Bearer <token>`, `X-MCP-Readonly: true`) on every outgoing request — enough for a fixed Personal Access Token, without pulling in `golang.org/x/oauth2`'s token-refresh machinery for a problem (expiring tokens) a PAT doesn't have.

#### 5. Get a free GitHub Personal Access Token

Create one at [github.com/settings/personal-access-tokens/new](https://github.com/settings/personal-access-tokens/new) with read-only repository access, then set it in your `.env`:

```
GITHUB_TOKEN=your_token_here
```

#### 6. Run it

```bash
go run ./cmd/mcp-github console
```

Try asking: "What are the open issues on google/adk-go?" — no subprocess, no `npx`, no local sandboxing: the tool call goes straight over HTTPS to GitHub's servers. (This session didn't have a real token available to capture live output with — `internal/agents/mcpgithub/agent_test.go`'s `TestGithubAuthTransport_SetsBothHeaders` proves the header-setting logic directly, and `TestGitHubMCP_ListsOpenIssues_Ollama` skips cleanly without `GITHUB_TOKEN` set, the same discipline this repo's Gemini-optional tests already follow.)

A few things change when the server is remote instead of local: there's no subprocess lifecycle to manage, network failures (timeouts, rate limits, an expired token) become real possibilities a local stdio server never has, and the security surface shifts from "the subprocess has filesystem access" to "don't hardcode the credential" — which is why the token comes from `.env`, never a literal string in code.

### 7. Read `internal/agents/mcpgithub/agent_test.go`

`TestGithubAuthTransport_SetsBothHeaders` and `TestGithubAuthTransport_DefaultsToDefaultTransport` prove the round-tripper's header logic and its nil-base fallback, both without a network call.

## Self-Reflection Questions 🤔
- `mcptoolset.New` never dials anything — what would change about this lab's tests if MCP sessions connected eagerly, at construction time, instead?
- `tool.FilterToolset` restricts which tools the LLM sees, but the MCP server itself still has every tool available to whoever else connects to it. What's the actual security boundary here, and what isn't it?
- If you wanted the filesystem agent to also *write* files, not just list and read them, what would need to change — anything in this repo's own code, or something else entirely?
- Why does `githubAuthTransport` clone the request before mutating its headers, instead of modifying `req` directly?

<hr/>

### Looking for the solution? 🔍

Hint: read `internal/agents/mcpfilesystem/agent.go` and `internal/agents/mcpgithub/agent.go` for the real mechanism — two small agent packages, each wrapping `mcptoolset.New` with a different transport.
