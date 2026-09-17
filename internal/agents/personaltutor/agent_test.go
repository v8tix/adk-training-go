package personaltutor

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/v8tix/adk-training-go/internal/infrastructure/llm"
	"google.golang.org/adk/v2/agent"
	"google.golang.org/adk/v2/artifact"
	"google.golang.org/adk/v2/runner"
	"google.golang.org/adk/v2/session"
	"google.golang.org/genai"
)

// testConfig and ollamaReachable are set once by TestMain, matching
// internal/agents/memory's own pattern.
var (
	testConfig      llm.Config
	ollamaReachable bool
)

func TestMain(m *testing.M) {
	testConfig = llm.LoadConfig()
	ollamaReachable = llm.OllamaReachable(testConfig, 2*time.Second)
	os.Exit(m.Run())
}

// askTutor drives the real BuildRootAgent through a single Run() call. It
// returns the final text answer, plus every tool's own FunctionResponse seen
// in that turn, keyed by tool name — the structural signal this test relies
// on instead of parsing the model's free-text reply, extending module-10's
// askMemoryAgent to a turn that may call more than one tool.
func askTutor(ctx context.Context, r *runner.Runner, userID, sessionID, message string) (answer string, toolResponses map[string]map[string]any, err error) {
	toolResponses = make(map[string]map[string]any)
	msg := genai.NewContentFromText(message, genai.RoleUser)
	for event, err := range r.Run(ctx, userID, sessionID, msg, agent.RunConfig{}) {
		if err != nil {
			return "", nil, err
		}
		if event.Content == nil {
			continue
		}
		for _, p := range event.Content.Parts {
			if p.FunctionResponse != nil {
				toolResponses[p.FunctionResponse.Name] = p.FunctionResponse.Response
			}
			if event.IsFinalResponse() && p.Text != "" && !p.Thought {
				answer = p.Text
			}
		}
	}
	return answer, toolResponses, nil
}

// TestPersonalTutor_PersistsUserStateAcrossTurns_Ollama exercises this
// module's own finding against the local Ollama backend.
func TestPersonalTutor_PersistsUserStateAcrossTurns_Ollama(t *testing.T) {
	if !llm.SelectedTestBackend().IncludesOllama() {
		t.Skip("skipping: TEST_BACKEND excludes ollama")
	}
	if !ollamaReachable {
		t.Skip("skipping: local Ollama server (" + testConfig.OllamaBaseURL + ") is not reachable")
	}

	cfg := testConfig
	cfg.ModelType = llm.ModelTypeOllama
	assertTutorPersistsState(t, cfg)
}

// TestPersonalTutor_PersistsUserStateAcrossTurns_Gemini is the same
// behavior against the real Gemini backend.
func TestPersonalTutor_PersistsUserStateAcrossTurns_Gemini(t *testing.T) {
	if !llm.SelectedTestBackend().IncludesGemini() {
		t.Skip("skipping: TEST_BACKEND excludes gemini")
	}
	if testConfig.GoogleAPIKey == "" {
		t.Skip("skipping: GOOGLE_AI_STUDIO_API_KEY is not set")
	}

	cfg := testConfig
	cfg.ModelType = llm.ModelTypeGemini
	assertTutorPersistsState(t, cfg)
}

