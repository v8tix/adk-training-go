package personaltutor

import (
	"errors"
	"iter"
	"testing"

	"google.golang.org/adk/v2/agent"
	"google.golang.org/adk/v2/session"
)

// fakeState is a minimal map-backed session.State double, reused from
// module-10's own precedent (internal/agents/memory) — just enough to
// prove each tool's own logic (call Set/Get, handle missing keys), not to
// re-verify the SDK's own event-delta merging, which agent_test.go's real
// integration test already covers end-to-end.
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

// erroringState always fails — used to prove each tool propagates a
// genuine session.State failure unchanged, distinct from the
// ErrStateKeyNotExist "not found" case fakeState covers. Matches
// module-10's own precedent (internal/agents/memory/tools_test.go).
type erroringState struct {
	err error
}

func (s *erroringState) Get(string) (any, error) { return nil, s.err }
func (s *erroringState) Set(string, any) error   { return s.err }
func (s *erroringState) All() iter.Seq2[string, any] {
	return func(func(string, any) bool) {}
}

// setErroringState wraps a working fakeState but fails every Set call —
// used to prove a tool propagates a genuine write failure after its own
// reads already succeeded.
type setErroringState struct {
	*fakeState
	err error
}

func (s *setErroringState) Set(string, any) error { return s.err }

// getErroringState wraps a working fakeState but fails every Get call —
// used to prove a tool propagates a genuine read failure that occurs after
// an earlier write in the same call already succeeded.
type getErroringState struct {
	*fakeState
	err error
}

func (s *getErroringState) Get(string) (any, error) { return nil, s.err }

// fakeContext embeds the SDK's own agent.StrictContextMock and overrides
// only State() — every other Context method panics if a test accidentally
// calls it. state is the session.State interface, not the concrete
// *fakeState, so tests can swap in an erroring double without a second
// context type.
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

func TestSetUserPreferences(t *testing.T) {
	ctx := newFakeContext()

	got, err := setUserPreferences(ctx, SetUserPreferencesArgs{Language: "es", DifficultyLevel: "advanced"})
	if err != nil {
		t.Fatalf("setUserPreferences() error = %v", err)
	}
	if got.Status != "success" {
		t.Fatalf("setUserPreferences() Status = %q, want %q", got.Status, "success")
	}

	lang, _ := ctx.state.Get(userLanguageKey)
	if lang != "es" {
		t.Errorf("state[%q] = %v, want %q", userLanguageKey, lang, "es")
	}
	diff, _ := ctx.state.Get(userDifficultyKey)
	if diff != "advanced" {
		t.Errorf("state[%q] = %v, want %q", userDifficultyKey, diff, "advanced")
	}
}

func TestRecordTopicCompletion_AccumulatesAcrossCalls(t *testing.T) {
	ctx := newFakeContext()

	if _, err := recordTopicCompletion(ctx, RecordTopicCompletionArgs{Topic: "Python Basics", QuizScore: 80}); err != nil {
		t.Fatalf("recordTopicCompletion() error = %v", err)
	}
	got, err := recordTopicCompletion(ctx, RecordTopicCompletionArgs{Topic: "Data Structures", QuizScore: 90})
	if err != nil {
		t.Fatalf("recordTopicCompletion() error = %v", err)
	}

	if got.TopicsCount != 2 {
		t.Errorf("TopicsCount = %d, want 2 (accumulated, not overwritten)", got.TopicsCount)
	}
	topics, _ := ctx.state.Get(userTopicsKey)
	if len(topics.([]string)) != 2 {
		t.Errorf("state[%q] has %d topics, want 2", userTopicsKey, len(topics.([]string)))
	}
	scores, _ := ctx.state.Get(userScoresKey)
	scoreMap := scores.(map[string]int)
	if scoreMap["Python Basics"] != 80 || scoreMap["Data Structures"] != 90 {
		t.Errorf("state[%q] = %v, want both scores preserved", userScoresKey, scoreMap)
	}
}

func TestGetUserProgress_DefaultsForFirstTimeUser(t *testing.T) {
	ctx := newFakeContext()

	got, err := getUserProgress(ctx, GetUserProgressArgs{})
	if err != nil {
		t.Fatalf("getUserProgress() error = %v, want no error on empty state (first-time user)", err)
	}
	if got.TopicsCompleted != 0 || got.AverageQuizScore != 0 {
		t.Errorf("getUserProgress() on empty state = %+v, want all zero defaults", got)
	}
	if got.DifficultyLevel != "beginner" {
		t.Errorf("DifficultyLevel = %q, want default %q", got.DifficultyLevel, "beginner")
	}
}

