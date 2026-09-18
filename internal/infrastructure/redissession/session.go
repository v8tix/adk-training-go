package redissession

import (
	"iter"
	"maps"
	"sync"
	"time"

	"google.golang.org/adk/v2/session"
)

// redisSession is an in-memory view of one session, loaded from Redis by
// the service and handed to the caller. It implements session.Session.
// Like the SDK's own InMemoryService, storage I/O lives in the service —
// these types are pure, in-memory snapshots.
type redisSession struct {
	id         string
	appName    string
	userID     string
	state      *redisState
	events     *redisEvents
	updateTime time.Time
}

func (s *redisSession) ID() string                { return s.id }
func (s *redisSession) AppName() string           { return s.appName }
func (s *redisSession) UserID() string            { return s.userID }
func (s *redisSession) State() session.State      { return s.state }
func (s *redisSession) Events() session.Events    { return s.events }
func (s *redisSession) LastUpdateTime() time.Time { return s.updateTime }

// redisState implements session.State: a mutex-protected map, matching the
// SDK's own InMemoryService state representation.
type redisState struct {
	mu    sync.RWMutex
	state map[string]any
}

func newRedisState(initial map[string]any) *redisState {
	if initial == nil {
		initial = make(map[string]any)
	}
	return &redisState{state: initial}
}

func (s *redisState) Get(key string) (any, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	val, ok := s.state[key]
	if !ok {
		return nil, session.ErrStateKeyNotExist
	}
	return val, nil
}

func (s *redisState) Set(key string, val any) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.state == nil {
		s.state = make(map[string]any)
	}
	s.state[key] = val
	return nil
}

func (s *redisState) All() iter.Seq2[string, any] {
	s.mu.RLock()
	snapshot := maps.Clone(s.state)
	s.mu.RUnlock()

	return maps.All(snapshot)
}

// redisEvents implements session.Events: an ordered, mutex-protected
// snapshot of events loaded from Redis.
type redisEvents struct {
	mu     sync.RWMutex
	events []*session.Event
}

func newRedisEvents(events []*session.Event) *redisEvents {
	return &redisEvents{events: events}
}

func (e *redisEvents) All() iter.Seq[*session.Event] {
	e.mu.RLock()
	snapshot := make([]*session.Event, len(e.events))
	copy(snapshot, e.events)
	e.mu.RUnlock()

	return func(yield func(*session.Event) bool) {
		for _, ev := range snapshot {
			if !yield(ev) {
				return
			}
		}
	}
}

func (e *redisEvents) Len() int {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return len(e.events)
}

func (e *redisEvents) At(i int) *session.Event {
	e.mu.RLock()
	defer e.mu.RUnlock()
	if i < 0 || i >= len(e.events) {
		return nil
	}
	return e.events[i]
}
