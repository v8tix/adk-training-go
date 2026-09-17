package personaltutor

import (
	"errors"
	"fmt"
	"strings"

	"google.golang.org/adk/v2/agent"
	"google.golang.org/adk/v2/session"
)

// State keys. user*/currentTopicKey follow session.KeyPrefixUser and no
// prefix respectively; the temp* keys follow session.KeyPrefixTemp.
const (
	userLanguageKey   = session.KeyPrefixUser + "language"
	userDifficultyKey = session.KeyPrefixUser + "difficulty_level"
	userTopicsKey     = session.KeyPrefixUser + "topics"
	userScoresKey     = session.KeyPrefixUser + "scores"
	currentTopicKey   = "current_topic"
	tempPercentageKey = session.KeyPrefixTemp + "percentage"
	tempRawScoreKey   = session.KeyPrefixTemp + "raw_score"
)

// ===== set_user_preferences =====

// SetUserPreferencesArgs holds the user's learning preferences.
type SetUserPreferencesArgs struct {
	Language        string `json:"language" jsonschema:"preferred language (en, es, fr, etc.)"`
	DifficultyLevel string `json:"difficulty_level" jsonschema:"beginner, intermediate, or advanced"`
}

// SetUserPreferencesResult is set_user_preferences's result.
type SetUserPreferencesResult struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

// setUserPreferences stores the user's preferences under the user: prefix —
// persistent across every session for this user, not just this one.
func setUserPreferences(ctx agent.Context, args SetUserPreferencesArgs) (SetUserPreferencesResult, error) {
	if err := ctx.State().Set(userLanguageKey, args.Language); err != nil {
		return SetUserPreferencesResult{}, err
	}
	if err := ctx.State().Set(userDifficultyKey, args.DifficultyLevel); err != nil {
		return SetUserPreferencesResult{}, err
	}
	return SetUserPreferencesResult{
		Status:  "success",
		Message: fmt.Sprintf("Preferences saved: %s, %s level", args.Language, args.DifficultyLevel),
	}, nil
}

// ===== record_topic_completion =====

// RecordTopicCompletionArgs holds a completed topic and its quiz score.
type RecordTopicCompletionArgs struct {
	Topic     string `json:"topic" jsonschema:"topic name, e.g. Python Basics"`
	QuizScore int    `json:"quiz_score" jsonschema:"score out of 100"`
}

// RecordTopicCompletionResult is record_topic_completion's result.
type RecordTopicCompletionResult struct {
	Status      string `json:"status"`
	TopicsCount int    `json:"topics_count"`
	Message     string `json:"message"`
}

// recordTopicCompletion appends to the user's persistent topic list and
// score map — read-then-write, since session.State has no native append.
// Recording the same topic twice appends a second list entry while the
// score map merely overwrites that topic's score, so TopicsCount can
// exceed the number of distinct topics in AllScores. A known
// simplification matching Python's own lab scope, not a bug to fix here.
func recordTopicCompletion(ctx agent.Context, args RecordTopicCompletionArgs) (RecordTopicCompletionResult, error) {
	topics, err := getUserTopics(ctx)
	if err != nil {
		return RecordTopicCompletionResult{}, err
	}
	scores, err := getUserScores(ctx)
	if err != nil {
		return RecordTopicCompletionResult{}, err
	}

	topics = append(topics, args.Topic)
	scores[args.Topic] = args.QuizScore

	if err := ctx.State().Set(userTopicsKey, topics); err != nil {
		return RecordTopicCompletionResult{}, err
	}
	if err := ctx.State().Set(userScoresKey, scores); err != nil {
		return RecordTopicCompletionResult{}, err
	}

	return RecordTopicCompletionResult{
		Status:      "success",
		TopicsCount: len(topics),
		Message:     fmt.Sprintf("Recorded: %s with score %d/100", args.Topic, args.QuizScore),
	}, nil
}

// ===== get_user_progress =====

// GetUserProgressArgs is empty — get_user_progress takes no parameters.
type GetUserProgressArgs struct{}

