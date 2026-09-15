// Package redissession implements session.Service (google.golang.org/adk/v2/session)
// backed by Redis, demonstrating the SDK's own pluggable-persistence
// extension point — the direct equivalent of Python's BaseSessionService
// subclassing.
//
// A real, confirmed architectural difference from Python: session.Service
// is a plain Go interface with no default method bodies. Python's
// BaseSessionService.append_event has real default logic (applying
// event.actions.state_delta to session state, including its three-tier
// app:/user:/session-scoped split) that a subclass calls via
// super().append_event(...) before adding its own persistence. Go's
// interface has no base to delegate to — this package reimplements that
// logic itself, matching exactly what google.golang.org/adk/v2's own
// InMemoryService does internally (confirmed by reading session/inmemory.go
// and its internal/sessionutils helper directly, since neither is
// importable across module boundaries):
//   - a StateDelta key prefixed "app:" is shared across every session for
//     the same app, regardless of user;
//   - a key prefixed "user:" is shared across every session for the same
//     user within the app;
//   - any other key is private to the one session;
//   - a key prefixed "temp:" is visible on the same in-memory session
//     object for the rest of the current invocation, but is never written
//     to Redis — it does not survive a fresh Get.
//
// Correctness is verified against the SDK's own conformance suite,
// session/sessiontestsuite — the same suite the SDK uses to test its own
// built-in session/database (SQL/GORM) service — rather than a hand-written
// spot-check suite. That suite is what actually caught the three-tier
// scoping requirement above: an earlier version of this package treated all
// state as session-private and failed StateManagement/app_state_is_shared
// and StateManagement/user_state_is_user_specific until fixed. See
// service_test.go.
package redissession

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"google.golang.org/adk/v2/platform"
	"google.golang.org/adk/v2/session"
)

// ErrSessionAlreadyExists is returned by Create when the requested session
// ID already has a stored session, matching InMemoryService's own behavior.
var ErrSessionAlreadyExists = errors.New("redissession: session already exists")

// ErrCreateRequiresAppAndUser is returned by Create when AppName or UserID
// is empty.
var ErrCreateRequiresAppAndUser = errors.New("redissession: app_name and user_id are required")

// storedSession is the JSON shape persisted at the session's own metadata
// key — session-scoped state only; app- and user-scoped state live in
// their own separate Redis hashes (appStateKey, userStateKey).
type storedSession struct {
	State      map[string]any `json:"state"`
	UpdateTime time.Time      `json:"updateTime"`
}

type service struct {
	client *redis.Client
}

// NewService returns a session.Service backed by client.
func NewService(client *redis.Client) session.Service {
	return &service{client: client}
}

func sessionKey(appName, userID, sessionID string) string {
	return fmt.Sprintf("session:%s:%s:%s", appName, userID, sessionID)
}

func eventsKey(appName, userID, sessionID string) string {
	return sessionKey(appName, userID, sessionID) + ":events"
}

func userSessionSetKey(appName, userID string) string {
	return fmt.Sprintf("sessions:%s:%s", appName, userID)
}

// appSessionSetKey indexes every session in an app, across all users, as
// "userID/sessionID" members — needed for List when ListRequest.UserID is
// empty (the SDK's own doc comment marks it optional; the conformance suite
// exercises exactly this case).
func appSessionSetKey(appName string) string {
	return fmt.Sprintf("sessions:%s", appName)
}

func appStateKey(appName string) string {
	return fmt.Sprintf("app-state:%s", appName)
}

func userStateKey(appName, userID string) string {
	return fmt.Sprintf("user-state:%s:%s", appName, userID)
}

// extractStateDeltas splits delta by key prefix, matching the SDK's own
// (unexported, unimportable) sessionutils.ExtractStateDeltas exactly: a
// "temp:"-prefixed key is dropped from all three results.
func extractStateDeltas(delta map[string]any) (appDelta, userDelta, sessionDelta map[string]any) {
	appDelta = make(map[string]any)
	userDelta = make(map[string]any)
	sessionDelta = make(map[string]any)
	for key, value := range delta {
		switch {
		case strings.HasPrefix(key, session.KeyPrefixApp):
			appDelta[strings.TrimPrefix(key, session.KeyPrefixApp)] = value
		case strings.HasPrefix(key, session.KeyPrefixUser):
			userDelta[strings.TrimPrefix(key, session.KeyPrefixUser)] = value
		case strings.HasPrefix(key, session.KeyPrefixTemp):
			// dropped
		default:
			sessionDelta[key] = value
		}
	}
	return appDelta, userDelta, sessionDelta
}

