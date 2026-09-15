package memory

import (
	"errors"

	"google.golang.org/adk/v2/agent"
	"google.golang.org/adk/v2/session"
)

// stateKey is the single session-state key this package's tools read and
// write. Kept as a named constant since both handlers must agree on it.
const stateKey = "user_name"

// StoreNameArgs holds the name to remember.
type StoreNameArgs struct {
	Name string `json:"name" jsonschema:"the user's name"`
}

// StoreResult is store_name's result.
type StoreResult struct {
	Status string `json:"status"`
}

// RecallNameArgs is empty — recall_name takes no data parameters, matching
// Python's recall_name(tool_context: ToolContext) -> str. An empty struct
// (not omitted entirely) because functiontool.New requires TArgs to be a
// struct or map.
type RecallNameArgs struct{}

// RecallResult is recall_name's result.
type RecallResult struct {
	Name string `json:"name"`
}

// storeName saves args.Name to session state under stateKey.
// ctx.State() is Go's direct equivalent of Python's tool_context.state —
// unlike Python's opt-in tool_context parameter, agent.Context (and its
// State() accessor) is always the first parameter of every function tool.
func storeName(ctx agent.Context, args StoreNameArgs) (StoreResult, error) {
	if err := ctx.State().Set(stateKey, args.Name); err != nil {
		return StoreResult{}, err
	}
	return StoreResult{Status: "success"}, nil
}

// recallName reads stateKey back from session state, returning "Stranger"
// if nothing was ever stored — the Go equivalent of Python's
// tool_context.state.get("user_name", "Stranger"), expressed as an explicit
// errors.Is check against the session package's own not-found sentinel
// rather than a get-with-default method.
func recallName(ctx agent.Context, _ RecallNameArgs) (RecallResult, error) {
	val, err := ctx.State().Get(stateKey)
	if errors.Is(err, session.ErrStateKeyNotExist) {
		return RecallResult{Name: "Stranger"}, nil
	}
	if err != nil {
		return RecallResult{}, err
	}
	// storeName is stateKey's only writer, and it always writes a string, so
	// a failed assertion here can't currently happen; the ignored ok just
	// falls back to "" rather than panicking if that ever stops being true
	// (e.g. a second writer under the same key someday).
	name, _ := val.(string)
	return RecallResult{Name: name}, nil
}
