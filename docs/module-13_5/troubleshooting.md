# Troubleshooting: Module 13.5 (Go)

### `ask` doesn't remember what `set` stored

**Cause:** the two invocations aren't pointing at the same Redis instance, or aren't using the same `appName`/`userID`/`sessionID`.

**Fix:** confirm both invocations point at the same `REDIS_ADDR` and the same `appName`/`userID`/`sessionID` — `cmd/persistent-agent/main.go` hardcodes them as constants for this lab; a real application would derive them per-user.

### Tests fail to start a container

**Cause:** Docker isn't running, or Testcontainers can't reach it.

**Fix:** confirm Docker is running (`docker info`). The tests skip, rather than fail, if Testcontainers genuinely can't reach Docker at all — a build or connection error inside a running Docker is a different, real problem worth investigating separately.
