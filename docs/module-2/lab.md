# Lab 2: Environment Setup Challenge (Go)

## Prerequisites

Before you begin, ensure you have the following ready:

* **Code Editor (IDE):** [VS Code](https://code.visualstudio.com/) or your preferred Go-aware editor.
* **Go 1.27+** installed locally. This repo doesn't have Codespaces or DevContainer support yet — local setup is the only supported path for now.

## Goal

Your task is to verify this repo's Go environment is ready for ADK 2.0 development. If you get stuck, work backward from the Troubleshooting section below.

## Lab Tasks

### Step 0: Ensure Go 1.27+ (Crucial)

Check your installed version:

```bash
go version
```

If it reports anything below `go1.27`, install a newer Go from `https://go.dev/dl/` before continuing.

### Step 1: Confirm the ADK Dependency

From the repo root, confirm `google.golang.org/adk/v2` is a declared dependency:

```bash
grep adk go.mod
```

If it's missing (e.g. you're working from an earlier point in the repo's history), add it yourself:

```bash
go get google.golang.org/adk/v2
go mod tidy
```

### Step 2: Configure Authentication (Optional)

By default, `cmd/verify-setup` calls a local Ollama server — no `.env` file needed. If you want to check Google Cloud/AI Studio credentials instead:

1. Copy `.env.example` to `.env` (already covered by `.gitignore` — never commit the real one).
2. Set your key and switch paths:
   ```
   GOOGLE_API_KEY="YOUR_API_KEY"
   MODEL_TYPE=gemini
   ```

### Step 3: Read `cmd/verify-setup`

Open `cmd/verify-setup/main.go`, `checks.go`, `config.go`, and `model_factory.go`. Find:

1. The function that checks the resolved `google.golang.org/adk/v2` version — how does it read that without importing an `internal` package it isn't allowed to?
2. The function that checks the Go version.
3. Where the local Ollama model gets built vs. where Gemini does, and what decides which one runs.
4. Where `OLLAMA_BASE_URL`, `OLLAMA_MODEL`, and the other `.env`-configurable values get their defaults.

### Step 4: Run the Verification

```bash
go run ./cmd/verify-setup
```

Or, to check the Gemini path instead:

```bash
MODEL_TYPE=gemini go run ./cmd/verify-setup
```

### 💡 Troubleshooting: Connection Refused

If you see a connection error to `localhost:11434`, the local Ollama server isn't running — confirm it's up (`ollama list`), or fall back to `MODEL_TYPE=gemini` with a configured `.env`.

## Self-Reflection Questions

* Go's toolchain bakes in what Python needs `uv` for (dependency locking, reproducible builds). What's the tradeoff of that being a language feature versus a separate tool?
* Why does `cmd/verify-setup` read the ADK SDK's version via `runtime/debug.ReadBuildInfo()` instead of importing the SDK's own version package directly?
* What are the security implications of defaulting to a local model with no API key, versus requiring cloud credentials up front?

<hr/>

### Looking for the solution?

Hint: read `cmd/verify-setup/model_factory.go`'s `buildModel` function and the `modelFactories` map — that's what dispatches between Ollama and Gemini, based on the `MODEL_TYPE` environment variable.
