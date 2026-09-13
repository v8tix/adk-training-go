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
}

// LoadConfig reads Config from the environment, falling back to this
// course's local-first defaults for anything unset.
func LoadConfig() Config {
	return Config{
		ModelType:     getEnv("MODEL_TYPE", ModelTypeOllama),
		OllamaBaseURL: getEnv("OLLAMA_BASE_URL", "http://localhost:11434/v1"),
		OllamaModel:   getEnv("OLLAMA_MODEL", "qwen38-standard"),
		GeminiModel:   getEnv("GEMINI_MODEL", "gemini-3.5-flash"),
		GoogleAPIKey:  getEnv("GOOGLE_API_KEY", ""),
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
