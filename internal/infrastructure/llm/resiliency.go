package llm

import (
	"time"

	"github.com/openai/openai-go/v3/option"
	"google.golang.org/genai"
)

// retryableStatusCodes are the HTTP statuses worth retrying: request
// timeouts and rate limiting are transient by nature, and a 5xx means the
// server itself failed, not the request. Every other 4xx (400, 401, 403,
// 404, ...) means the request itself is wrong — retrying it only wastes the
// retry budget without any chance of succeeding. Only genai.HTTPRetryOptions
// (the Gemini backend) exposes this list as configurable; openai-go (the
// Ollama backend) hardcodes its own, close but not identical, default (408,
// 409, 429, and every 5xx).
var retryableStatusCodes = []int32{408, 429, 500, 502, 503, 504}

// productionRetryAttempts and productionMaxRetryDelay are shared between
// both backends' retry policies so "5 attempts, capped at 10s between
// retries" means the same thing regardless of which one is configured —
// even though the two SDKs express it differently (see productionRetryOptions
// and productionOllamaRetryOptions).
const (
	productionRetryAttempts  = 5
	productionMaxRetryDelay  = 10 * time.Second
	productionExpBackoffBase = 2.0
	productionJitter         = 0.5
)

// productionRetryOptions is this project's retry policy for the Gemini
// backend: exponential backoff with jitter (to avoid many failed requests
// retrying in lockstep), scoped to the status codes above.
func productionRetryOptions() *genai.HTTPRetryOptions {
	maxDelaySeconds := productionMaxRetryDelay.Seconds()
	expBase := productionExpBackoffBase
	jitter := productionJitter
	attempts := int32(productionRetryAttempts)
	return &genai.HTTPRetryOptions{
		Attempts:        &attempts,
		MaxDelay:        &maxDelaySeconds,
		ExpBase:         &expBase,
		Jitter:          &jitter,
		HTTPStatusCodes: retryableStatusCodes,
	}
}

// productionOllamaRetryOptions is this project's retry policy for the Ollama
// backend, expressed in openai-go's own (less configurable) terms:
// productionRetryAttempts total attempts (openai-go's WithMaxRetries counts
// retries *after* the original request, hence attempts-1), capped at
// productionMaxRetryDelay between retries. openai-go has no equivalent of
// ExpBase/Jitter/HTTPStatusCodes — it hardcodes its own base-2 exponential
// backoff with up-to-25%-of-delay jitter, and its own retryable-status set
// (408, 409, 429, and every 5xx) — close in spirit to the Gemini policy
// above, not numerically identical, because openai-go doesn't expose those
// knobs.
func productionOllamaRetryOptions() []option.RequestOption {
	return []option.RequestOption{
		option.WithMaxRetries(productionRetryAttempts - 1),
		option.WithMaxRetryDelay(productionMaxRetryDelay),
	}
}
