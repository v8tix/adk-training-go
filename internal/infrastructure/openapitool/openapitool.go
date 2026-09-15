// Package openapitool builds ADK tool.Tool/tool.Toolset values from a small
// declarative operation spec — one Go struct per REST operation, no
// hand-written wrapper function per endpoint. This is the Go SDK's answer to
// a real, confirmed gap: google.golang.org/adk/v2 has no OpenAPIToolset
// equivalent at all (confirmed exhaustively, module-11 — no package, no
// type, no dependency anywhere in the SDK). functiontool.New can't fill that
// gap either, since its generics need a compile-time Go struct for
// arguments, not a runtime parameter list. The framework's own tool
// dispatch, though, only requires a type satisfy a small method set
// (confirmed via internal/toolinternal.FunctionTool's shape) — this package
// builds exactly that, by hand, from a spec.
//
// The real HTTP call goes through kawa (github.com/v8tix/kawa), the same
// typed HTTP-call library module-7's bonus path used. Unlike that use case,
// the response shape here isn't known at compile time — it depends on which
// real API an OperationSpec points at — so the response type is a named
// map, not a struct. kawa decodes via jsonx.ReadJSONAs internally, which
// rejects unknown JSON fields for struct targets (confirmed in module-7);
// that check is a no-op for a map target, since a map has no fixed field set
// to be "unknown" against — confirmed by reading jsonx's decoder, which is a
// plain wrapper around stdlib's own encoding/json.Decoder.DisallowUnknownFields.
package openapitool

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/v8tix/kawa"
	"google.golang.org/adk/v2/agent"
	"google.golang.org/adk/v2/model"
	"google.golang.org/adk/v2/tool"
	"google.golang.org/adk/v2/tool/toolutils"
	"google.golang.org/genai"
)

// operationDeadline bounds a single call.
const operationDeadline = 15 * time.Second

// operationMaxRetries covers transient failures (kawa's default
// HTTPStatusPolicy retries 408/429/500/502/503/504) without wasting
// attempts on a permanent error.
const operationMaxRetries = 2

var (
	// ErrMissingOperationID indicates an OperationSpec has no OperationID —
	// it becomes both the tool's name and the map key the framework tracks
	// it under, so it can't be empty.
	ErrMissingOperationID = errors.New("openapitool: missing operation ID")
	// ErrMissingBaseURL indicates an OperationSpec has no BaseURL.
	ErrMissingBaseURL = errors.New("openapitool: missing base URL")
	// ErrMissingPath indicates an OperationSpec has no Path.
	ErrMissingPath = errors.New("openapitool: missing path")
	// ErrDuplicateOperationID indicates two specs passed to the same
	// NewToolset call share an OperationID. The SDK's own toolutils.PackTool
	// would catch this too, but only at first-request time with a less
	// specific error — failing here, at construction, is both earlier and
	// clearer.
	ErrDuplicateOperationID = errors.New("openapitool: duplicate operation ID")
	// ErrCallingAPI indicates the real HTTP call to the operation's
	// endpoint failed — a network error, a timeout, or a malformed
	// response body. This is a genuine plumbing failure, not something the
	// LLM can reason about, so it's a Go error rather than a structured
	// result. A real API-level error response (a valid HTTP reply with a
	// 4xx/5xx status) is not this — see Run.
	ErrCallingAPI = errors.New("openapitool: calling API")
)

// dynamicResponse satisfies kawa.Res. Unlike a typical kawa response type,
// this has no fixed fields, because the real shape depends on which API an
// OperationSpec points at — see the package doc comment for why that's safe
// under kawa's strict decoding.
type dynamicResponse map[string]any

func (dynamicResponse) Res() {}

// ParamSpec describes one query parameter an operation accepts — Go's
// declarative equivalent of one entry in an OpenAPI operation's
// "parameters" list.
type ParamSpec struct {
	// Name is the parameter's name, both in the JSON schema shown to the
	// model and in the query string sent to the API.
	Name string
	// Type is the parameter's JSON Schema type: "string", "number",
	// "integer", or "boolean".
	Type string
	// Description tells the model what this parameter means and how to
	// supply it — the same role Python's docstring Args: line plays for a
	// custom function tool's parameters.
	Description string
	// Required marks the parameter as required in the generated schema.
	Required bool
}

