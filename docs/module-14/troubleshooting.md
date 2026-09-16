# Troubleshooting: Module 14 (Go) 🔧

### `unable to fetch the results` 😤

**Cause:** `github.com/trietmn/go-wiki`'s underlying Wikipedia API calls fail with this real error if you don't set a distinctive User-Agent — Wikimedia rate-limits the package's generic default, since it's shared by every user of the package. Confirmed live.

**Fix:** call `gowiki.SetUserAgent("your-app-name/1.0 (contact-info)")` once, before any request:

```go
func init() {
    gowiki.SetUserAgent("your-app-name/1.0 (contact-info)")
}
```

`internal/agents/factfinder/tools.go` does this in its own package `init()`, confirmed live to be honored by later calls made from a different function — the classic "set once, anywhere before first use" pattern a package-level fix always needs. Still seeing the error after confirming `tools.go`'s `init()` actually runs (check it's actually set, not just documented)? Wait a few seconds and retry — Wikimedia occasionally rate-limits shared or cloud IP ranges regardless of a correctly-set User-Agent, a separate, transient issue.