// mergeStates re-adds the app:/user: prefixes and combines all three scopes
// into the single flat map callers see through session.State.
func mergeStates(appState, userState, sessionState map[string]any) map[string]any {
	merged := make(map[string]any, len(appState)+len(userState)+len(sessionState))
	maps.Copy(merged, sessionState)
	for k, v := range appState {
		merged[session.KeyPrefixApp+k] = v
	}
	for k, v := range userState {
		merged[session.KeyPrefixUser+k] = v
	}
	return merged
}

// loadScopedState reads the app- and user-level state hashes for
// appName/userID, JSON-decoding each field's value.
func (s *service) loadScopedState(ctx context.Context, appName, userID string) (appState, userState map[string]any, err error) {
	appState, err = s.loadHash(ctx, appStateKey(appName))
	if err != nil {
		return nil, nil, fmt.Errorf("loading app state: %w", err)
	}
	userState, err = s.loadHash(ctx, userStateKey(appName, userID))
	if err != nil {
		return nil, nil, fmt.Errorf("loading user state: %w", err)
	}
	return appState, userState, nil
}

func (s *service) loadHash(ctx context.Context, key string) (map[string]any, error) {
	raw, err := s.client.HGetAll(ctx, key).Result()
	if err != nil {
		return nil, err
	}
	out := make(map[string]any, len(raw))
	for field, value := range raw {
		var decoded any
		if err := json.Unmarshal([]byte(value), &decoded); err != nil {
			return nil, fmt.Errorf("unmarshaling field %q: %w", field, err)
		}
		out[field] = decoded
	}
	return out, nil
}

// mergeHash writes delta's fields into the hash at key, each JSON-encoded,
// via HSET (a no-op if delta is empty).
func (s *service) mergeHash(ctx context.Context, key string, delta map[string]any) error {
	if len(delta) == 0 {
		return nil
	}
	fields := make(map[string]any, len(delta))
	for k, v := range delta {
		encoded, err := json.Marshal(v)
		if err != nil {
			return fmt.Errorf("marshaling field %q: %w", k, err)
		}
		fields[k] = encoded
	}
	return s.client.HSet(ctx, key, fields).Err()
}

func (s *service) Create(ctx context.Context, req *session.CreateRequest) (*session.CreateResponse, error) {
	if req.AppName == "" || req.UserID == "" {
		return nil, fmt.Errorf("%w: got app_name %q, user_id %q", ErrCreateRequiresAppAndUser, req.AppName, req.UserID)
	}

	sessionID := req.SessionID
	if sessionID == "" {
		sessionID = platform.NewUUID(ctx)
	}

	key := sessionKey(req.AppName, req.UserID, sessionID)
	exists, err := s.client.Exists(ctx, key).Result()
	if err != nil {
		return nil, fmt.Errorf("checking existing session: %w", err)
	}
	if exists > 0 {
		return nil, fmt.Errorf("%w: %q", ErrSessionAlreadyExists, sessionID)
	}

	appDelta, userDelta, sessionDelta := extractStateDeltas(req.State)
	if err := s.mergeHash(ctx, appStateKey(req.AppName), appDelta); err != nil {
		return nil, fmt.Errorf("storing initial app state: %w", err)
	}
	if err := s.mergeHash(ctx, userStateKey(req.AppName, req.UserID), userDelta); err != nil {
		return nil, fmt.Errorf("storing initial user state: %w", err)
	}

	stored := storedSession{State: sessionDelta, UpdateTime: time.Now()}
	data, err := json.Marshal(stored)
	if err != nil {
		return nil, fmt.Errorf("marshaling new session: %w", err)
	}
	if err := s.client.Set(ctx, key, data, 0).Err(); err != nil {
		return nil, fmt.Errorf("storing new session: %w", err)
	}
	if err := s.client.SAdd(ctx, userSessionSetKey(req.AppName, req.UserID), sessionID).Err(); err != nil {
		return nil, fmt.Errorf("indexing new session: %w", err)
	}
	if err := s.client.SAdd(ctx, appSessionSetKey(req.AppName), req.UserID+"/"+sessionID).Err(); err != nil {
		return nil, fmt.Errorf("indexing new session app-wide: %w", err)
	}

	appState, userState, err := s.loadScopedState(ctx, req.AppName, req.UserID)
	if err != nil {
		return nil, err
	}

	return &session.CreateResponse{
		Session: &redisSession{
			id:         sessionID,
			appName:    req.AppName,
			userID:     req.UserID,
			state:      newRedisState(mergeStates(appState, userState, sessionDelta)),
			events:     newRedisEvents(nil),
			updateTime: stored.UpdateTime,
		},
	}, nil
}

