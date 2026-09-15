package redissession

import (
	"errors"
	"testing"
	"time"

	"google.golang.org/adk/v2/session"
)

// TestRedisState_GetSet covers session.State's Get/Set directly — real,
// exercised code paths (any tool calling ctx.State().Get/Set) that the
// borrowed conformance suite never exercises, since it reads state only via
// State().All() (through sessiontestsuite.Snapshot).
func TestRedisState_GetSet(t *testing.T) {
	s := newRedisState(map[string]any{"existing": "value"})

	got, err := s.Get("existing")
	if err != nil {
		t.Fatalf("Get(existing) error = %v", err)
	}
	if got != "value" {
		t.Errorf("Get(existing) = %v, want %q", got, "value")
	}

	if _, err := s.Get("missing"); !errors.Is(err, session.ErrStateKeyNotExist) {
		t.Errorf("Get(missing) error = %v, want session.ErrStateKeyNotExist", err)
	}

	if err := s.Set("new", "added"); err != nil {
		t.Fatalf("Set(new) error = %v", err)
	}
	got, err = s.Get("new")
	if err != nil {
		t.Fatalf("Get(new) error = %v", err)
	}
	if got != "added" {
		t.Errorf("Get(new) = %v, want %q", got, "added")
	}
}

func TestRedisState_SetOnNilMap(t *testing.T) {
	s := &redisState{}
	if err := s.Set("k", "v"); err != nil {
		t.Fatalf("Set() error = %v", err)
	}
	got, err := s.Get("k")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if got != "v" {
		t.Errorf("Get() = %v, want %q", got, "v")
	}
}

// TestRedisEvents_At covers session.Events' At directly — real, exercised
// code (anything indexing into event history by position) the conformance
// suite never calls, since it reads events only via Events().All().
func TestRedisEvents_At(t *testing.T) {
	first := &session.Event{ID: "first"}
	second := &session.Event{ID: "second"}
	e := newRedisEvents([]*session.Event{first, second})

	if got := e.At(0); got != first {
		t.Errorf("At(0) = %v, want %v", got, first)
	}
	if got := e.At(1); got != second {
		t.Errorf("At(1) = %v, want %v", got, second)
	}
	if got := e.At(-1); got != nil {
		t.Errorf("At(-1) = %v, want nil", got)
	}
	if got := e.At(2); got != nil {
		t.Errorf("At(2) = %v, want nil", got)
	}
}

func TestRedisSession_LastUpdateTime(t *testing.T) {
	now := time.Now()
	s := &redisSession{updateTime: now}
	if got := s.LastUpdateTime(); !got.Equal(now) {
		t.Errorf("LastUpdateTime() = %v, want %v", got, now)
	}
}
