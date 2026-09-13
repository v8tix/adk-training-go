# Module 2: Setting Up Your Development Environment (Go)

## Theory

### The Importance of a Clean Environment

Before diving into building agents, it's crucial to set up a proper development environment. A well-structured environment ensures that your project's dependencies are isolated, preventing conflicts with other Go projects on your system. It makes your project self-contained and easily reproducible by others.

### Go 1.27+ and ADK 2.0 Requirements

* **Go Requirement:** Strictly **1.27 or higher** — this course's chosen floor (matches `go.mod`).
* **ADK Requirement:** `google.golang.org/adk/v2` (`v2.4.0`+), Google ADK 2.0's official Go SDK. Note its own package requirement is Go 1.25+ — lower than this course's floor, since 1.27 is what this repo actually targets.

### Go Modules: No Extra Tool Needed

Python needs `uv` (or `pip`/`venv`) as a separate package manager layered on top of the language. Go doesn't — module management, dependency locking, and reproducible builds are all built directly into the `go` toolchain:

* `go.mod` declares the module and its dependencies (equivalent to `uv`'s project file).
* `go.sum` locks exact dependency versions and their checksums (equivalent to `uv.lock`).
* `go get <module>` adds a dependency; `go mod tidy` keeps `go.mod`/`go.sum` in sync with what your code actually imports.

There's no separate virtual environment to activate either — a Go module *is* your isolated, reproducible unit.

### Development Workflow

This repo currently supports local Go development only:

1. Install Go 1.27+ from `https://go.dev/dl/`.
2. Verify installation: `go version`.
3. From the repo root, dependencies are already declared in `go.mod` — `go build ./...` fetches and builds everything.

> Codespaces and a `.devcontainer` configuration aren't set up for this repo yet. If you're used to the Python course's one-click browser setup, that parity is a separate, future piece of work — not part of this module.

### Authentication: Connecting to a Model

Unlike the Python course, this repo defaults to a **local** model — no cloud credentials needed to get started.

#### Option A: Local Ollama (Default, Recommended for This Course)

`cmd/verify-setup` and later modules call a local Ollama server by default: `http://localhost:11434` (OpenAI-compatible `/v1` endpoint), model `qwen38-standard`. No API key or `.env` entry is required for this path — Ollama doesn't check the key value. All of this is configurable: copy `.env.example` to `.env` and override `OLLAMA_BASE_URL` / `OLLAMA_MODEL` if your Ollama server lives elsewhere.

#### Option B: Google AI Studio API Key

1. Get an API key from Google AI Studio: `https://aistudio.google.com/app/apikey`.
2. Copy `.env.example` to `.env` and set `GOOGLE_API_KEY="YOUR_API_KEY"` and `MODEL_TYPE=gemini` — this switches `cmd/verify-setup` (and later modules) to this path.

#### Option C: Google Cloud Authentication (Enterprise)

For production, use Application Default Credentials via the Google Cloud CLI:

```bash
gcloud auth application-default login
gcloud config set project YOUR_PROJECT_ID
```

### Troubleshooting Common Errors

#### 1. 🛑 Build fails with "no required module provides package ..."

Run `go mod tidy` — it resolves and locks any dependency your code imports but `go.mod` doesn't declare yet.

#### 2. 🛑 `404: Model not found` (Gemini path only)

This usually means the model hasn't been deployed to your Google Cloud location yet. Change `GOOGLE_CLOUD_LOCATION` (or `LOCATION` in `.env`) to one of: `us-central1`, `us-east4`, `europe-west9`.

#### 3. 🛑 `PermissionDenied: 403` (Gemini path only)

Your account is missing the "Agent Platform User" role. Grant it via IAM in the Cloud Console (`roles/aiplatform.user`).

#### 4. 🛑 Connection refused to `localhost:11434` (Ollama path)

The local Ollama server isn't running, or `OLLAMA_BASE_URL` in your `.env` points somewhere unreachable. Confirm Ollama is up (`ollama list`), or switch to the Gemini path (`MODEL_TYPE=gemini`) with Option B or C above.

### Key Takeaways

- **Go 1.27+** (this course's floor) and `google.golang.org/adk/v2` (`v2.4.0`+) are strictly required.
- Go's own toolchain (`go.mod`/`go.sum`/`go mod tidy`) replaces what `uv` does for Python — no extra package manager needed.
- This course defaults to a **local Ollama model**; use a `.env` file and `MODEL_TYPE=gemini` only if you need to verify Google Cloud/AI Studio credentials specifically.
