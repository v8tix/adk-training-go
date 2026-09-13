package llm

import (
	"slices"
	"testing"
)

func TestRetryableStatusCodes(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int32
		wantRetry  bool
	}{
		{name: "408 request timeout is retryable", statusCode: 408, wantRetry: true},
		{name: "429 too many requests is retryable", statusCode: 429, wantRetry: true},
		{name: "500 internal server error is retryable", statusCode: 500, wantRetry: true},
		{name: "502 bad gateway is retryable", statusCode: 502, wantRetry: true},
		{name: "503 service unavailable is retryable", statusCode: 503, wantRetry: true},
		{name: "504 gateway timeout is retryable", statusCode: 504, wantRetry: true},
		{name: "400 bad request is not retryable", statusCode: 400, wantRetry: false},
		{name: "401 unauthorized is not retryable", statusCode: 401, wantRetry: false},
		{name: "403 forbidden is not retryable", statusCode: 403, wantRetry: false},
		{name: "404 not found is not retryable", statusCode: 404, wantRetry: false},
		{name: "409 conflict is not retryable", statusCode: 409, wantRetry: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := slices.Contains(retryableStatusCodes, tt.statusCode)
			if got != tt.wantRetry {
				t.Fatalf("retryableStatusCodes contains %d = %v, want %v", tt.statusCode, got, tt.wantRetry)
			}
		})
	}
}

func TestProductionRetryOptions(t *testing.T) {
	got := productionRetryOptions()

	if got.Attempts == nil || *got.Attempts != 5 {
		t.Errorf("Attempts = %v, want 5", got.Attempts)
	}
	if got.MaxDelay == nil || *got.MaxDelay != 10.0 {
		t.Errorf("MaxDelay = %v, want 10.0", got.MaxDelay)
	}
	if got.ExpBase == nil || *got.ExpBase != 2.0 {
		t.Errorf("ExpBase = %v, want 2.0", got.ExpBase)
	}
	if got.Jitter == nil || *got.Jitter != 0.5 {
		t.Errorf("Jitter = %v, want 0.5", got.Jitter)
	}
	if !slices.Equal(got.HTTPStatusCodes, retryableStatusCodes) {
		t.Errorf("HTTPStatusCodes = %v, want %v", got.HTTPStatusCodes, retryableStatusCodes)
	}
}

// TestProductionOllamaRetryOptions can only assert the count of returned
// options, not what each one sets internally: option.RequestOption is an
// opaque closure over openai-go's RequestConfig, which lives in an
// internal/ package this module can't import. This still catches the two
// real regressions worth catching — an accidentally dropped option, or an
// accidentally duplicated one.
func TestProductionOllamaRetryOptions(t *testing.T) {
	got := productionOllamaRetryOptions()

	const wantOptions = 2 // WithMaxRetries + WithMaxRetryDelay
	if len(got) != wantOptions {
		t.Fatalf("len(productionOllamaRetryOptions()) = %d, want %d", len(got), wantOptions)
	}
	for i, opt := range got {
		if opt == nil {
			t.Errorf("productionOllamaRetryOptions()[%d] is nil", i)
		}
	}
}
