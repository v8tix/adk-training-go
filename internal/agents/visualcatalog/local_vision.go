// This file (bonus, outside this course's ADK-SDK lesson): DescribeImageLocally
// proves the local model server itself handles an image correctly, by calling
// it directly over HTTP instead of through llmagent/runner. The package's
// real lesson stays BuildRootAgent + Gemini through the actual ADK
// agent/runner framework (see agent.go's own package doc comment) — this
// file exists only to back up the claim that the gap found in module-7 is in
// this SDK's local-model client (model/openaimodel), not in Ollama or the
// model itself. Uses kawa (github.com/v8tix/kawa) as the HTTP call layer, the
// same library and pattern used in the companion pdf2md/vision-describe tool
// this finding came from.

package visualcatalog

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/v8tix/adk-training-go/internal/infrastructure/prompts"
	"github.com/v8tix/kawa"
)

// localVisionDeadline bounds a single call — local inference on a large
// image can take a while, especially for a thinking-capable model. Set on
// both the underlying http.Client (via kawa.NewHTTPClient) and the call's own
// WithDeadline: the two are redundant for this single-attempt-per-client
// usage today, kept as deliberate defense-in-depth against a hung RoundTrip
// rather than a single point of failure for the timeout.
const localVisionDeadline = 2 * time.Minute

// localVisionMaxRetries covers transient failures without wasting attempts
// on a permanent error — kawa's default HTTPStatusPolicy already treats a
// 4xx/most-5xx as permanent and stops immediately.
const localVisionMaxRetries = 2

var (
	// ErrUnsupportedImageType indicates imagePath's extension isn't one this
	// package knows how to map to a MIME type.
	ErrUnsupportedImageType = errors.New("unsupported image type")
	// ErrReadingImage indicates imagePath couldn't be read from disk.
	ErrReadingImage = errors.New("reading image")
	// ErrCallingOllama indicates the HTTP call to Ollama's chat/completions
	// endpoint itself failed (transport error or non-2xx status).
	ErrCallingOllama = errors.New("calling ollama")
	// ErrEmptyOllamaResponse indicates Ollama returned a well-formed response
	// with no usable description in it.
	ErrEmptyOllamaResponse = errors.New("empty ollama response")
)

func localImageMIMEType(path string) (string, error) {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".jpg", ".jpeg":
		return "image/jpeg", nil
	case ".png":
		return "image/png", nil
	case ".webp":
		return "image/webp", nil
	case ".gif":
		return "image/gif", nil
	default:
		return "", fmt.Errorf("%w: %q", ErrUnsupportedImageType, filepath.Ext(path))
	}
}

// ollamaChatRequest/ollamaChatResponse mirror the OpenAI-compatible
// chat/completions shape Ollama accepts. ollamaChatResponse must declare
// every top-level field Ollama's response actually contains — kawa decodes
// via jsonx.ReadJSONAs internally, which rejects unknown fields with no way
// to opt out through kawa's own public API (a real difference from stdlib's
// encoding/json.Unmarshal, which silently ignores fields a struct doesn't
// declare — confirmed while building the companion pdf2md tool).
type ollamaChatRequest struct {
	Model    string              `json:"model"`
	Messages []ollamaChatMessage `json:"messages"`
}

func (ollamaChatRequest) Req() {}

type ollamaChatMessage struct {
	Role    string                  `json:"role"`
	Content []ollamaChatContentPart `json:"content"`
}

type ollamaChatContentPart struct {
	Type     string              `json:"type"`
	Text     string              `json:"text,omitempty"`
	ImageURL *ollamaChatImageURL `json:"image_url,omitempty"`
}

type ollamaChatImageURL struct {
	URL string `json:"url"`
}

type ollamaChatChoice struct {
	Index   int `json:"index"`
	Message struct {
		Role string `json:"role"`
		// Content is the model's final answer. Thinking-capable models
		// return their reasoning separately in their own "reasoning"
		// field, so Content alone is already the answer.
		Content   string `json:"content"`
		Reasoning string `json:"reasoning,omitempty"`
	} `json:"message"`
	FinishReason string `json:"finish_reason"`
}

type ollamaChatResponse struct {
	ID                string             `json:"id"`
	Object            string             `json:"object"`
	Created           int64              `json:"created"`
	Model             string             `json:"model"`
	SystemFingerprint string             `json:"system_fingerprint"`
	Choices           []ollamaChatChoice `json:"choices"`
	Usage             ollamaChatUsage    `json:"usage"`
}

type ollamaChatUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

func (ollamaChatResponse) Res() {}

// DescribeImageLocally sends imagePath directly to the Ollama server at
// baseURL (its OpenAI-compatible /chat/completions endpoint) using model,
// bypassing llmagent/runner entirely. It exists to prove the local model
// server can handle the same image this package's real, ADK-based
// BuildRootAgent path cannot send — not as a replacement for it.
//
// baseURL and model are trusted, developer-supplied configuration (they come
// from internal/infrastructure/llm.Config, loaded from the caller's own
// .env), not untrusted external input — this isn't meant to be pointed at an
// arbitrary caller-supplied endpoint.
func DescribeImageLocally(ctx context.Context, baseURL, model, imagePath string) (string, error) {
	mt, err := localImageMIMEType(imagePath)
	if err != nil {
		return "", err
	}

	instruction, err := prompts.Get(PromptNamespace + "/local_vision_instruction")
	if err != nil {
		return "", err
	}

	imageBytes, err := os.ReadFile(imagePath)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrReadingImage, err)
	}
	dataURL := fmt.Sprintf("data:%s;base64,%s", mt, base64.StdEncoding.EncodeToString(imageBytes))

	req := ollamaChatRequest{
		Model: model,
		Messages: []ollamaChatMessage{
			{
				Role: "user",
				Content: []ollamaChatContentPart{
					{Type: "text", Text: instruction},
					{Type: "image_url", ImageURL: &ollamaChatImageURL{URL: dataURL}},
				},
			},
		},
	}

	httpClient := kawa.NewHTTPClient(localVisionDeadline, nil, nil)
	call := kawa.NewCall[ollamaChatRequest, ollamaChatResponse](httpClient, kawa.Post, baseURL+"/chat/completions").
		WithDeadline(localVisionDeadline).
		WithMaxRetries(localVisionMaxRetries)

	env, err := call.DoWithRetry(ctx, &req)
	if err != nil {
		if httpErr, ok := errors.AsType[kawa.ErrInvalidHTTPStatus](err); ok {
			return "", fmt.Errorf("%w: status %d: %s", ErrCallingOllama, httpErr.StatusCode, httpErr.Body)
		}
		return "", fmt.Errorf("%w: %v", ErrCallingOllama, err)
	}
	if len(env.Body.Choices) == 0 {
		return "", fmt.Errorf("%w: no choices in response", ErrEmptyOllamaResponse)
	}

	text := strings.TrimSpace(env.Body.Choices[0].Message.Content)
	if text == "" {
		return "", fmt.Errorf("%w: model returned no text", ErrEmptyOllamaResponse)
	}
	return text, nil
}
