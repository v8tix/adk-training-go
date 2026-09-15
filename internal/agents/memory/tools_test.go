package memory

import (
	"errors"
	"iter"
	"testing"

	"google.golang.org/adk/v2/agent"
	"google.golang.org/adk/v2/session"
)

// fakeState is a minimal map-backed session.State double — just enough to
// prove storeName/recallName's own logic (call Set/Get, handle the
// not-found case), not to re-verify the SDK's own event-delta merging,
// which agent_test.go's real integration test already covers end-to-end.
type fakeState struct {
	data map[string]any
}

func (s *fakeState) Get(key string) (any, error) {
	if val, ok := s.data[key]; ok {
		return val, nil
	}
	return nil, session.ErrStateKeyNotExist
}

func (s *fakeState) Set(key string, val any) error {
	if s.data == nil {
		s.data = make(map[string]any)
	}
	s.data[key] = val
	return nil
}

func (s *fakeState) All() iter.Seq2[string, any] {
	return func(yield func(string, any) bool) {
		for k, v := range s.data {
			if !yield(k, v) {
				return
			}
		}
	}
}

// erroringState always fails — used to prove storeName/recallName propagate
// a genuine session.State failure unchanged, distinct from the
// ErrStateKeyNotExist "not found" case fakeState covers.
type erroringState struct {
	err error
}

func (s *erroringState) Get(string) (any, error) { return nil, s.err }
func (s *erroringState) Set(string, any) error   { return s.err }
func (s *erroringState) All() iter.Seq2[string, any] {
	return func(func(string, any) bool) {}
}

// fakeContext embeds the SDK's own agent.StrictContextMock and overrides
// only State() — every other Context method panics if a test accidentally
// calls it, exactly the loud-failure behavior StrictContextMock exists for.
// state is the session.State interface, not the concrete *fakeState, so
// tests can swap in erroringState without a second context type.
type fakeContext struct {
	agent.StrictContextMock
	state session.State
}

func (c *fakeContext) State() session.State {
	return c.state
}

func newFakeContext() *fakeContext {
	return &fakeContext{state: &fakeState{}}
}

func newFakeContextWithState(s session.State) *fakeContext {
	return &fakeContext{state: s}
}

func TestStoreName(t *testing.T) {
	ctx := newFakeContext()

	got, err := storeName(ctx, StoreNameArgs{Name: "Mario"})
	if err != nil {
		t.Fatalf("storeName() error = %v", err)
	}
	if got.Status != "success" {
		t.Fatalf("storeName() Status = %q, want %q", got.Status, "success")
	}

	stored, err := ctx.state.Get(stateKey)
	if err != nil {
		t.Fatalf("state.Get(%q) error = %v", stateKey, err)
	}
	if stored != "Mario" {
		t.Errorf("state[%q] = %v, want %q", stateKey, stored, "Mario")
	}
}

func TestRecallName(t *testing.T) {
	tests := []struct {
		name     string
		preset   string // empty means don't store anything first
		wantName string
	}{
		{name: "recalls a previously stored name", preset: "Mario", wantName: "Mario"},
		{name: "defaults to Stranger when nothing was stored", preset: "", wantName: "Stranger"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := newFakeContext()
			if tt.preset != "" {
				if err := ctx.state.Set(stateKey, tt.preset); err != nil {
					t.Fatalf("state.Set() error = %v", err)
				}
			}

			got, err := recallName(ctx, RecallNameArgs{})
			if err != nil {
				t.Fatalf("recallName() error = %v", err)
			}
			if got.Name != tt.wantName {
				t.Errorf("recallName() Name = %q, want %q", got.Name, tt.wantName)
			}
		})
	}
}

// TestStoreName_PropagatesStateError proves storeName's error path is real,
// not dead code — fakeState alone can never trigger it, since fakeState.Set
// never fails.
func TestStoreName_PropagatesStateError(t *testing.T) {
	wantErr := errors.New("boom")
	ctx := newFakeContextWithState(&erroringState{err: wantErr})

	_, err := storeName(ctx, StoreNameArgs{Name: "Mario"})
	if !errors.Is(err, wantErr) {
		t.Fatalf("storeName() error = %v, want errors.Is(err, wantErr)", err)
	}
}

// TestRecallName_PropagatesStateError proves recallName distinguishes a
// genuine state failure from the "key not found" case — fakeState alone can
// never trigger this branch, since fakeState.Get only ever returns nil or
// session.ErrStateKeyNotExist.
func TestRecallName_PropagatesStateError(t *testing.T) {
	wantErr := errors.New("boom")
	ctx := newFakeContextWithState(&erroringState{err: wantErr})

	_, err := recallName(ctx, RecallNameArgs{})
	if !errors.Is(err, wantErr) {
		t.Fatalf("recallName() error = %v, want errors.Is(err, wantErr)", err)
	}
}
