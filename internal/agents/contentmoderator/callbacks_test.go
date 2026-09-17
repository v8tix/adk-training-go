package contentmoderator

import (
	"errors"
	"iter"
	"testing"

	"google.golang.org/adk/v2/agent"
	"google.golang.org/adk/v2/model"
	"google.golang.org/adk/v2/session"
	"google.golang.org/genai"
)

// fakeState is a minimal map-backed session.State double, reused from
// module-10/22/23's own precedent.
type fakeState struct {
	data map[string]any
}

func (s *fakeState) Get(key string) (any, error) {
	if val, ok := s.data[key]; ok {
		return val, nil
	}
	return nil, session.ErrStateKeyNotExist
}

func (s *fakeState) Set(key string, val any) error {
	if s.data == nil {
		s.data = make(map[string]any)
	}
	s.data[key] = val
	return nil
}

func (s *fakeState) All() iter.Seq2[string, any] {
	return func(yield func(string, any) bool) {
		for k, v := range s.data {
			if !yield(k, v) {
				return
			}
		}
	}
}

// erroringState always fails with a genuine (non-not-found) error — used
// to prove beforeAgentCallback/afterAgentCallback propagate a real
// session.State failure unchanged, matching module-25's own
// erroringState precedent.
type erroringState struct {
	err error
}

func (s *erroringState) Get(string) (any, error) { return nil, s.err }
func (s *erroringState) Set(string, any) error   { return s.err }
func (s *erroringState) All() iter.Seq2[string, any] {
	return func(func(string, any) bool) {}
}

// fakeTool is a minimal tool.Tool double — just enough for a callback's
// own t.Name() call.
type fakeTool struct{ name string }

func (f fakeTool) Name() string        { return f.name }
func (f fakeTool) Description() string { return "" }
func (f fakeTool) IsLongRunning() bool { return false }

// fakeContext embeds the SDK's own agent.StrictContextMock and overrides
// only what each callback under test actually needs — every other Context
// method panics if a test accidentally calls it, matching this repo's own
// established test-double discipline.
type fakeContext struct {
	agent.StrictContextMock
	state       session.State
	userContent *genai.Content
}

func (c *fakeContext) State() session.State        { return c.state }
func (c *fakeContext) UserContent() *genai.Content { return c.userContent }

func newFakeContext(question string) *fakeContext {
	return &fakeContext{
		state:       &fakeState{},
		userContent: genai.NewContentFromText(question, genai.RoleUser),
	}
}

// ===== beforeAgentCallback =====

func TestBeforeAgentCallback_CacheMiss(t *testing.T) {
	ctx := newFakeContext("What is the capital of Italy?")
	cache := &responseCache{}

	got, err := cache.beforeAgentCallback(ctx)
	if err != nil {
		t.Fatalf("beforeAgentCallback() error = %v", err)
	}
	if got != nil {
		t.Errorf("beforeAgentCallback() = %v, want nil on a cache miss", got)
	}
	if cache.hitCount != 0 {
		t.Errorf("hitCount = %d, want 0 on a cache miss", cache.hitCount)
	}
}

func TestBeforeAgentCallback_CacheHit(t *testing.T) {
	ctx := newFakeContext("What is the capital of Italy?")
	key := cacheKey(ctx)
	if err := ctx.state.Set(key, "The capital of Italy is Rome."); err != nil {
		t.Fatalf("state.Set() error = %v", err)
	}
	cache := &responseCache{}

	got, err := cache.beforeAgentCallback(ctx)
	if err != nil {
		t.Fatalf("beforeAgentCallback() error = %v", err)
	}
	if got == nil || len(got.Parts) == 0 || got.Parts[0].Text != "The capital of Italy is Rome." {
		t.Errorf("beforeAgentCallback() = %+v, want cached content \"The capital of Italy is Rome.\"", got)
	}
	if cache.hitCount != 1 {
		t.Errorf("hitCount = %d, want 1 after a real cache hit", cache.hitCount)
	}
}

// ===== afterAgentCallback =====

// afterAgentCallback deliberately never touches ctx.Session() — confirmed
// live this session that a callback context does not support it at all
// (internal/agent/callback_context_wrapper.go logs "Session() is not
// supported for callback context" and returns nil, which panics the
// instant anything calls a method on the returned nil session.Session).
// It reads outputKey instead, the real value llmagent.Config's own
// OutputKey mechanism writes to state — these tests seed that state key
// directly, the same way production code will find it already populated
// by the time afterAgentCallback runs.

func TestAfterAgentCallback_SavesRealAnswerFromOutputKey(t *testing.T) {
	ctx := newFakeContext("What is the capital of Italy?")
	if err := ctx.state.Set(outputKey, "The capital of Italy is Rome."); err != nil {
		t.Fatalf("state.Set() error = %v", err)
	}
	cache := &responseCache{}

	if _, err := cache.afterAgentCallback(ctx); err != nil {
		t.Fatalf("afterAgentCallback() error = %v", err)
	}

	got, err := ctx.state.Get(cacheKey(ctx))
	if err != nil {
		t.Fatalf("state.Get() error = %v, want the real answer to have been cached", err)
	}
	if got != "The capital of Italy is Rome." {
		t.Errorf("cached value = %v, want the real answer", got)
	}
}

