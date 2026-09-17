# Module 2: Setting Up Your Development Environment (Go) 🛠️

## Theory

### Why a Clean Environment Matters

Before you build any agents, let's get your dev environment squared away. A well-set-up environment keeps your project's dependencies isolated — no conflicts with other Go projects on your machine — and makes everything self-contained and easy for anyone else to reproduce. 🧹

### Go 1.27+ and ADK 2.0: What You Need

* **Go:** strictly **1.27 or higher** — that's this course's chosen floor (matches `go.mod`).
* **ADK:** `google.golang.org/adk/v2` (`v2.4.0`+), the official Go SDK for ADK 2.0. Fun fact: its own package requirement is Go 1.25+ — lower than this course's floor, since 1.27 is what this repo actually targets.

### Go Modules: Dependency Management, Built In 📦

Go's toolchain handles module management, dependency locking, and reproducible builds directly — no extra package manager to install:

* `go.mod` declares your module and its dependencies.
* `go.sum` locks exact dependency versions and their checksums.
* `go get <module>` adds a dependency; `go mod tidy` keeps `go.mod`/`go.sum` in sync with what your code actually imports.

No separate virtual environment to activate either — a Go module *is* your isolated, reproducible unit. Nice, right?

<hr/>

> **Coming from Python?** 🐍 `go.mod` plays the role of `uv`'s (or `pip`'s) project file, and `go.sum` is Go's equivalent of `uv.lock` — but there's no separate tool to install first; it's all baked into the `go` command you already have.

### Development Workflow

This repo currently supports local Go development only:

1. Install Go 1.27+ from `https://go.dev/dl/`.
2. Verify it: `go version`.
3. From the repo root, dependencies are already declared in `go.mod` — `go build ./...` fetches and builds everything.

> Codespaces and a `.devcontainer` setup aren't there yet for this repo — a one-click browser setup is future work, not part of this module.

### Authentication: Connecting to a Model 🔑

This course defaults to a **local** model — no cloud credentials needed to get started. 🎉

#### Option A: Local Ollama (Default, Recommended for This Course)

`cmd/verify-setup` and every later module call a local Ollama server by default: `http://localhost:11434` (OpenAI-compatible `/v1` endpoint), model `qwen3.8:27b` — a GGUF quantization, chosen because it's the one confirmed (module-4) to support JSON-schema-constrained structured output as well as plain text. No API key or `.env` entry needed here — Ollama doesn't check the key value. It's all configurable too: copy `.env.example` to `.env` and override `OLLAMA_BASE_URL` / `OLLAMA_MODEL` if your Ollama server lives elsewhere.

#### Option B: Google AI Studio API Key

1. Grab an API key from Google AI Studio: `https://aistudio.google.com/app/apikey`.
2. Copy `.env.example` to `.env` and set `GOOGLE_AI_STUDIO_API_KEY="YOUR_API_KEY"` and `MODEL_TYPE=gemini` — this switches `cmd/verify-setup` (and everything after) to this path.

#### Option C: Google Cloud Authentication (Enterprise)

For production, use Application Default Credentials via the Google Cloud CLI:

```bash
gcloud auth application-default login
gcloud config set project YOUR_PROJECT_ID
```

### Troubleshooting Common Errors 🔧

#### 1. 🛑 Build fails with "no required module provides package ..."

Run `go mod tidy` — it resolves and locks any dependency your code imports but `go.mod` doesn't declare yet.

#### 2. 🛑 `404: Model not found` (Gemini path only)

Usually means the model hasn't been deployed to your Google Cloud location yet. Change `GOOGLE_CLOUD_LOCATION` (or `LOCATION` in `.env`) to one of: `us-central1`, `us-east4`, `europe-west9`.

#### 3. 🛑 `PermissionDenied: 403` (Gemini path only)

Your account is missing the "Agent Platform User" role. Grant it via IAM in the Cloud Console (`roles/aiplatform.user`).

#### 4. 🛑 Connection refused to `localhost:11434` (Ollama path)

The local Ollama server isn't running, or `OLLAMA_BASE_URL` in your `.env` points somewhere unreachable. Confirm Ollama is up (`ollama list`), or switch to the Gemini path (`MODEL_TYPE=gemini`) with Option B or C above.

### Key Takeaways ✅

- **Go 1.27+** (this course's floor) and `google.golang.org/adk/v2` (`v2.4.0`+) are strictly required.
- Go's own toolchain (`go.mod`/`go.sum`/`go mod tidy`) handles dependency management directly — no extra package manager to install.
- This course defaults to a **local Ollama model**; only reach for a `.env` file and `MODEL_TYPE=gemini` when you need to verify Google Cloud/AI Studio credentials specifically.