// assertTutorPersistsState drives four SEPARATE Run() calls against the
// same session — not four messages within one call — since cross-Run()
// persistence of user: state, and the equally real non-persistence of
// temp: state, are the actual behaviors this module is about. The session
// service is built explicitly (rather than via runner.NewInMemory) so the
// test can call sessionService.Get directly afterward and inspect state
// with no model text-parsing involved.
func assertTutorPersistsState(t *testing.T, cfg llm.Config) {
	t.Helper()

	llmModel, _, err := llm.BuildModel(t.Context(), cfg)
	if err != nil {
		t.Fatalf("BuildModel() error = %v", err)
	}

	tutorAgent, err := BuildRootAgent(llmModel)
	if err != nil {
		t.Fatalf("BuildRootAgent() error = %v", err)
	}

	const appName = "personal_tutor_test_app"
	sessionService := session.InMemoryService()
	r, err := runner.New(runner.Config{
		AppName:           appName,
		Agent:             tutorAgent,
		SessionService:    sessionService,
		ArtifactService:   artifact.InMemoryService(),
		AutoCreateSession: true,
	})
	if err != nil {
		t.Fatalf("building runner: %v", err)
	}

	const (
		userID    = "test_user"
		sessionID = "test_session"
	)
	skipIfExhausted := func(turn string, err error) {
		t.Helper()
		if err != nil && strings.Contains(err.Error(), "RESOURCE_EXHAUSTED") {
			t.Skipf("skipping: Gemini free-tier quota exhausted for today (%v) — this is an external rate limit, not a code defect; re-run once it resets (%s)", err, turn)
		}
	}

	// Turn 1: set preferences.
	_, turn1Tools, err := askTutor(t.Context(), r, userID, sessionID, "Please set my preferred language to Spanish and my difficulty level to intermediate.")
	skipIfExhausted("turn 1", err)
	if err != nil {
		t.Fatalf("askTutor(turn 1) error = %v", err)
	}
	if turn1Tools["set_user_preferences"] == nil {
		t.Fatal("turn 1: set_user_preferences was not called")
	}

	// Turn 2: start a learning session, on a SEPARATE Run() call — proving
	// user:difficulty_level set in turn 1 is visible here.
	_, turn2Tools, err := askTutor(t.Context(), r, userID, sessionID, "I'd like to start learning about Goroutines now.")
	skipIfExhausted("turn 2", err)
	if err != nil {
		t.Fatalf("askTutor(turn 2) error = %v", err)
	}
	startResp := turn2Tools["start_learning_session"]
	if startResp == nil {
		t.Fatal("turn 2: start_learning_session was not called")
	}
	if got, _ := startResp["difficulty_level"].(string); !strings.EqualFold(got, "intermediate") {
		t.Errorf("turn 2: start_learning_session's difficulty_level = %v, want %q (read back from turn 1's user: state)", startResp["difficulty_level"], "intermediate")
	}

	// Turn 3: grade and record a quiz — writes temp: keys that must not
	// survive past this invocation.
	_, turn3Tools, err := askTutor(t.Context(), r, userID, sessionID, "I just took the Goroutines quiz and got 8 out of 10 correct. Please grade it, then record my completion of the Goroutines topic with that score.")
	skipIfExhausted("turn 3", err)
	if err != nil {
		t.Fatalf("askTutor(turn 3) error = %v", err)
	}
	if turn3Tools["calculate_quiz_grade"] == nil {
		t.Fatal("turn 3: calculate_quiz_grade was not called")
	}
	if turn3Tools["record_topic_completion"] == nil {
		t.Fatal("turn 3: record_topic_completion was not called")
	}

	// Turn 4: ask for overall progress — a fourth, separate Run() call.
	answer, turn4Tools, err := askTutor(t.Context(), r, userID, sessionID, "How is my overall learning progress so far?")
	skipIfExhausted("turn 4", err)
	if err != nil {
		t.Fatalf("askTutor(turn 4) error = %v", err)
	}
	progressResp := turn4Tools["get_user_progress"]
	if progressResp == nil {
		t.Fatal("turn 4: get_user_progress was not called")
	}
	if got, _ := progressResp["topics_completed"].(float64); got != 1 {
		t.Errorf("turn 4: get_user_progress topics_completed = %v, want 1", progressResp["topics_completed"])
	}
	if answer == "" {
		t.Error("turn 4: agent returned no final answer")
	}

	// Structural proof, read directly from the session service rather than
	// parsed from any model reply: user: state and the plain, unprefixed
	// current_topic survived all four separate Run() calls, while temp:
	// state written mid-turn-3 did not survive past that one invocation.
	getResp, err := sessionService.Get(t.Context(), &session.GetRequest{
		AppName:   appName,
		UserID:    userID,
		SessionID: sessionID,
	})
	if err != nil {
		t.Fatalf("sessionService.Get() error = %v", err)
	}
	state := getResp.Session.State()

	if lang, err := state.Get(userLanguageKey); err != nil {
		t.Errorf("state[%q] error = %v, want the language set in turn 1 to have persisted", userLanguageKey, err)
	} else if s, _ := lang.(string); s == "" {
		t.Errorf("state[%q] = %q, want a non-empty language persisted from turn 1", userLanguageKey, s)
	}

	if topic, err := state.Get(currentTopicKey); err != nil {
		t.Errorf("state[%q] error = %v, want the topic started in turn 2 to have persisted", currentTopicKey, err)
	} else if s, _ := topic.(string); !strings.Contains(strings.ToLower(s), "goroutine") {
		t.Errorf("state[%q] = %q, want it to mention Goroutines", currentTopicKey, s)
	}

	if topics, err := state.Get(userTopicsKey); err != nil {
		t.Errorf("state[%q] error = %v, want the topic recorded in turn 3 to have persisted", userTopicsKey, err)
	} else if ts, _ := topics.([]string); len(ts) == 0 {
		t.Errorf("state[%q] = %v, want at least one recorded topic", userTopicsKey, topics)
	}

	if _, err := state.Get(tempPercentageKey); !errors.Is(err, session.ErrStateKeyNotExist) {
		t.Errorf("state[%q] lookup error = %v, want session.ErrStateKeyNotExist — temp: state must not survive past its own invocation", tempPercentageKey, err)
	}
	if _, err := state.Get(tempRawScoreKey); !errors.Is(err, session.ErrStateKeyNotExist) {
		t.Errorf("state[%q] lookup error = %v, want session.ErrStateKeyNotExist — temp: state must not survive past its own invocation", tempRawScoreKey, err)
	}
}