// GetUserProgressResult is get_user_progress's result.
type GetUserProgressResult struct {
	Status           string         `json:"status"`
	Language         string         `json:"language"`
	DifficultyLevel  string         `json:"difficulty_level"`
	TopicsCompleted  int            `json:"topics_completed"`
	Topics           []string       `json:"topics"`
	AverageQuizScore float64        `json:"average_quiz_score"`
	AllScores        map[string]int `json:"all_scores"`
}

// getUserProgress reads back everything set_user_preferences and
// record_topic_completion have stored, defaulting a first-time user's
// missing keys rather than erroring.
func getUserProgress(ctx agent.Context, _ GetUserProgressArgs) (GetUserProgressResult, error) {
	language, err := getStateStringOrDefault(ctx, userLanguageKey, "en")
	if err != nil {
		return GetUserProgressResult{}, err
	}
	difficulty, err := getStateStringOrDefault(ctx, userDifficultyKey, "beginner")
	if err != nil {
		return GetUserProgressResult{}, err
	}
	topics, err := getUserTopics(ctx)
	if err != nil {
		return GetUserProgressResult{}, err
	}
	scores, err := getUserScores(ctx)
	if err != nil {
		return GetUserProgressResult{}, err
	}

	var avg float64
	if len(scores) > 0 {
		var sum int
		for _, s := range scores {
			sum += s
		}
		avg = float64(sum) / float64(len(scores))
	}

	return GetUserProgressResult{
		Status:           "success",
		Language:         language,
		DifficultyLevel:  difficulty,
		TopicsCompleted:  len(topics),
		Topics:           topics,
		AverageQuizScore: avg,
		AllScores:        scores,
	}, nil
}

// ===== start_learning_session =====

// StartLearningSessionArgs holds the topic to start.
type StartLearningSessionArgs struct {
	Topic string `json:"topic" jsonschema:"the topic to start learning"`
}

// StartLearningSessionResult is start_learning_session's result.
type StartLearningSessionResult struct {
	Status          string `json:"status"`
	Topic           string `json:"topic"`
	DifficultyLevel string `json:"difficulty_level"`
	Message         string `json:"message"`
}

// startLearningSession writes the current topic to plain, unprefixed
// session state — scoped to this conversation only, unlike the user:
// preferences it reads for personalization.
func startLearningSession(ctx agent.Context, args StartLearningSessionArgs) (StartLearningSessionResult, error) {
	if err := ctx.State().Set(currentTopicKey, args.Topic); err != nil {
		return StartLearningSessionResult{}, err
	}
	difficulty, err := getStateStringOrDefault(ctx, userDifficultyKey, "beginner")
	if err != nil {
		return StartLearningSessionResult{}, err
	}

	return StartLearningSessionResult{
		Status:          "success",
		Topic:           args.Topic,
		DifficultyLevel: difficulty,
		Message:         fmt.Sprintf("Started learning session: %s at %s level", args.Topic, difficulty),
	}, nil
}

// ===== calculate_quiz_grade =====

// CalculateQuizGradeArgs holds a quiz's raw results.
type CalculateQuizGradeArgs struct {
	CorrectAnswers int `json:"correct_answers"`
	TotalQuestions int `json:"total_questions"`
}

// CalculateQuizGradeResult is calculate_quiz_grade's result.
type CalculateQuizGradeResult struct {
	Status     string  `json:"status"`
	Score      string  `json:"score"`
	Percentage float64 `json:"percentage"`
	Grade      string  `json:"grade"`
	Message    string  `json:"message"`
}