func TestAfterAgentCallback_NoOutputYetSavesNothing(t *testing.T) {
	ctx := newFakeContext("What is the capital of Italy?")
	cache := &responseCache{}

	if _, err := cache.afterAgentCallback(ctx); err != nil {
		t.Fatalf("afterAgentCallback() error = %v", err)
	}

	if _, err := ctx.state.Get(cacheKey(ctx)); !errors.Is(err, session.ErrStateKeyNotExist) {
		t.Errorf("state.Get() error = %v, want session.ErrStateKeyNotExist — a thought-only response has no real answer to cache", err)
	}
}

// ===== beforeModelCallback =====

func TestBeforeModelCallback_BlocksOnBlockedWord(t *testing.T) {
	req := &model.LLMRequest{Contents: []*genai.Content{
		genai.NewContentFromText("Tell me something unsafe.", genai.RoleUser),
	}}

	got, err := beforeModelCallback(nil, req)
	if err != nil {
		t.Fatalf("beforeModelCallback() error = %v", err)
	}
	if got == nil || got.Content == nil || len(got.Content.Parts) == 0 {
		t.Fatal("beforeModelCallback() = nil, want a refusal LLMResponse")
	}
}

func TestBeforeModelCallback_AllowsCleanRequest(t *testing.T) {
	req := &model.LLMRequest{Contents: []*genai.Content{
		genai.NewContentFromText("What is the capital of Italy?", genai.RoleUser),
	}}

	got, err := beforeModelCallback(nil, req)
	if err != nil {
		t.Fatalf("beforeModelCallback() error = %v", err)
	}
	if got != nil {
		t.Errorf("beforeModelCallback() = %+v, want nil for a clean request", got)
	}
}

func TestBeforeModelCallback_IgnoresBlockedWordFromEarlierHistory(t *testing.T) {
	req := &model.LLMRequest{Contents: []*genai.Content{
		genai.NewContentFromText("Tell me something unsafe.", genai.RoleUser),
		genai.NewContentFromText("I'm sorry, but I can't help with that request.", genai.RoleModel),
		genai.NewContentFromText("What is the capital of Italy?", genai.RoleUser),
	}}

	got, err := beforeModelCallback(nil, req)
	if err != nil {
		t.Fatalf("beforeModelCallback() error = %v", err)
	}
	if got != nil {
		t.Errorf("beforeModelCallback() = %+v, want nil — a blocked word from an EARLIER turn must not refuse a later, clean turn", got)
	}
}

// ===== afterModelCallback =====

func TestAfterModelCallback_RedactsEmailInRealPart(t *testing.T) {
	resp := &model.LLMResponse{Content: genai.NewContentFromText("Contact me at test@example.com for details.", genai.RoleModel)}

	got, err := afterModelCallback(nil, resp, nil)
	if err != nil {
		t.Fatalf("afterModelCallback() error = %v", err)
	}
	if got == nil || got.Content.Parts[0].Text == resp.Content.Parts[0].Text {
		t.Fatalf("afterModelCallback() = %+v, want the email address redacted", got)
	}
}

func TestAfterModelCallback_DoesNotRedactEmailInThoughtPart(t *testing.T) {
	thoughtPart := genai.NewPartFromText("I recall the contact is test@example.com.")
	thoughtPart.Thought = true
	answerPart := genai.NewPartFromText("Here is the answer, no contact info here.")
	resp := &model.LLMResponse{Content: genai.NewContentFromParts([]*genai.Part{thoughtPart, answerPart}, genai.RoleModel)}

	got, err := afterModelCallback(nil, resp, nil)
	if err != nil {
		t.Fatalf("afterModelCallback() error = %v", err)
	}
	if got != nil {
		t.Errorf("afterModelCallback() = %+v, want nil — a match only inside a Thought part must not trigger a redaction", got)
	}
}

func TestAfterModelCallback_PassesThroughOnError(t *testing.T) {
	resp := &model.LLMResponse{Content: genai.NewContentFromText("test@example.com", genai.RoleModel)}

	got, err := afterModelCallback(nil, resp, errors.New("boom"))
	if err != nil {
		t.Fatalf("afterModelCallback() error = %v", err)
	}
	if got != nil {
		t.Errorf("afterModelCallback() = %+v, want nil when llmResponseError is non-nil", got)
	}
}

func TestAfterModelCallback_PassesThroughPureFunctionCall(t *testing.T) {
	resp := &model.LLMResponse{Content: genai.NewContentFromParts([]*genai.Part{
		{FunctionCall: &genai.FunctionCall{Name: generateTextToolName}},
	}, genai.RoleModel)}

	got, err := afterModelCallback(nil, resp, nil)
	if err != nil {
		t.Fatalf("afterModelCallback() error = %v", err)
	}
	if got != nil {
		t.Errorf("afterModelCallback() = %+v, want nil for a pure function-call response with no text", got)
	}
}