func TestGetUserProgress_ComputesAverage(t *testing.T) {
	ctx := newFakeContext()
	if _, err := recordTopicCompletion(ctx, RecordTopicCompletionArgs{Topic: "A", QuizScore: 80}); err != nil {
		t.Fatalf("recordTopicCompletion() error = %v", err)
	}
	if _, err := recordTopicCompletion(ctx, RecordTopicCompletionArgs{Topic: "B", QuizScore: 100}); err != nil {
		t.Fatalf("recordTopicCompletion() error = %v", err)
	}

	got, err := getUserProgress(ctx, GetUserProgressArgs{})
	if err != nil {
		t.Fatalf("getUserProgress() error = %v", err)
	}
	if got.AverageQuizScore != 90 {
		t.Errorf("AverageQuizScore = %v, want 90", got.AverageQuizScore)
	}
}

func TestStartLearningSession_WritesUnprefixedKey(t *testing.T) {
	ctx := newFakeContext()
	if err := ctx.state.Set(userDifficultyKey, "intermediate"); err != nil {
		t.Fatalf("state.Set() error = %v", err)
	}

	got, err := startLearningSession(ctx, StartLearningSessionArgs{Topic: "Goroutines"})
	if err != nil {
		t.Fatalf("startLearningSession() error = %v", err)
	}
	if got.DifficultyLevel != "intermediate" {
		t.Errorf("DifficultyLevel = %q, want %q (read from user: state)", got.DifficultyLevel, "intermediate")
	}

	topic, err := ctx.state.Get(currentTopicKey)
	if err != nil {
		t.Fatalf("state.Get(%q) error = %v", currentTopicKey, err)
	}
	if topic != "Goroutines" {
		t.Errorf("state[%q] = %v, want %q — plain, unprefixed session key", currentTopicKey, topic, "Goroutines")
	}
}

func TestStartLearningSession_DefaultsDifficulty(t *testing.T) {
	ctx := newFakeContext()

	got, err := startLearningSession(ctx, StartLearningSessionArgs{Topic: "Channels"})
	if err != nil {
		t.Fatalf("startLearningSession() error = %v", err)
	}
	if got.DifficultyLevel != "beginner" {
		t.Errorf("DifficultyLevel = %q, want default %q when user:difficulty_level was never set", got.DifficultyLevel, "beginner")
	}
}

func TestCalculateQuizGrade(t *testing.T) {
	tests := []struct {
		name           string
		correct, total int
		wantGrade      string
		wantPercentage float64
	}{
		{name: "perfect score is an A", correct: 10, total: 10, wantGrade: "A", wantPercentage: 100},
		{name: "boundary at 90 is an A", correct: 9, total: 10, wantGrade: "A", wantPercentage: 90},
		{name: "boundary at 80 is a B", correct: 8, total: 10, wantGrade: "B", wantPercentage: 80},
		{name: "boundary at 70 is a C", correct: 7, total: 10, wantGrade: "C", wantPercentage: 70},
		{name: "boundary at 60 is a D", correct: 6, total: 10, wantGrade: "D", wantPercentage: 60},
		{name: "below 60 is an F", correct: 5, total: 10, wantGrade: "F", wantPercentage: 50},
		{name: "correct greater than total clamps to 100%", correct: 15, total: 10, wantGrade: "A", wantPercentage: 100},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := newFakeContext()

			got, err := calculateQuizGrade(ctx, CalculateQuizGradeArgs{CorrectAnswers: tt.correct, TotalQuestions: tt.total})
			if err != nil {
				t.Fatalf("calculateQuizGrade() error = %v", err)
			}
			if got.Grade != tt.wantGrade {
				t.Errorf("Grade = %q, want %q", got.Grade, tt.wantGrade)
			}
			if got.Percentage != tt.wantPercentage {
				t.Errorf("Percentage = %v, want %v", got.Percentage, tt.wantPercentage)
			}

			pct, _ := ctx.state.Get(tempPercentageKey)
			if pct != got.Percentage {
				t.Errorf("state[%q] = %v, want %v (temp: scoped intermediate value)", tempPercentageKey, pct, got.Percentage)
			}
			raw, _ := ctx.state.Get(tempRawScoreKey)
			if raw != tt.correct {
				t.Errorf("state[%q] = %v, want %d — the raw score reflects what was reported, unclamped", tempRawScoreKey, raw, tt.correct)
			}
		})
	}
}

