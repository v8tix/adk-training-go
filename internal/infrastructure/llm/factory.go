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
	ModelTypeOllama   = "ollama"
	ModelTypeGemini   = "gemini"
	ModelTypeVertexAI = "vertexai"
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
	ModelTypeOllama:   newOllamaModel,
	ModelTypeGemini:   newGeminiModel,
	ModelTypeVertexAI: newVertexAIModel,
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
	m, err := openaimodel.NewModel(ctx, cfg.OllamaModel, ollamaClientConfig(cfg))
	return m, cfg.OllamaModel, err
}

// ollamaClientConfig builds the client configuration for the Ollama backend,
// including the production retry policy — split out from newOllamaModel so
// it's testable without a live network call, mirroring geminiClientConfig.
// Ollama ignores the API key value but openaimodel requires a non-empty
// string.
func ollamaClientConfig(cfg Config) *openaimodel.ClientConfig {
	return &openaimodel.ClientConfig{
		APIKey:  "ollama",
		BaseURL: cfg.OllamaBaseURL,
		Options: productionOllamaRetryOptions(),
	}
}

// newGeminiModel builds the cloud path, for checking Google Cloud/AI Studio
// credentials specifically (MODEL_TYPE=gemini).
func newGeminiModel(ctx context.Context, cfg Config) (model.LLM, string, error) {
	m, err := gemini.NewModel(ctx, cfg.GeminiModel, geminiClientConfig(cfg))
	return m, cfg.GeminiModel, err
}

// geminiClientConfig builds the client configuration for the Gemini backend,
// including the production retry policy — split out from newGeminiModel so
// it's testable without a live network call.
func geminiClientConfig(cfg Config) *genai.ClientConfig {
	return &genai.ClientConfig{
		APIKey: cfg.GoogleAPIKey,
		HTTPOptions: genai.HTTPOptions{
			RetryOptions: productionRetryOptions(),
		},
	}
}

// newVertexAIModel builds the Vertex AI backend, required for module-12's
// google_maps_grounding built-in tool, which the public Gemini API doesn't
// offer. Two real auth paths, tried in order — see vertexAIClientConfig.
func newVertexAIModel(ctx context.Context, cfg Config) (model.LLM, string, error) {
	m, err := gemini.NewModel(ctx, cfg.VertexAIModel, vertexAIClientConfig(cfg))
	return m, cfg.VertexAIModel, err
}

// vertexAIClientConfig builds the client configuration for the Vertex AI
// backend, including the production retry policy — split out from
// newVertexAIModel so it's testable without a live network call, mirroring
// geminiClientConfig.
//
// Two real, mutually exclusive auth paths: VertexAIAPIKey (Express Mode —
// API-key-only, confirmed live to work ONLY for a key generated through
// Vertex AI Studio's own Express enrollment flow, not a plain Google Cloud
// API key) takes precedence when set; otherwise VertexAIProject +
// VertexAILocation are passed through with no explicit Credentials, letting
// genai.NewClient resolve Application Default Credentials itself (gcloud
// auth application-default login, or a service account) — confirmed live
// against a real GCP project with ADC set up via gcloud, the path most
// existing Google Cloud accounts actually need.
func vertexAIClientConfig(cfg Config) *genai.ClientConfig {
	cc := &genai.ClientConfig{
		Backend: genai.BackendVertexAI,
		HTTPOptions: genai.HTTPOptions{
			RetryOptions: productionRetryOptions(),
		},
	}
	if cfg.VertexAIAPIKey != "" {
		cc.APIKey = cfg.VertexAIAPIKey
		return cc
	}
	cc.Project = cfg.VertexAIProject
	cc.Location = cfg.VertexAILocation
	return cc
}
