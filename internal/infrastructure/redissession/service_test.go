package redissession

import (
	"context"
	"testing"

	"github.com/redis/go-redis/v9"
	tcredis "github.com/testcontainers/testcontainers-go/modules/redis"
	"google.golang.org/adk/v2/session"
	"google.golang.org/adk/v2/session/sessiontestsuite"
)

// newTestRedisClient starts a real, ephemeral Redis container via
// Testcontainers and returns a connected client, cleaned up (client closed,
// container terminated) when the test ends. Skips (not fails) if
// Docker/Testcontainers can't start a container, matching this repo's own
// "skip on unavailable external dependency" discipline (the
// OllamaReachable pattern, adapted for Docker). Extracted during Phase 6
// simplify after this exact sequence appeared twice.
func newTestRedisClient(ctx context.Context, t *testing.T) *redis.Client {
	t.Helper()

	container, err := tcredis.Run(ctx, "redis:7-alpine")
	if err != nil {
		t.Skipf("skipping: could not start a Redis container via Testcontainers (%v) — Docker may be unavailable", err)
	}
	t.Cleanup(func() {
		if err := container.Terminate(ctx); err != nil {
			t.Logf("terminating container: %v", err)
		}
	})

	connStr, err := container.ConnectionString(ctx)
	if err != nil {
		t.Fatalf("ConnectionString() error = %v", err)
	}
	opts, err := redis.ParseURL(connStr)
	if err != nil {
		t.Fatalf("ParseURL() error = %v", err)
	}
	client := redis.NewClient(opts)
	t.Cleanup(func() { client.Close() })
	return client
}

// TestRedisSessionService verifies this package's session.Service
// implementation against the SDK's own conformance suite
// (session/sessiontestsuite) — the same suite the SDK uses to test its own
// built-in session/database (SQL/GORM) service — rather than a hand-written
// spot-check suite. One real Redis container (via Testcontainers) is
// started for the whole test; setup flushes it before each subtest for
// isolation, since starting a fresh container per subtest would be far
// slower for no additional correctness benefit.
func TestRedisSessionService(t *testing.T) {
	ctx := context.Background()
	client := newTestRedisClient(ctx, t)

	sessiontestsuite.RunServiceTests(t, sessiontestsuite.SuiteOptions{
		SupportsUserProvidedSessionID: true,
	}, func(t *testing.T) session.Service {
		t.Helper()
		if err := client.FlushDB(ctx).Err(); err != nil {
			t.Fatalf("FlushDB() error = %v", err)
		}
		return NewService(client)
	})
}

// TestAppendEvent_TempStateVisibleWithinSameInvocation is a permanent
// regression guard for a real bug found in Phase 5 review: AppendEvent's
// in-memory mutation of the caller's own session object used the
// already-scoped, temp-stripped delta instead of the raw one, silently
// dropping "temp:"-prefixed state even for the rest of the *same*
// invocation — not just across a fresh Get, which sessiontestsuite's own
// temp_state_is_not_persisted test does not check (it only re-fetches via a
// fresh Get). Confirmed against the real SDK (session.InMemoryService)
// that a "temp:" key must remain visible on the same session object after
// AppendEvent returns — the SDK's own instruction-template resolution and
// multi-step workflow patterns (e.g. examples/workflowagents/sequentialCode)
// rely on exactly this.
func TestAppendEvent_TempStateVisibleWithinSameInvocation(t *testing.T) {
	ctx := context.Background()
	client := newTestRedisClient(ctx, t)

	svc := NewService(client)
	created, err := svc.Create(ctx, &session.CreateRequest{AppName: "app", UserID: "u1"})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if err := svc.AppendEvent(ctx, created.Session, &session.Event{
		ID:           "event1",
		Author:       "user",
		InvocationID: "inv1",
		Actions:      session.EventActions{StateDelta: map[string]any{"temp:scratch": "hello", "sk": "v"}},
	}); err != nil {
		t.Fatalf("AppendEvent() error = %v", err)
	}

	// The SAME session object, not a fresh Get — this is exactly what
	// sessiontestsuite's temp_state_is_not_persisted does not check.
	got, err := created.Session.State().Get("temp:scratch")
	if err != nil {
		t.Fatalf("State().Get(temp:scratch) on the same session object error = %v, want it visible for the rest of this invocation", err)
	}
	if got != "hello" {
		t.Errorf("State().Get(temp:scratch) = %v, want %q", got, "hello")
	}

	// A fresh Get must NOT see it — the persistence-side guarantee, still
	// covered by sessiontestsuite too, reasserted here for a clear contrast.
	fresh, err := svc.Get(ctx, &session.GetRequest{AppName: "app", UserID: "u1", SessionID: created.Session.ID()})
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if _, err := fresh.Session.State().Get("temp:scratch"); err == nil {
		t.Error("a fresh Get() saw temp:scratch — it must never be persisted")
	}
}