// OperationSpec describes one REST operation to expose as a tool — Go's
// declarative equivalent of one path+method entry in an OpenAPI "paths"
// object. Like Python's own lab (which also builds its spec as a literal
// dict, not a parsed spec file), this is a small Go struct, not a full
// OpenAPI JSON/YAML document — building a general spec parser would solve a
// problem this course's labs don't actually have.
type OperationSpec struct {
	// OperationID becomes the tool's name, exactly as an OpenAPI spec's
	// operationId becomes the generated tool's name in Python.
	OperationID string
	// Summary becomes the tool's description, shown to the model.
	Summary string
	// BaseURL is the API's root, e.g. "https://api.frankfurter.dev/v1".
	BaseURL string
	// Path is the operation's path relative to BaseURL, e.g. "/latest".
	Path string
	// Parameters are sent as query parameters on every call.
	Parameters []ParamSpec
}

func (s OperationSpec) validate() error {
	if s.OperationID == "" {
		return ErrMissingOperationID
	}
	if s.BaseURL == "" {
		return ErrMissingBaseURL
	}
	if s.Path == "" {
		return ErrMissingPath
	}
	return nil
}

// Toolset implements tool.Toolset, producing one tool.Tool per OperationSpec
// it was built from — the direct equivalent of handing Python's
// OpenAPIToolset object into an agent's tools=[toolset]: one value, many
// tools, no per-operation wrapper function to write by hand.
type Toolset struct {
	name  string
	tools []tool.Tool
}

// NewToolset validates each spec and builds a Toolset from them. Each
// operationTool gets its own long-lived HTTP client, built once here rather
// than per Run() call, so repeated calls reuse connections instead of
// paying transport-setup cost every time.
func NewToolset(name string, specs ...OperationSpec) (*Toolset, error) {
	seen := make(map[string]bool, len(specs))
	tools := make([]tool.Tool, 0, len(specs))
	for _, spec := range specs {
		if err := spec.validate(); err != nil {
			return nil, fmt.Errorf("%w (operation %q)", err, spec.OperationID)
		}
		if seen[spec.OperationID] {
			return nil, fmt.Errorf("%w: %q", ErrDuplicateOperationID, spec.OperationID)
		}
		seen[spec.OperationID] = true
		tools = append(tools, &operationTool{
			spec:   spec,
			client: kawa.NewHTTPClient(operationDeadline, nil, nil),
		})
	}
	return &Toolset{name: name, tools: tools}, nil
}

// Name implements tool.Toolset.
func (s *Toolset) Name() string { return s.name }

// Tools implements tool.Toolset.
func (s *Toolset) Tools(_ agent.ReadonlyContext) ([]tool.Tool, error) {
	return s.tools, nil
}

// operationTool is the hand-written tool.Tool for one OperationSpec. It's
// unexported: callers describe an operation declaratively via OperationSpec
// and get a working tool back from NewToolset — they never construct one of
// these directly, the same way a learner never hand-writes a tool class in
// Python's OpenAPIToolset flow either.
//
// Only GET is currently supported — Run always issues a GET with parameters
// baked into the query string. OperationSpec has no HTTP-method field yet;
// a second operation needing a body-carrying method (POST/PUT/PATCH) would
// need one added.
type operationTool struct {
	spec   OperationSpec
	client *http.Client
}

// Name implements tool.Tool.
func (t *operationTool) Name() string { return t.spec.OperationID }

// Description implements tool.Tool.
func (t *operationTool) Description() string { return t.spec.Summary }

// IsLongRunning implements tool.Tool.
func (t *operationTool) IsLongRunning() bool { return false }

// ProcessRequest packs this tool's declaration into the LLM request — the
// same mechanism functiontool's own tools use.
func (t *operationTool) ProcessRequest(_ agent.Context, req *model.LLMRequest) error {
	return toolutils.PackTool(req, t)
}