// calculateQuizGrade stores the intermediate percentage/raw score under
// temp: — scoped to this single invocation only, never persisted, unlike
// every user:-prefixed key above.
func calculateQuizGrade(ctx agent.Context, args CalculateQuizGradeArgs) (CalculateQuizGradeResult, error) {
	// Clamp a nonsensical correct-answers count (e.g. the model hallucinating
	// correct > total) so the reported percentage never exceeds 100.
	correct := min(args.CorrectAnswers, args.TotalQuestions)

	var percentage float64
	if args.TotalQuestions > 0 {
		percentage = float64(correct) / float64(args.TotalQuestions) * 100
	}

	if err := ctx.State().Set(tempPercentageKey, percentage); err != nil {
		return CalculateQuizGradeResult{}, err
	}
	if err := ctx.State().Set(tempRawScoreKey, args.CorrectAnswers); err != nil {
		return CalculateQuizGradeResult{}, err
	}

	grade := letterGrade(percentage)

	return CalculateQuizGradeResult{
		Status:     "success",
		Score:      fmt.Sprintf("%d/%d", args.CorrectAnswers, args.TotalQuestions),
		Percentage: percentage,
		Grade:      grade,
		Message:    fmt.Sprintf("Quiz grade: %s (%.1f%%)", grade, percentage),
	}, nil
}

func letterGrade(percentage float64) string {
	switch {
	case percentage >= 90:
		return "A"
	case percentage >= 80:
		return "B"
	case percentage >= 70:
		return "C"
	case percentage >= 60:
		return "D"
	default:
		return "F"
	}
}

// ===== search_past_lessons =====

// SearchPastLessonsArgs holds the search query.
type SearchPastLessonsArgs struct {
	Query string `json:"query" jsonschema:"topic or keyword to search for"`
}

// SearchPastLessonsResult is search_past_lessons's result.
type SearchPastLessonsResult struct {
	Status  string `json:"status"`
	Found   bool   `json:"found"`
	Message string `json:"message"`
}

// searchPastLessons simulates a memory search over the user's own recorded
// topics — a plain substring match, not a real call to memory.Service.
// Matching Python's own lab scope: in production this would call
// memory.Service's SearchMemory instead, proven to genuinely work in
// memory_test.go, but not wired into this tool itself.
func searchPastLessons(ctx agent.Context, args SearchPastLessonsArgs) (SearchPastLessonsResult, error) {
	topics, err := getUserTopics(ctx)
	if err != nil {
		return SearchPastLessonsResult{}, err
	}

	for _, topic := range topics {
		if strings.Contains(strings.ToLower(topic), strings.ToLower(args.Query)) {
			return SearchPastLessonsResult{
				Status:  "success",
				Found:   true,
				Message: fmt.Sprintf("Found a past session on %q related to %q.", topic, args.Query),
			}, nil
		}
	}

	return SearchPastLessonsResult{
		Status:  "success",
		Found:   false,
		Message: fmt.Sprintf("No past sessions found for %q", args.Query),
	}, nil
}

// ===== shared helpers =====

// getStateStringOrDefault's writers (setUserPreferences, for both keys it
// calls this for) always write a string, so a failed assertion here can't
// currently happen; the ignored ok just falls back to "" rather than
// panicking if that ever stops being true.
func getStateStringOrDefault(ctx agent.Context, key, def string) (string, error) {
	val, err := ctx.State().Get(key)
	if errors.Is(err, session.ErrStateKeyNotExist) {
		return def, nil
	}
	if err != nil {
		return "", err
	}
	s, _ := val.(string)
	return s, nil
}

// getUserTopics's only writer, recordTopicCompletion, always writes
// []string, so a failed assertion here can't currently happen; the ignored
// ok just falls back to nil rather than panicking if that ever stops being
// true.
func getUserTopics(ctx agent.Context) ([]string, error) {
	val, err := ctx.State().Get(userTopicsKey)
	if errors.Is(err, session.ErrStateKeyNotExist) {
		return []string{}, nil
	}
	if err != nil {
		return nil, err
	}
	topics, _ := val.([]string)
	return topics, nil
}

// getUserScores's only writer, recordTopicCompletion, always writes a
// non-nil map[string]int, so a failed assertion here can't currently
// happen; the nil guard below only matters if that ever stops being true.
func getUserScores(ctx agent.Context) (map[string]int, error) {
	val, err := ctx.State().Get(userScoresKey)
	if errors.Is(err, session.ErrStateKeyNotExist) {
		return map[string]int{}, nil
	}
	if err != nil {
		return nil, err
	}
	scores, _ := val.(map[string]int)
	if scores == nil {
		scores = map[string]int{}
	}
	return scores, nil
}
