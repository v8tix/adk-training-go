# 🎓 ADK Training for Go 🚀

Welcome! This repo is a **Go-language mirror** of [**Google ADK Training: From Zero to Hero**](https://mauripsale.github.io/doc-adk-training/docs/) ([source](https://github.com/mauripsale/doc-adk-training)), a hands-on training course for the **Google Agent Development Kit (ADK)**. Every module here is built and verified against the real, official Go SDK (`google.golang.org/adk/v2`) — no Python required.

## 🧭 What This Is (and Isn't)

The original course teaches ADK 2.0 through Python, across 40 modules. This repo mirrors that same curriculum **module by module in Go**, adapting every lesson, lab, and code sample to real, idiomatic Go — including calling out the places where Go's SDK behaves genuinely differently from Python's (and there are more of those than you'd expect 👀).

**Current coverage:** modules 1 through 30 — everything from your first agent through distributed, multi-agent systems over real A2A network calls, plus state and memory, artifacts, evaluation, observability, RAI safety plugins, callbacks/guardrails, building/consuming MCP stateful tools, custom UI integration, and bidirectional audio streaming over a custom voice client. See the [full module list](#-course-outline) below.

Every technical claim in these docs is grounded in one of two things: the actual Go SDK source, read directly, or a live probe/test that proved the behavior — never just a Python-to-Go guess. Where Go's SDK does something differently (sometimes better, sometimes just different) than Python's, the docs say so explicitly.

## ✍️ Original Course & Attribution

This project is an **Adapted Material** derivative of [doc-adk-training](https://github.com/mauripsale/doc-adk-training) — read the original course itself at **[mauripsale.github.io/doc-adk-training/docs](https://mauripsale.github.io/doc-adk-training/docs/)** — created and maintained by [**Maurizio Ipsale**](https://www.linkedin.com/in/maurizioipsale/), a Google Cloud Authorized Trainer and Google Developer Expert (GDE) in AI and Cloud. All credit for the course design, module structure, and pedagogical approach goes to the original author.

The training curriculum content is licensed under [**Creative Commons Attribution 4.0 International (CC BY 4.0)**](https://creativecommons.org/licenses/by/4.0/) — see [`LICENSE-DOCS`](LICENSE-DOCS). This repo exercises that license by translating and adapting the curriculum into Go: **the docs, labs, and course structure in `docs/` are Adapted Material derived from the original Python course**, modified to teach the Go SDK instead. The original material is provided as-is, with no warranty, under the same license.

## 📦 What's Actually in Here

```
docs/          → the training curriculum itself (Theory + Lab per module) — Adapted Material, CC BY 4.0
internal/      → Go agent implementations (one package per module/lab) — original code, Apache 2.0
cmd/           → runnable launcher programs, one per module's lab exercise — original code, Apache 2.0
```

Every module's docs live at `docs/module-N/`: `README.md` (Theory), `lab.md` (the hands-on exercise), and `troubleshooting.md` where a module has real, confirmed gotchas worth documenting separately. Every doc file also has a native Spanish translation (`.es.md`) alongside it.

**Documentation conventions:**
- Every live interaction with an agent — any question, command, or console turn typed to it while building or capturing a "real, confirmed output" transcript — is conducted in English, regardless of which doc (English or `.es.md`) the transcript ends up in. A captured console-output block stays in English verbatim in both language twins; only the surrounding prose is translated for `.es.md`. The one exception is a module whose lesson is itself a foreign-language capability by design (e.g. module-15's "Spanish greeter" specialist), where the foreign-language example is the actual lesson content, not a stray leak.
- This README's own Course Outline table (below) and its "Current coverage" line are updated as part of shipping every module — never left to fall behind.

## 🚀 Getting Started

Requires **Go 1.27+**. See [`docs/module-02/README.md`](docs/module-02/README.md) for the full environment setup walkthrough (installing Go, Ollama for a local model, or configuring Gemini).

```bash
go get google.golang.org/adk/v2
cp .env.example .env   # defaults to a local Ollama model — no cloud credentials needed
```

Each module's lab is a runnable `cmd/` program. For example:

```bash
go run ./cmd/echo-agent console
```

Full details for each lab — including exact commands, expected output, and troubleshooting — live in that module's own `docs/module-N/lab.md`.

### 🧰 Tooling

Every lab from module 1 onward only needs Go and (optionally) Ollama. A few later modules exercise something genuinely external — each one degrades gracefully (a skipped, not failed, test) if the tool below isn't installed, but you'll need it to actually run that module's own live example:

| Tool | Needed for | Install |
|---|---|---|
| **Go 1.27+** | Every module | [go.dev/dl](https://go.dev/dl/) |
| **Ollama** | The default local model every module uses unless you switch to a cloud backend | [ollama.com](https://ollama.com/) |
| **Docker** | Module 13.5's Redis integration tests (Testcontainers spins up and tears down an ephemeral Redis container automatically — no manual Redis setup needed just to run `go test`) | [docker.com/get-started](https://www.docker.com/get-started/) |
| **A reachable Redis** (default `localhost:6379`, override with `REDIS_ADDR`) | Module 13.5's own live demo (`cmd/persistent-agent`) — separate from Docker above, this one needs an actual persistent instance, not Testcontainers' ephemeral one | `./scripts/up.sh` (uses this repo's own `docker-compose.yml`; `./scripts/stop.sh`/`clean.sh` to shut it down), or a native install from [redis.io](https://redis.io/docs/latest/operate/oss_and_stack/install/) |
| **Node.js (includes `npx`)** | Module 27's filesystem MCP server (`@modelcontextprotocol/server-filesystem`), a third-party Node package | [nodejs.org](https://nodejs.org/) |
| **Google Cloud CLI (`gcloud`)** | Module 12's `google_maps_grounding` bonus, via Vertex AI Application Default Credentials | `brew install --cask google-cloud-sdk` (macOS/Homebrew), or `curl https://sdk.cloud.google.com \| bash` — then `gcloud auth application-default login` |
| **Google Chrome or Chromium** | Module 29's headless-browser UI test (`github.com/chromedp/chromedp` drives an existing install — it doesn't download its own browser) | [google.com/chrome](https://www.google.com/chrome/) or your package manager's `chromium` |

This table grows as later modules add their own real, external tool. If a module can't run without something beyond Go/Ollama, it's listed here. Every automated test tied to one of these tools skips cleanly (not fails) when the tool isn't available — you only need to actually install one if you want to run that specific module's own live example yourself.

## 📚 Course Outline

| # | Module | Docs |
|---|---|---|
| 1 | Introduction to AI Agents & Google ADK 🤖 | [README](docs/module-01/README.md) · [Lab](docs/module-01/lab.md) |
| 2 | Setting Up Your Development Environment 🛠️ | [README](docs/module-02/README.md) · [Lab](docs/module-02/lab.md) |
| 3 | Your First Agent: The "Echo" Agent 🦜 | [README](docs/module-03/README.md) · [Lab](docs/module-03/lab.md) |
| 4 | Core Agent Concepts: Agent Deep Dive 🔬 | [README](docs/module-04/README.md) · [Lab](docs/module-04/lab.md) |
| 4.5 | Professional Model Configuration, Resiliency & Portability 🛡️ | [README](docs/module-04_5/README.md) · [Lab](docs/module-04_5/lab.md) |
| 5 | Running and Interacting with Agents 🖥️ | [README](docs/module-05/README.md) · [Lab](docs/module-05/lab.md) |
| 6 | Programmatic Execution: Apps and Runners ⚙️ | [README](docs/module-06/README.md) · [Lab](docs/module-06/lab.md) |
| 7 | Multimodal and Image Processing 🖼️ | [README](docs/module-07/README.md) · [Lab](docs/module-07/lab.md) |
| 8 | Introduction to Tools 🔧 | [README](docs/module-08/README.md) · [Lab](docs/module-08/lab.md) |
| 9 | Creating Custom Function Tools 🛠️ | [README](docs/module-09/README.md) · [Lab](docs/module-09/lab.md) · [Troubleshooting](docs/module-09/troubleshooting.md) |
| 10 | Giving Agents Memory with Stateful Tools 🧠 | [README](docs/module-10/README.md) · [Lab](docs/module-10/lab.md) |
| 11 | Enterprise Integration with a Declarative Tool 🔌 | [README](docs/module-11/README.md) · [Lab](docs/module-11/lab.md) |
| 12 | Built-in Tools and Grounding 🌐 | [README](docs/module-12/README.md) · [Lab](docs/module-12/lab.md) · [Troubleshooting](docs/module-12/troubleshooting.md) |
| 13 | Advanced Interactions — Actions & HITL 🛑 | [README](docs/module-13/README.md) · [Lab](docs/module-13/lab.md) · [Troubleshooting](docs/module-13/troubleshooting.md) |
| 13.5 | Extending ADK — Custom Persistence with Redis 🗄️ | [README](docs/module-13_5/README.md) · [Lab](docs/module-13_5/lab.md) · [Troubleshooting](docs/module-13_5/troubleshooting.md) |
| 14 | Integrating Third-Party Tools 📦 | [README](docs/module-14/README.md) · [Lab](docs/module-14/lab.md) · [Troubleshooting](docs/module-14/troubleshooting.md) |
| 15 | Introduction to Multi-Agent Systems 🤝 | [README](docs/module-15/README.md) · [Lab](docs/module-15/lab.md) |
| 16 | Static Orchestration — Linear and Parallel Edges 🧩 | [README](docs/module-16/README.md) · [Lab](docs/module-16/lab.md) · [Troubleshooting](docs/module-16/troubleshooting.md) |
| 17 | Structured Routing — Edges and Dictionaries 🔀 | [README](docs/module-17/README.md) · [Lab](docs/module-17/lab.md) · [Troubleshooting](docs/module-17/troubleshooting.md) |
| 18 | Dynamic Orchestration — Programmable Graphs 🎛️ | [README](docs/module-18/README.md) · [Lab](docs/module-18/lab.md) · [Troubleshooting](docs/module-18/troubleshooting.md) |
| 19 | Collaborative Teams — Modes and Hand-offs 🤝 | [README](docs/module-19/README.md) · [Lab](docs/module-19/lab.md) · [Troubleshooting](docs/module-19/troubleshooting.md) |
| 20 | Cyclic Workflows — Iteration and Self-Correction 🔁 | [README](docs/module-20/README.md) · [Lab](docs/module-20/lab.md) · [Troubleshooting](docs/module-20/troubleshooting.md) |
| 21 | Distributed Graphs — A2A and External Nodes 🌐 | [README](docs/module-21/README.md) · [Lab](docs/module-21/lab.md) · [Troubleshooting](docs/module-21/troubleshooting.md) |
| 21.5 | MAS Knowledge Milestone — Architecture Choice 🏆 | [README](docs/module-21_5/README.md) · [Lab](docs/module-21_5/lab.md) |
| 22 | State and Memory — Persistent Agent Context 🧠📚 | [README](docs/module-22/README.md) · [Lab](docs/module-22/lab.md) |
| 23 | Handling Files with Artifacts 📄🖼️ | [README](docs/module-23/README.md) · [Lab](docs/module-23/lab.md) |
| 24 | Evaluating Agent Performance 🧪📊 | [README](docs/module-24/README.md) · [Lab](docs/module-24/lab.md) |
| 25 | Advanced Observability with Plugins 🔭📊 | [README](docs/module-25/README.md) · [Lab](docs/module-25/lab.md) |
| 25.5 | Responsible AI (RAI) & Safety Plugins 🛡️🚫 | [README](docs/module-25_5/README.md) · [Lab](docs/module-25_5/lab.md) |
| 26 | Callbacks and Guardrails — Agent Safety and Monitoring 🚦🧯 | [README](docs/module-26/README.md) · [Lab](docs/module-26/lab.md) |
| 27 | Introduction to MCP & Stateful Tools 🔌🗂️ | [README](docs/module-27/README.md) · [Lab](docs/module-27/lab.md) |
| 28 | Building a Custom MCP Tool 🛠️🔌 | [README](docs/module-28/README.md) · [Lab](docs/module-28/lab.md) |
| 29 | Introduction to UI Integration 💬🌐 | [README](docs/module-29/README.md) · [Lab](docs/module-29/lab.md) |
| 30 | Building a Custom Streaming Client 🎙️🔊 | [README](docs/module-30/README.md) · [Lab](docs/module-30/lab.md) |

## 📜 License

This project uses a split license, matching the same split the original course uses for its own code samples vs. curriculum:

- **Go source code** (`internal/`, `cmd/`) — [Apache License 2.0](LICENSE).
- **Training curriculum** (`docs/`) — [Creative Commons Attribution 4.0 International](LICENSE-DOCS), as **Adapted Material** derived from [doc-adk-training](https://github.com/mauripsale/doc-adk-training) by Maurizio Ipsale. The curriculum is shared and modified under the same license; this repo does not claim endorsement by, or affiliation with, the original author or Google.

The original material — and this adaptation of it — is provided **as-is**, with no warranties of any kind. See the full disclaimer of warranties in each LICENSE file.
