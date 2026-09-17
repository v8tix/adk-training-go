package observability

import (
	"errors"

	"google.golang.org/adk/v2/agent"
)

// ErrSimulatedFailure is riskyOperation's deliberate failure — a genuine Go
// error, not a structured {"status": "error"} result like
// calculator.divide's. llmagent.OnToolErrorCallback only fires on a real
// error return, and this module's whole point is observing that path, the
// same way Python's own lab tool does with `raise ValueError(...)`.
var ErrSimulatedFailure = errors.New("simulated failure")

// RiskyOperationArgs holds whether this call should deliberately fail.
type RiskyOperationArgs struct {
	ShouldFail bool `json:"should_fail" jsonschema:"whether this call should deliberately fail, for testing error handling"`
}

// RiskyOperationResult is risky_operation's result on success.
type RiskyOperationResult struct {
	Status string `json:"status"`
}

// riskyOperation performs an operation that can be made to fail on demand,
// for testing error handling — matching Python's own lab tool exactly.
func riskyOperation(_ agent.Context, args RiskyOperationArgs) (RiskyOperationResult, error) {
	if args.ShouldFail {
		return RiskyOperationResult{}, ErrSimulatedFailure
	}
	return RiskyOperationResult{Status: "success"}, nil
}