// Declaration implements the framework's internal tool-dispatch contract,
// building a JSON Schema object straight from Parameters — no reflection,
// since there's no compile-time Go type describing an arbitrary operation's
// arguments.
func (t *operationTool) Declaration() *genai.FunctionDeclaration {
	properties := make(map[string]any, len(t.spec.Parameters))
	required := make([]string, 0, len(t.spec.Parameters))
	for _, p := range t.spec.Parameters {
		properties[p.Name] = map[string]any{
			"type":        p.Type,
			"description": p.Description,
		}
		if p.Required {
			required = append(required, p.Name)
		}
	}
	return &genai.FunctionDeclaration{
		Name:        t.spec.OperationID,
		Description: t.spec.Summary,
		ParametersJsonSchema: map[string]any{
			"type":       "object",
			"properties": properties,
			"required":   required,
		},
	}
}

// formatQueryValue renders v as a query-string value appropriate for
// paramType. Numeric arguments arrive as float64 — the framework decodes
// every JSON number that way, confirmed via internal/llminternal — so the
// generic %v verb would render a large amount in scientific notation
// (fmt.Sprintf("%v", 1000000.0) == "1e+06"), which no real API expects.
// strconv.FormatFloat with 'f' avoids that; every other declared type
// already stringifies correctly with %v.
func formatQueryValue(paramType string, v any) string {
	if paramType == "number" || paramType == "integer" {
		if f, ok := v.(float64); ok {
			return strconv.FormatFloat(f, 'f', -1, 64)
		}
	}
	return fmt.Sprintf("%v", v)
}

// missingRequiredParams returns the names of any required parameters absent
// from args, in spec order.
func missingRequiredParams(spec OperationSpec, args map[string]any) []string {
	var missing []string
	for _, p := range spec.Parameters {
		if p.Required {
			if _, ok := args[p.Name]; !ok {
				missing = append(missing, p.Name)
			}
		}
	}
	return missing
}

// Run implements the framework's internal tool-dispatch contract. args
// arrives as the map[string]any the framework decoded from the model's
// function call. Query parameters are baked into the URL before the call —
// kawa's own Req type describes a request *body*, which a GET here has none
// of; kawa.NoReq (a sentinel it ships for exactly this shape) is used as the
// request type, and the call is made with a nil request pointer, matching
// kawa's own documented "no body" convention.
//
// A genuine plumbing failure (network error, timeout, malformed response
// body) returns a Go error; a real API-level error response (e.g. a 404 for
// an unknown currency code) — or the model omitting a required parameter —
// returns a structured result instead, so the LLM has something to read and
// explain, the same distinction module-9's divide-by-zero handling makes.
// kawa surfaces a plumbing failure as a plain error and a real API error as
// the distinct kawa.ErrInvalidHTTPStatus type, so the two cases are told
// apart by checking for that type, not by inspecting a status code after
// the fact.
func (t *operationTool) Run(ctx agent.Context, args any) (map[string]any, error) {
	m, _ := args.(map[string]any)

	if missing := missingRequiredParams(t.spec, m); len(missing) > 0 {
		return map[string]any{
			"status": "error",
			"error":  fmt.Sprintf("missing required parameter(s): %v", missing),
		}, nil
	}

	q := url.Values{}
	for _, p := range t.spec.Parameters {
		if v, ok := m[p.Name]; ok {
			q.Set(p.Name, formatQueryValue(p.Type, v))
		}
	}

	fullURL := t.spec.BaseURL + t.spec.Path
	if len(q) > 0 {
		fullURL += "?" + q.Encode()
	}

	call := kawa.NewCall[kawa.NoReq, dynamicResponse](t.client, kawa.Get, fullURL).
		WithDeadline(operationDeadline).
		WithMaxRetries(operationMaxRetries)

	env, err := call.DoWithRetry(ctx, nil)
	if err != nil {
		if httpErr, ok := errors.AsType[kawa.ErrInvalidHTTPStatus](err); ok {
			var body map[string]any
			_ = json.Unmarshal(httpErr.Body, &body) // best-effort; body might not be JSON
			return map[string]any{
				"status":     "error",
				"statusCode": httpErr.StatusCode,
				"body":       body,
			}, nil
		}
		return nil, fmt.Errorf("%w: %v", ErrCallingAPI, err)
	}
	// A 204 No Content leaves env.Body nil (confirmed in kawa's own
	// execute()) — not an error, just nothing to report back.
	if env.Body == nil {
		return map[string]any{"status": "success"}, nil
	}
	return map[string]any(*env.Body), nil
}
