# Module 23: Handling Files with Artifacts (Go) 📄🖼️

## Theory

### Why Artifacts, Not Just State

Module 22's session state is great for small key-value facts — a preference, a running score. But a real report, a generated chart, or a processed document is a *file*, and it deserves its own history: every time you save it, you want the old version kept, not silently overwritten. That's exactly what Go's **Artifact** system gives you.

### One Store, Real Versions

An artifact is a named file, automatically versioned on every save. `artifact.Service` (confirmed via `go doc`) is the interface: `Save`, `Load`, `Delete`, `List`, `Versions`, `GetArtifactVersion`. `artifact.InMemoryService()` is the local-dev implementation — same role as `session.InMemoryService()` from Module 22, just for files. A real, production-ready `artifact/gcsartifact.NewService(ctx, bucketName, opts...)` backs artifacts with a Google Cloud Storage bucket when you need them to survive past a single process.

**Important, confirmed by tracing both backends' own `Save` implementation:** a file's *first* save is version **1**, not version 0. Both `InMemoryService` and `gcsartifact`'s `NewService` compute `nextVersion := 1` when nothing exists yet — this repo's own tests prove it live (`TestArtifactService_VersionsStartAtOne`), not just cite it.

### One Context, Same Pattern as Module 22

Just like `agent.Context.State()` unified Python's `ToolContext`/`CallbackContext` for session data, `agent.Context.Artifacts()` is the *one* accessor every custom function tool uses for files:

```go
func extractText(ctx agent.Context, args ExtractTextArgs) (ExtractTextResult, error) {
    resp, err := ctx.Artifacts().Save(ctx, name, genai.NewPartFromText(content))
    if err != nil {
        return ExtractTextResult{}, err
    }
    return ExtractTextResult{Version: int(resp.Version)}, nil
}
```

`Save(ctx, name, *genai.Part)` returns the new version number. `Load(ctx, name)` fetches the latest version (or a specific one via `LoadVersion`). `List(ctx)` returns every filename in the current scope. Every call is auto-scoped to the current app/user/session — you never pass those explicitly, the same way Module 22's `ctx.State()` never needed them either.

### Binary Content Needs a Real MIME Type

Text content uses `genai.NewPartFromText(text)`. Binary content — an image, a PDF, audio — uses `genai.NewPartFromBytes(data, mimeType)`, and the MIME type isn't optional decoration: it's how any later reader (including this module's own `create_report` tool) tells a text artifact from an image one, via `part.InlineData.MIMEType`.

```go
resp, err := ctx.Artifacts().Save(ctx, "chart.png", genai.NewPartFromBytes(pngBytes, "image/png"))
```

### Two Scopes: Session and User

By default, an artifact lives only in the session that created it. Prefix the filename with `user:` and it becomes visible to that user across *every* session they start — confirmed identically in both backends' source (`strings.HasPrefix(filename, "user:")` in `artifact/inmemory.go` and `artifact/gcsartifact/service.go`), the same convention Module 22 already established for state keys, just applied to filenames instead.

- `"report.txt"` → this session only.
- `"user:preferences.json"` → this user, any session.

This repo's own tests prove both directions live: `TestArtifactService_UserPrefixScopesAcrossSessions` confirms a `user:`-prefixed file crosses sessions, and `TestArtifactService_PlainFilenameIsSessionScoped` confirms a plain one genuinely doesn't (a real negative case, not just the positive one).

### Missing Artifacts Use a Standard Library Sentinel

Load a filename that was never saved and `Load` returns `fmt.Errorf("artifact not found: %w", fs.ErrNotExist)` — wrapping the *standard library's own* `io/fs.ErrNotExist`, not a bespoke sentinel. Check it the same way you'd check any wrapped error:

```go
if _, err := ctx.Artifacts().Load(ctx, name); errors.Is(err, fs.ErrNotExist) {
    // handle the not-yet-created case
}
```

### No Async/Await to Teach

Every artifact call here is an ordinary, synchronous Go function taking a `context.Context` — there's no separate "this must be awaited" distinction to learn. It's simply not a dimension Go's tool-calling model has.

### A Real, Differently-Shaped Mechanism for Credentials

There's no `SaveCredential`/`LoadCredential` method on `agent.Context` — confirmed by reading its full method set. But `google.golang.org/adk/v2/auth` is real and substantial: a `CredentialProvider` (`StaticToken`, `APIKey`, `ADC`, `ServiceAccount`) resolves a `Credential` that writes itself onto an outbound HTTP request, applied per-call via a `Transport`. It solves the same real problem — a tool needs a secret to call something — just shaped as an HTTP-transport-level resolver rather than a session-scoped secret store. This module's own lab doesn't build it hands-on, matching the source curriculum's own scope for this topic.

> **Going Further:** if you want to try the real `auth` package, wire a `CredentialProvider` (`auth.StaticToken` is the simplest starting point) into an HTTP client one of your own tools uses, and confirm the outbound request actually carries the resolved credential.

### Key Takeaways ✅
- An artifact is a named, automatically versioned file — `artifact.Service`'s `Save`/`Load`/`Delete`/`List`/`Versions`.
- The first save of any file is **version 1**, confirmed live against both the in-memory and GCS backends' own source.
- `agent.Context.Artifacts()` is the one accessor every tool uses — no separate context type, matching Module 22's own `State()` pattern.
- Binary content needs a real MIME type (`genai.NewPartFromBytes`); a missing artifact reports as the standard library's own `fs.ErrNotExist`, wrapped.
- The `user:` filename prefix scopes an artifact across every session for that user — proven live in both directions, positive and negative.
- Go's real, differently-shaped credential mechanism lives in the `auth` package — not a like-for-like match for a session-scoped secret store, but not an absence either.

<hr/>

> **Coming from Python?** 🐍 Python's own module covers the same Artifact model — `save_artifact`/`load_artifact`/`list_artifacts`, `types.Part.from_bytes()`/`from_text()`, the `user:` scoping convention, `InMemoryArtifactService`/`GcsArtifactService`. The one thing worth flagging directly: Python's docs describe versions as 0-indexed ("the first save creates version 0"); Go's own backends — both of them, traced directly — start at 1 instead.