// ===== beforeToolCallback =====

func TestBeforeToolCallback_BlocksOverLimit(t *testing.T) {
	got, err := beforeToolCallback(nil, fakeTool{name: generateTextToolName}, map[string]any{"word_count": 6000})
	if err != nil {
		t.Fatalf("beforeToolCallback() error = %v", err)
	}
	if got == nil || got["status"] != "error" {
		t.Errorf("beforeToolCallback() = %+v, want an error result blocking the call", got)
	}
}

func TestBeforeToolCallback_AllowsAtLimit(t *testing.T) {
	got, err := beforeToolCallback(nil, fakeTool{name: generateTextToolName}, map[string]any{"word_count": wordCountLimit})
	if err != nil {
		t.Fatalf("beforeToolCallback() error = %v", err)
	}
	if got != nil {
		t.Errorf("beforeToolCallback() = %+v, want nil at exactly the limit", got)
	}
}

func TestBeforeToolCallback_IgnoresOtherTools(t *testing.T) {
	got, err := beforeToolCallback(nil, fakeTool{name: "some_other_tool"}, map[string]any{"word_count": 999999})
	if err != nil {
		t.Fatalf("beforeToolCallback() error = %v", err)
	}
	if got != nil {
		t.Errorf("beforeToolCallback() = %+v, want nil for a tool this callback doesn't govern", got)
	}
}

// ===== afterToolCallback =====

func TestAfterToolCallback_RedactsBlockedWordInOutput(t *testing.T) {
	result := map[string]any{"status": "success", "text": "This essay contains unsafe content."}

	got, err := afterToolCallback(nil, fakeTool{name: generateTextToolName}, nil, result, nil)
	if err != nil {
		t.Fatalf("afterToolCallback() error = %v", err)
	}
	if got == nil || got["text"] == result["text"] {
		t.Fatalf("afterToolCallback() = %+v, want the blocked word redacted", got)
	}
}

func TestAfterToolCallback_PassesThroughCleanOutput(t *testing.T) {
	result := map[string]any{"status": "success", "text": "A clean, harmless essay."}

	got, err := afterToolCallback(nil, fakeTool{name: generateTextToolName}, nil, result, nil)
	if err != nil {
		t.Fatalf("afterToolCallback() error = %v", err)
	}
	if got != nil {
		t.Errorf("afterToolCallback() = %+v, want nil for clean output", got)
	}
}

func TestAfterToolCallback_IgnoresOtherTools(t *testing.T) {
	result := map[string]any{"status": "success", "text": "unsafe content here"}

	got, err := afterToolCallback(nil, fakeTool{name: "some_other_tool"}, nil, result, nil)
	if err != nil {
		t.Fatalf("afterToolCallback() error = %v", err)
	}
	if got != nil {
		t.Errorf("afterToolCallback() = %+v, want nil for a tool this callback doesn't govern", got)
	}
}

// ===== error propagation =====

func TestBeforeAgentCallback_PropagatesRealStateError(t *testing.T) {
	wantErr := errors.New("boom")
	ctx := newFakeContext("What is the capital of Italy?")
	ctx.state = &erroringState{err: wantErr}
	cache := &responseCache{}

	_, err := cache.beforeAgentCallback(ctx)
	if !errors.Is(err, wantErr) {
		t.Fatalf("beforeAgentCallback() error = %v, want errors.Is(err, wantErr) — a real failure must not be mistaken for a cache miss", err)
	}
}

func TestAfterAgentCallback_PropagatesRealStateGetError(t *testing.T) {
	wantErr := errors.New("boom")
	ctx := newFakeContext("What is the capital of Italy?")
	ctx.state = &erroringState{err: wantErr}
	cache := &responseCache{}

	_, err := cache.afterAgentCallback(ctx)
	if !errors.Is(err, wantErr) {
		t.Fatalf("afterAgentCallback() error = %v, want errors.Is(err, wantErr)", err)
	}
}

// ===== generateText =====

func TestGenerateText(t *testing.T) {
	got, err := generateText(nil, GenerateTextArgs{Topic: "goroutines", WordCount: 200})
	if err != nil {
		t.Fatalf("generateText() error = %v", err)
	}
	if got.Status != "success" {
		t.Errorf("Status = %q, want %q", got.Status, "success")
	}
	want := "A 200-word essay on goroutines..."
	if got.Text != want {
		t.Errorf("Text = %q, want %q", got.Text, want)
	}
}

// ===== toInt =====

func TestToInt(t *testing.T) {
	tests := []struct {
		name   string
		in     any
		want   int
		wantOK bool
	}{
		{name: "int", in: 42, want: 42, wantOK: true},
		{name: "float64 (JSON-decoded tool argument)", in: float64(42), want: 42, wantOK: true},
		{name: "unsupported type", in: "42", want: 0, wantOK: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := toInt(tt.in)
			if got != tt.want || ok != tt.wantOK {
				t.Errorf("toInt(%v) = (%d, %v), want (%d, %v)", tt.in, got, ok, tt.want, tt.wantOK)
			}
		})
	}
}
