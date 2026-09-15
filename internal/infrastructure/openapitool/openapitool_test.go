package openapitool

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"google.golang.org/adk/v2/agent"
	"google.golang.org/genai"
)

// runnableTool mirrors the framework's own internal dispatch contract
// (internal/toolinternal.FunctionTool) closely enough for these tests to
// reach Declaration()/Run() on the tool.Tool values Toolset.Tools returns,
// without needing to import an internal package.
type runnableTool interface {
	Declaration() *genai.FunctionDeclaration
	Run(ctx agent.Context, args any) (map[string]any, error)
}

// fakeContext wraps the SDK's own agent.StrictContextMock, matching
// module-10's pattern. Run only actually uses agent.Context's embedded
// context.Context (kawa needs a real one — confirmed live, a literal nil
// panics inside kawa's retry backoff); every other method still panics if a
// test accidentally calls it.
func fakeContext(t *testing.T) agent.Context {
	t.Helper()
	mock := agent.NewStrictContextMock(t.Context())
	return &mock
}

func validSpec(baseURL string) OperationSpec {
	return OperationSpec{
		OperationID: "get_latest_rates",
		Summary:     "Get latest exchange rates",
		BaseURL:     baseURL,
		Path:        "/latest",
		Parameters: []ParamSpec{
			{Name: "amount", Type: "number", Description: "the amount to convert", Required: true},
			{Name: "from", Type: "string", Description: "the currency to convert from", Required: true},
			{Name: "to", Type: "string", Description: "the currency to convert to", Required: true},
		},
	}
}

func TestNewToolset_Validation(t *testing.T) {
	tests := []struct {
		name    string
		spec    OperationSpec
		wantErr error
	}{
		{name: "valid spec", spec: validSpec("https://example.com")},
		{name: "missing operation ID", spec: OperationSpec{BaseURL: "https://example.com", Path: "/x"}, wantErr: ErrMissingOperationID},
		{name: "missing base URL", spec: OperationSpec{OperationID: "op", Path: "/x"}, wantErr: ErrMissingBaseURL},
		{name: "missing path", spec: OperationSpec{OperationID: "op", BaseURL: "https://example.com"}, wantErr: ErrMissingPath},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewToolset("test", tt.spec)
			if tt.wantErr == nil {
				if err != nil {
					t.Fatalf("NewToolset() error = %v, want nil", err)
				}
				return
			}
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("NewToolset() error = %v, want errors.Is(err, %v)", err, tt.wantErr)
			}
		})
	}
}

func TestNewToolset_RejectsDuplicateOperationID(t *testing.T) {
	dup := validSpec("https://example.com")
	_, err := NewToolset("test", dup, dup)
	if !errors.Is(err, ErrDuplicateOperationID) {
		t.Fatalf("NewToolset() error = %v, want errors.Is(err, ErrDuplicateOperationID)", err)
	}
}

func TestToolset_ProducesOneToolPerOperation(t *testing.T) {
	ts, err := NewToolset("frankfurter", validSpec("https://example.com"), OperationSpec{
		OperationID: "get_historical_rates",
		Summary:     "Get historical exchange rates",
		BaseURL:     "https://example.com",
		Path:        "/2020-01-01",
	})
	if err != nil {
		t.Fatalf("NewToolset() error = %v", err)
	}
	if ts.Name() != "frankfurter" {
		t.Errorf("Name() = %q, want %q", ts.Name(), "frankfurter")
	}
	tools, err := ts.Tools(nil)
	if err != nil {
		t.Fatalf("Tools() error = %v", err)
	}
	if len(tools) != 2 {
		t.Fatalf("Tools() returned %d tools, want 2", len(tools))
	}
	if tools[0].Name() != "get_latest_rates" || tools[1].Name() != "get_historical_rates" {
		t.Errorf("Tools() names = [%q, %q], want [get_latest_rates, get_historical_rates]", tools[0].Name(), tools[1].Name())
	}
}

func TestOperationTool_Declaration(t *testing.T) {
	ts, err := NewToolset("frankfurter", validSpec("https://example.com"))
	if err != nil {
		t.Fatalf("NewToolset() error = %v", err)
	}
	tools, _ := ts.Tools(nil)
	rt := tools[0].(runnableTool)

	decl := rt.Declaration()
	if decl.Name != "get_latest_rates" {
		t.Errorf("Declaration().Name = %q, want %q", decl.Name, "get_latest_rates")
	}
	schema, ok := decl.ParametersJsonSchema.(map[string]any)
	if !ok {
		t.Fatalf("ParametersJsonSchema is %T, want map[string]any", decl.ParametersJsonSchema)
	}
	properties, ok := schema["properties"].(map[string]any)
	if !ok {
		t.Fatalf("schema[\"properties\"] is %T, want map[string]any", schema["properties"])
	}
	for _, name := range []string{"amount", "from", "to"} {
		if _, ok := properties[name]; !ok {
			t.Errorf("properties missing %q", name)
		}
	}
	required, ok := schema["required"].([]string)
	if !ok || len(required) != 3 {
		t.Errorf("schema[\"required\"] = %v, want all three parameter names", schema["required"])
	}
}

