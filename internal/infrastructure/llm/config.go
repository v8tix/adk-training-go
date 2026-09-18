// Package llm builds a model.LLM from environment-driven config, shared by
// every cmd/ program that needs to talk to a language model.
package llm

import "os"

// Config holds runtime settings for selecting and building a model.LLM,
// sourced from environment variables (loaded from .env via godotenv.Load()
// in the caller's main, if present). Defaults match this course's
// local-first setup — no .env is required to run against the local Ollama
// server.
type Config struct {
	// ModelType selects which registered factory in modelFactories builds the
	// model — a type string (e.g. "ollama", "gemini"), not a boolean/numeric
	// flag, so adding a third backend later doesn't need a new config field.
	ModelType     string
	OllamaBaseURL string
	OllamaModel   string
	GeminiModel   string
	GoogleAPIKey  string
	// VertexAIAPIKey enables Vertex AI's Express Mode (API-key auth, no GCP
	// project/location needed) — required for genai.BackendVertexAI-only
	// capabilities like the google_maps_grounding built-in tool, which the
	// public Gemini API (GoogleAPIKey above) doesn't offer. See
	// docs/module-12/README.md.
	//
	// Confirmed live: a plain Google Cloud API key (the classic "AIzaSy..."
	// format) is NOT sufficient here — the aiplatform.googleapis.com
	// PredictionService rejects it with 401 CREDENTIALS_MISSING unless that
	// specific key was generated through Vertex AI Studio's own Express Mode
	// enrollment flow (console.cloud.google.com/vertex-ai/generative/express),
	// which most existing Google Cloud accounts can't access. When
	// VertexAIAPIKey doesn't work or isn't set, use VertexAIProject +
	// VertexAILocation instead (Application Default Credentials) below.
	VertexAIAPIKey string
	// VertexAIProject and VertexAILocation select Vertex AI's other real auth
	// path: Application Default Credentials (gcloud auth application-default
	// login, or a service account), the same mechanism every other Google
	// Cloud SDK uses. genai.ClientConfig leaves Credentials nil and resolves
	// ADC itself when both are set and VertexAIAPIKey is empty. This is the
	// path most Google Cloud accounts actually need, unlike the Express-Mode
	// API key above.
	VertexAIProject  string
	VertexAILocation string
	// VertexAIModel is separate from GeminiModel because Vertex AI's publisher
	// model catalog uses different, often-lagging model identifiers than the
	// public Gemini API — confirmed live: "gemini-3.5-flash" (GeminiModel's
	// own default) 404s on Vertex ("Publisher model ... was not found"),
	// while "gemini-2.5-flash" works.
	VertexAIModel string
	// VertexAILiveModel is a separate model identifier from VertexAIModel
	// because the Gemini Live API (bidirectional audio streaming, module-30)
	// draws from a genuinely distinct model catalog than regular
	// request/response Vertex models — confirmed live: VertexAIModel's own
	// default ("gemini-2.5-flash") doesn't support the Live API's
	// client.Live.Connect at all, while "gemini-live-2.5-flash-native-audio"
	// does, with no Go-vs-Python name lag this time (unlike VertexAIModel's
	// own gemini-3.5-flash/gemini-2.5-flash mismatch).
	VertexAILiveModel string
}

// LoadConfig reads Config from the environment, falling back to this
// course's local-first defaults for anything unset.
//
// OllamaModel defaults to a GGUF (Q4_K_M) quantization — confirmed live
// (module-4) that it supports JSON-schema-constrained output
// (llmagent.Config.OutputSchema), unlike some other quantizations of the
// same model family, which return "501 structured output is unavailable"
// for it. Every module shares this one default so none of them need a
// per-module OLLAMA_MODEL override.
func LoadConfig() Config {
	return Config{
		ModelType:         getEnv("MODEL_TYPE", ModelTypeOllama),
		OllamaBaseURL:     getEnv("OLLAMA_BASE_URL", "http://localhost:11434/v1"),
		OllamaModel:       getEnv("OLLAMA_MODEL", "qwen3.8:27b"),
		GeminiModel:       getEnv("GEMINI_MODEL", "gemini-3.5-flash"),
		GoogleAPIKey:      getEnv("GOOGLE_AI_STUDIO_API_KEY", ""),
		VertexAIAPIKey:    getEnv("VERTEX_AI_API_KEY", ""),
		VertexAIProject:   getEnv("VERTEX_AI_PROJECT", ""),
		VertexAILocation:  getEnv("VERTEX_AI_LOCATION", ""),
		VertexAIModel:     getEnv("VERTEX_AI_MODEL", "gemini-2.5-flash"),
		VertexAILiveModel: getEnv("VERTEX_AI_LIVE_MODEL", "gemini-live-2.5-flash-native-audio"),
	}
}

// getEnv returns the named environment variable, or fallback if it's unset
// or empty.
func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
