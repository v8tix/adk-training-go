package observability

import (
	"errors"
	"testing"

	"google.golang.org/adk/v2/agent"
)

func TestRiskyOperation_Succeeds(t *testing.T) {
	var ctx agent.Context
	got, err := riskyOperation(ctx, RiskyOperationArgs{ShouldFail: false})
	if err != nil {
		t.Fatalf("riskyOperation() error = %v, want nil", err)
	}
	if got.Status != "success" {
		t.Errorf("Status = %q, want %q", got.Status, "success")
	}
}

func TestRiskyOperation_Fails(t *testing.T) {
	var ctx agent.Context
	_, err := riskyOperation(ctx, RiskyOperationArgs{ShouldFail: true})
	if !errors.Is(err, ErrSimulatedFailure) {
		t.Fatalf("riskyOperation() error = %v, want errors.Is(err, ErrSimulatedFailure)", err)
	}
}