func TestSearchPastLessons(t *testing.T) {
	ctx := newFakeContext()

	notFound, err := searchPastLessons(ctx, SearchPastLessonsArgs{Query: "Goroutines"})
	if err != nil {
		t.Fatalf("searchPastLessons() error = %v", err)
	}
	if notFound.Found {
		t.Error("Found = true on empty user:topics, want false")
	}

	if _, err := recordTopicCompletion(ctx, RecordTopicCompletionArgs{Topic: "Go Concurrency: Goroutines", QuizScore: 85}); err != nil {
		t.Fatalf("recordTopicCompletion() error = %v", err)
	}

	found, err := searchPastLessons(ctx, SearchPastLessonsArgs{Query: "goroutines"})
	if err != nil {
		t.Fatalf("searchPastLessons() error = %v", err)
	}
	if !found.Found {
		t.Error("Found = false after recording a matching topic, want true (case-insensitive substring match)")
	}
}

// The tests below prove each tool's error-propagation path is real, not
// dead code — fakeState alone can never trigger it, since fakeState's Get
// only ever returns nil or session.ErrStateKeyNotExist, and its Set never
// fails. Matches module-10's own precedent (internal/agents/memory).

func TestSetUserPreferences_PropagatesStateError(t *testing.T) {
	wantErr := errors.New("boom")
	ctx := newFakeContextWithState(&erroringState{err: wantErr})

	_, err := setUserPreferences(ctx, SetUserPreferencesArgs{Language: "es", DifficultyLevel: "advanced"})
	if !errors.Is(err, wantErr) {
		t.Fatalf("setUserPreferences() error = %v, want errors.Is(err, wantErr)", err)
	}
}

func TestRecordTopicCompletion_PropagatesGetError(t *testing.T) {
	wantErr := errors.New("boom")
	ctx := newFakeContextWithState(&erroringState{err: wantErr})

	_, err := recordTopicCompletion(ctx, RecordTopicCompletionArgs{Topic: "Goroutines", QuizScore: 80})
	if !errors.Is(err, wantErr) {
		t.Fatalf("recordTopicCompletion() error = %v, want errors.Is(err, wantErr)", err)
	}
}

func TestRecordTopicCompletion_PropagatesSetError(t *testing.T) {
	wantErr := errors.New("boom")
	ctx := newFakeContextWithState(&setErroringState{fakeState: &fakeState{}, err: wantErr})

	_, err := recordTopicCompletion(ctx, RecordTopicCompletionArgs{Topic: "Goroutines", QuizScore: 80})
	if !errors.Is(err, wantErr) {
		t.Fatalf("recordTopicCompletion() error = %v, want errors.Is(err, wantErr)", err)
	}
}

func TestGetUserProgress_PropagatesStateError(t *testing.T) {
	wantErr := errors.New("boom")
	ctx := newFakeContextWithState(&erroringState{err: wantErr})

	_, err := getUserProgress(ctx, GetUserProgressArgs{})
	if !errors.Is(err, wantErr) {
		t.Fatalf("getUserProgress() error = %v, want errors.Is(err, wantErr)", err)
	}
}

func TestStartLearningSession_PropagatesSetError(t *testing.T) {
	wantErr := errors.New("boom")
	ctx := newFakeContextWithState(&setErroringState{fakeState: &fakeState{}, err: wantErr})

	_, err := startLearningSession(ctx, StartLearningSessionArgs{Topic: "Goroutines"})
	if !errors.Is(err, wantErr) {
		t.Fatalf("startLearningSession() error = %v, want errors.Is(err, wantErr)", err)
	}
}

func TestStartLearningSession_PropagatesGetError(t *testing.T) {
	wantErr := errors.New("boom")
	ctx := newFakeContextWithState(&getErroringState{fakeState: &fakeState{}, err: wantErr})

	_, err := startLearningSession(ctx, StartLearningSessionArgs{Topic: "Goroutines"})
	if !errors.Is(err, wantErr) {
		t.Fatalf("startLearningSession() error = %v, want errors.Is(err, wantErr) — the current_topic write should have succeeded, only the difficulty_level read should fail", err)
	}
}

func TestCalculateQuizGrade_PropagatesSetError(t *testing.T) {
	wantErr := errors.New("boom")
	ctx := newFakeContextWithState(&setErroringState{fakeState: &fakeState{}, err: wantErr})

	_, err := calculateQuizGrade(ctx, CalculateQuizGradeArgs{CorrectAnswers: 8, TotalQuestions: 10})
	if !errors.Is(err, wantErr) {
		t.Fatalf("calculateQuizGrade() error = %v, want errors.Is(err, wantErr)", err)
	}
}

func TestSearchPastLessons_PropagatesGetError(t *testing.T) {
	wantErr := errors.New("boom")
	ctx := newFakeContextWithState(&erroringState{err: wantErr})

	_, err := searchPastLessons(ctx, SearchPastLessonsArgs{Query: "Goroutines"})
	if !errors.Is(err, wantErr) {
		t.Fatalf("searchPastLessons() error = %v, want errors.Is(err, wantErr)", err)
	}
}