func (s *service) Get(ctx context.Context, req *session.GetRequest) (*session.GetResponse, error) {
	key := sessionKey(req.AppName, req.UserID, req.SessionID)
	data, err := s.client.Get(ctx, key).Bytes()
	if errors.Is(err, redis.Nil) {
		return nil, fmt.Errorf("%w: %q", session.ErrNotFound, req.SessionID)
	}
	if err != nil {
		return nil, fmt.Errorf("loading session: %w", err)
	}

	var stored storedSession
	if err := json.Unmarshal(data, &stored); err != nil {
		return nil, fmt.Errorf("unmarshaling session: %w", err)
	}

	appState, userState, err := s.loadScopedState(ctx, req.AppName, req.UserID)
	if err != nil {
		return nil, err
	}

	events, err := s.loadEvents(ctx, req.AppName, req.UserID, req.SessionID, req.NumRecentEvents, req.After)
	if err != nil {
		return nil, fmt.Errorf("loading events: %w", err)
	}

	return &session.GetResponse{
		Session: &redisSession{
			id:         req.SessionID,
			appName:    req.AppName,
			userID:     req.UserID,
			state:      newRedisState(mergeStates(appState, userState, stored.State)),
			events:     newRedisEvents(events),
			updateTime: stored.UpdateTime,
		},
	}, nil
}

func (s *service) loadEvents(ctx context.Context, appName, userID, sessionID string, numRecent int, after time.Time) ([]*session.Event, error) {
	raw, err := s.client.LRange(ctx, eventsKey(appName, userID, sessionID), 0, -1).Result()
	if err != nil {
		return nil, err
	}

	events := make([]*session.Event, 0, len(raw))
	for _, item := range raw {
		var ev session.Event
		if err := json.Unmarshal([]byte(item), &ev); err != nil {
			return nil, fmt.Errorf("unmarshaling event: %w", err)
		}
		if !after.IsZero() && ev.Timestamp.Before(after) {
			continue
		}
		events = append(events, &ev)
	}

	if numRecent > 0 && len(events) > numRecent {
		events = events[len(events)-numRecent:]
	}
	return events, nil
}

func (s *service) List(ctx context.Context, req *session.ListRequest) (*session.ListResponse, error) {
	var members []string
	var err error
	if req.UserID == "" {
		members, err = s.client.SMembers(ctx, appSessionSetKey(req.AppName)).Result()
	} else {
		var ids []string
		ids, err = s.client.SMembers(ctx, userSessionSetKey(req.AppName, req.UserID)).Result()
		for _, id := range ids {
			members = append(members, req.UserID+"/"+id)
		}
	}
	if err != nil {
		return nil, fmt.Errorf("listing sessions: %w", err)
	}

	sessions := make([]session.Session, 0, len(members))
	for _, member := range members {
		userID, sessionID, ok := strings.Cut(member, "/")
		if !ok {
			continue
		}
		resp, err := s.Get(ctx, &session.GetRequest{AppName: req.AppName, UserID: userID, SessionID: sessionID})
		if errors.Is(err, session.ErrNotFound) {
			continue // stale index entry — the session was deleted since
		}
		if err != nil {
			return nil, err
		}
		sessions = append(sessions, resp.Session)
	}

	return &session.ListResponse{Sessions: sessions}, nil
}

func (s *service) Delete(ctx context.Context, req *session.DeleteRequest) error {
	key := sessionKey(req.AppName, req.UserID, req.SessionID)
	pipe := s.client.TxPipeline()
	pipe.Del(ctx, key)
	pipe.Del(ctx, eventsKey(req.AppName, req.UserID, req.SessionID))
	pipe.SRem(ctx, userSessionSetKey(req.AppName, req.UserID), req.SessionID)
	pipe.SRem(ctx, appSessionSetKey(req.AppName), req.UserID+"/"+req.SessionID)
	if _, err := pipe.Exec(ctx); err != nil {
		return fmt.Errorf("deleting session: %w", err)
	}
	return nil
}

