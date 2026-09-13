package llm

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"slices"

	"google.golang.org/adk/v2/model"
	"google.golang.org/adk/v2/model/gemini"
	"google.golang.org/adk/v2/model/openaimodel"
	"google.golang.org/genai"
)

// Model type strings selected via Config.ModelType (env MODEL_TYPE). New
// backends register their own factory in modelFactories instead of adding a
// branch to BuildModel.
const (
	ModelTypeOllama = "ollama"
	ModelTypeGemini = "gemini"
)

var (
	// ErrUnknownModelType indicates Config.ModelType doesn't match any
	// factory registered in modelFactories.
	ErrUnknownModelType = errors.New("unknown model type")
	// ErrBuildingModel indicates a registered factory failed to construct
	// its model.LLM.
	ErrBuildingModel = errors.New("building model failed")
)

// modelFactory builds a model.LLM for one model type, and returns a
// human-readable name for it.
type modelFactory func(ctx context.Context, cfg Config) (model.LLM, string, error)

var modelFactories = map[string]modelFactory{
	ModelTypeOllama: newOllamaModel,
	ModelTypeGemini: newGeminiModel,
}

// BuildModel dispatches to the factory registered for cfg.ModelType.
func BuildModel(ctx context.Context, cfg Config) (model.LLM, string, error) {
	factory, ok := modelFactories[cfg.ModelType]
	if !ok {
		return nil, "", fmt.Errorf("%w: %q (want one of: %v)", ErrUnknownModelType, cfg.ModelType, knownModelTypes())
	}

	m, name, err := factory(ctx, cfg)
	if err != nil {
		return nil, "", fmt.Errorf("%w: %s: %v", ErrBuildingModel, cfg.ModelType, err)
	}
	return m, name, nil
}

func knownModelTypes() []string {
	return slices.Sorted(maps.Keys(modelFactories))
}

// newOllamaModel builds the local-first default backend, per this project's
// local-first constraint.
func newOllamaModel(ctx context.Context, cfg Config) (model.LLM, string, error) {
	m, err := openaimodel.NewModel(ctx, cfg.OllamaModel, &openaimodel.ClientConfig{
		APIKey:  "ollama",
		BaseURL: cfg.OllamaBaseURL,
	})
	return m, cfg.OllamaModel, err
}

// newGeminiModel builds the cloud path, for checking Google Cloud/AI Studio
// credentials specifically (MODEL_TYPE=gemini).
func newGeminiModel(ctx context.Context, cfg Config) (model.LLM, string, error) {
	m, err := gemini.NewModel(ctx, cfg.GeminiModel, &genai.ClientConfig{
		APIKey: cfg.GoogleAPIKey,
	})
	return m, cfg.GeminiModel, err
}