func TestOperationTool_Run_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("from"); got != "USD" {
			t.Errorf("request query \"from\" = %q, want %q", got, "USD")
		}
		if got := r.URL.Query().Get("to"); got != "EUR" {
			t.Errorf("request query \"to\" = %q, want %q", got, "EUR")
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"amount":100.0,"base":"USD","date":"2026-01-01","rates":{"EUR":86.57}}`))
	}))
	defer server.Close()

	ts, err := NewToolset("frankfurter", validSpec(server.URL))
	if err != nil {
		t.Fatalf("NewToolset() error = %v", err)
	}
	tools, _ := ts.Tools(nil)
	rt := tools[0].(runnableTool)

	result, err := rt.Run(fakeContext(t), map[string]any{"amount": float64(100), "from": "USD", "to": "EUR"})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if result["base"] != "USD" {
		t.Errorf("result[\"base\"] = %v, want %q", result["base"], "USD")
	}
	rates, ok := result["rates"].(map[string]any)
	if !ok {
		t.Fatalf("result[\"rates\"] is %T, want map[string]any", result["rates"])
	}
	if _, ok := rates["EUR"]; !ok {
		t.Errorf("result[\"rates\"] missing \"EUR\": %v", rates)
	}
}

func TestOperationTool_Run_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"message":"not found"}`))
	}))
	defer server.Close()

	ts, err := NewToolset("frankfurter", validSpec(server.URL))
	if err != nil {
		t.Fatalf("NewToolset() error = %v", err)
	}
	tools, _ := ts.Tools(nil)
	rt := tools[0].(runnableTool)

	// A real API-level error response must come back as a structured
	// result the LLM can read and explain, not a Go error — the same
	// distinction module-9's divide-by-zero handling makes.
	result, err := rt.Run(fakeContext(t), map[string]any{"amount": float64(100), "from": "XYZ", "to": "EUR"})
	if err != nil {
		t.Fatalf("Run() error = %v, want a structured error result instead", err)
	}
	if result["status"] != "error" {
		t.Errorf("result[\"status\"] = %v, want %q", result["status"], "error")
	}
	if result["statusCode"] != http.StatusNotFound {
		t.Errorf("result[\"statusCode\"] = %v, want %d", result["statusCode"], http.StatusNotFound)
	}
}

func TestOperationTool_Run_NetworkFailure(t *testing.T) {
	ts, err := NewToolset("frankfurter", validSpec("http://127.0.0.1:0"))
	if err != nil {
		t.Fatalf("NewToolset() error = %v", err)
	}
	tools, _ := ts.Tools(nil)
	rt := tools[0].(runnableTool)

	_, err = rt.Run(fakeContext(t), map[string]any{"amount": float64(100), "from": "USD", "to": "EUR"})
	if !errors.Is(err, ErrCallingAPI) {
		t.Fatalf("Run() error = %v, want errors.Is(err, ErrCallingAPI)", err)
	}
}

// TestOperationTool_Run_LargeAmountFormatting proves numeric query
// parameters avoid scientific notation. The framework decodes every JSON
// number argument as float64 (confirmed via internal/llminternal); the
// generic %v verb renders a large float64 as "1e+06", which no real API
// expects. This test uses a value large enough to trigger that formatting
// under %v, so it would have caught the bug the earlier tests (which passed
// plain Go int literals) could not.
func TestOperationTool_Run_LargeAmountFormatting(t *testing.T) {
	var gotAmount string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAmount = r.URL.Query().Get("amount")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"amount":1000000.0,"base":"USD","date":"2026-01-01","rates":{"EUR":865700}}`))
	}))
	defer server.Close()

	ts, err := NewToolset("frankfurter", validSpec(server.URL))
	if err != nil {
		t.Fatalf("NewToolset() error = %v", err)
	}
	tools, _ := ts.Tools(nil)
	rt := tools[0].(runnableTool)

	if _, err := rt.Run(fakeContext(t), map[string]any{"amount": float64(1000000), "from": "USD", "to": "EUR"}); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if gotAmount != "1000000" {
		t.Errorf("request query \"amount\" = %q, want %q (not scientific notation)", gotAmount, "1000000")
	}
}

// TestOperationTool_Run_MissingRequiredParam proves an omitted required
// parameter becomes a structured result the model can read and explain,
// not a Go error and not a request silently sent without it.
func TestOperationTool_Run_MissingRequiredParam(t *testing.T) {
	called := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))
	defer server.Close()

	ts, err := NewToolset("frankfurter", validSpec(server.URL))
	if err != nil {
		t.Fatalf("NewToolset() error = %v", err)
	}
	tools, _ := ts.Tools(nil)
	rt := tools[0].(runnableTool)

	// "from" is required but omitted.
	result, err := rt.Run(fakeContext(t), map[string]any{"amount": float64(100), "to": "EUR"})
	if err != nil {
		t.Fatalf("Run() error = %v, want a structured error result instead", err)
	}
	if result["status"] != "error" {
		t.Errorf("result[\"status\"] = %v, want %q", result["status"], "error")
	}
	if called {
		t.Error("the real API was called despite a missing required parameter — it should have been rejected first")
	}
}

// TestOperationTool_Run_NoContent proves a 204 response (env.Body == nil in
// kawa's own execute()) doesn't panic on the unconditional dereference it
// used to.
func TestOperationTool_Run_NoContent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	ts, err := NewToolset("frankfurter", validSpec(server.URL))
	if err != nil {
		t.Fatalf("NewToolset() error = %v", err)
	}
	tools, _ := ts.Tools(nil)
	rt := tools[0].(runnableTool)

	result, err := rt.Run(fakeContext(t), map[string]any{"amount": float64(100), "from": "USD", "to": "EUR"})
	if err != nil {
		t.Fatalf("Run() error = %v, want no error for a 204 response", err)
	}
	if result == nil {
		t.Error("Run() returned a nil result for a 204 response, want a non-nil map")
	}
}