func (s *service) AppendEvent(ctx context.Context, curSession session.Session, event *session.Event) error {
	if curSession == nil {
		return fmt.Errorf("redissession: session is nil")
	}
	if event == nil {
		return fmt.Errorf("redissession: event is nil")
	}
	if event.Partial {
		return nil
	}
	if event.ID == "" {
		event.ID = platform.NewUUID(ctx)
	}

	key := sessionKey(curSession.AppName(), curSession.UserID(), curSession.ID())
	data, err := s.client.Get(ctx, key).Bytes()
	if errors.Is(err, redis.Nil) {
		return fmt.Errorf("%w: %q, cannot apply event", session.ErrNotFound, curSession.ID())
	}
	if err != nil {
		return fmt.Errorf("loading session for append: %w", err)
	}

	var stored storedSession
	if err := json.Unmarshal(data, &stored); err != nil {
		return fmt.Errorf("unmarshaling session for append: %w", err)
	}
	if stored.State == nil {
		stored.State = make(map[string]any)
	}

	// Apply the state delta — the three-tier app:/user:/session split Python's
	// BaseSessionService provides by default and a subclass inherits via
	// super().append_event(); Go's session.Service has no such base, so this
	// implementation owns it, matching the SDK's own InMemoryService.
	appDelta, userDelta, sessionDelta := extractStateDeltas(event.Actions.StateDelta)
	if err := s.mergeHash(ctx, appStateKey(curSession.AppName()), appDelta); err != nil {
		return fmt.Errorf("updating app state: %w", err)
	}
	if err := s.mergeHash(ctx, userStateKey(curSession.AppName(), curSession.UserID()), userDelta); err != nil {
		return fmt.Errorf("updating user state: %w", err)
	}
	maps.Copy(stored.State, sessionDelta)
	stored.UpdateTime = event.Timestamp

	// The persisted event record keeps its original StateDelta (app:/user:
	// prefixes intact, as the real historical record of what was applied),
	// minus any temp:-prefixed key — that part must never be replayed as
	// durable history, matching the SDK's own trimTempDeltaState.
	persistedEvent := trimTempDeltaState(event)

	updatedData, err := json.Marshal(stored)
	if err != nil {
		return fmt.Errorf("marshaling updated session: %w", err)
	}
	eventData, err := json.Marshal(persistedEvent)
	if err != nil {
		return fmt.Errorf("marshaling event: %w", err)
	}

	pipe := s.client.TxPipeline()
	pipe.Set(ctx, key, updatedData, 0)
	pipe.RPush(ctx, eventsKey(curSession.AppName(), curSession.UserID(), curSession.ID()), eventData)
	if _, err := pipe.Exec(ctx); err != nil {
		return fmt.Errorf("persisting event: %w", err)
	}

	// Update the in-memory view handed to the caller so it reflects the
	// change immediately, matching InMemoryService's own same-call behavior
	// — including a "temp:"-prefixed key, which must be visible on this same
	// session object for the rest of the current invocation (the SDK's own
	// instruction-template resolution and multi-step workflow patterns read
	// it that way) even though it's never persisted. Confirmed live this
	// module (Phase 5 review): using the already-scoped/temp-stripped
	// mergeStates(appDelta, userDelta, sessionDelta) here instead of the raw
	// delta silently dropped temp: state even within the same invocation —
	// a real, fixed bug.
	if rs, ok := curSession.(*redisSession); ok {
		rs.state.mu.Lock()
		if rs.state.state == nil {
			rs.state.state = make(map[string]any)
		}
		maps.Copy(rs.state.state, event.Actions.StateDelta)
		rs.state.mu.Unlock()

		rs.events.mu.Lock()
		rs.events.events = append(rs.events.events, persistedEvent)
		rs.events.mu.Unlock()

		rs.updateTime = stored.UpdateTime
	}

	return nil
}

// trimTempDeltaState returns a copy of event with any "temp:"-prefixed
// StateDelta keys removed, so a transient, turn-scoped value is applied to
// state (via AppendEvent above) but never durably persisted in the event
// record itself — matching InMemoryService's own trimTempDeltaState.
func trimTempDeltaState(event *session.Event) *session.Event {
	if len(event.Actions.StateDelta) == 0 {
		return event
	}

	filtered := make(map[string]any, len(event.Actions.StateDelta))
	for k, v := range event.Actions.StateDelta {
		if !strings.HasPrefix(k, session.KeyPrefixTemp) {
			filtered[k] = v
		}
	}
	if len(filtered) == len(event.Actions.StateDelta) {
		return event
	}

	eventCopy := *event
	eventCopy.Actions.StateDelta = filtered
	return &eventCopy
}
