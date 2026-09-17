// Package personaltutor defines the Personal Learning Tutor agent: it
// exercises all four session-state scopes through six custom function
// tools — user:-prefixed preferences and progress (persistent across
// sessions for this user), a plain, unprefixed current_topic (scoped to
// this session only), and temp:-prefixed quiz-grading intermediates
// (never persisted past the tool call that wrote them). Reuses module-10's
// (internal/agents/memory) exact function-tool + agent.Context.State()
// shape, just with more prefixes exercised.
//
// The instruction demonstrates {key?} optional templating with
// {app:course_version?} — confirmed live in
// internal/llminternal/instruction_processor.go: a trailing "?" on the
// variable name skips the usual error when the key is missing, instead of
// requiring every deployment to pre-seed an app: key that may not exist.
package personaltutor

import (
	"embed"
	"io/fs"
	"log"

	"github.com/v8tix/adk-training-go/internal/infrastructure/prompts"
	"google.golang.org/adk/v2/agent"
	"google.golang.org/adk/v2/agent/llmagent"
	"google.golang.org/adk/v2/model"
	"google.golang.org/adk/v2/tool"
	"google.golang.org/adk/v2/tool/functiontool"
)

// PromptNamespace identifies this package's entries in the shared prompts
// cache (internal/infrastructure/prompts).
const PromptNamespace = "personaltutor"

//go:embed prompts/*.md
var promptFS embed.FS

func init() {
	promptFiles, err := fs.Sub(promptFS, "prompts")
	if err != nil {
		log.Fatalf("resolving prompts directory: %v", err)
	}
	if err := prompts.Register(PromptNamespace, promptFiles, ".md"); err != nil {
		log.Fatalf("registering prompts: %v", err)
	}
}

// BuildRootAgent constructs the Personal Learning Tutor agent around
// llmModel, wrapping all six state-backed tools.
func BuildRootAgent(llmModel model.LLM) (agent.Agent, error) {
	instruction, err := prompts.Get(PromptNamespace + "/tutor_instruction")
	if err != nil {
		return nil, err
	}

	setPreferencesTool, err := functiontool.New(functiontool.Config{
		Name:        "set_user_preferences",
		Description: "Saves the user's preferred language and difficulty level. Use this when the user states either preference.",
	}, setUserPreferences)
	if err != nil {
		return nil, err
	}
	recordCompletionTool, err := functiontool.New(functiontool.Config{
		Name:        "record_topic_completion",
		Description: "Records a completed topic and its quiz score. Use this once a topic's score is known.",
	}, recordTopicCompletion)
	if err != nil {
		return nil, err
	}
	getProgressTool, err := functiontool.New(functiontool.Config{
		Name:        "get_user_progress",
		Description: "Returns the user's overall learning progress: topics completed and average quiz score.",
	}, getUserProgress)
	if err != nil {
		return nil, err
	}
	startSessionTool, err := functiontool.New(functiontool.Config{
		Name:        "start_learning_session",
		Description: "Starts a learning session on a topic, personalized to the user's difficulty level.",
	}, startLearningSession)
	if err != nil {
		return nil, err
	}
	calculateGradeTool, err := functiontool.New(functiontool.Config{
		Name:        "calculate_quiz_grade",
		Description: "Computes a letter grade and percentage from a quiz's correct/total answer counts.",
	}, calculateQuizGrade)
	if err != nil {
		return nil, err
	}
	searchLessonsTool, err := functiontool.New(functiontool.Config{
		Name:        "search_past_lessons",
		Description: "Searches the user's past completed topics for one matching a query.",
	}, searchPastLessons)
	if err != nil {
		return nil, err
	}

	return llmagent.New(llmagent.Config{
		Name:        "personal_tutor_agent",
		Model:       llmModel,
		Description: "A personalized learning tutor that tracks preferences, progress, and quiz results across sessions.",
		Instruction: instruction,
		Tools: []tool.Tool{
			setPreferencesTool,
			recordCompletionTool,
			getProgressTool,
			startSessionTool,
			calculateGradeTool,
			searchLessonsTool,
		},
	})
}
