package financeagent

import (
	"testing"

	"google.golang.org/adk/v2/agent"
	"google.golang.org/adk/v2/session"
)

// fakeContext embeds the SDK's own agent.StrictContextMock and overrides
// only Actions() — every other method panics loudly if executeInvestment
// ever calls it, exactly the loud-failure behavior StrictContextMock exists
// for.
type fakeContext struct {
	agent.StrictContextMock
	actions session.EventActions
}

func (c *fakeContext) Actions() *session.EventActions {
	return &c.actions
}

func TestExecuteInvestment(t *testing.T) {
	tests := []struct {
		name             string
		amount           float64
		wantStatus       string
		wantTransferedTo string
	}{
		{name: "below threshold succeeds without escalation", amount: 500, wantStatus: "success"},
		{name: "at threshold succeeds without escalation", amount: escalationThreshold, wantStatus: "success"},
		{name: "above threshold escalates to supervisor", amount: escalationThreshold + 1, wantStatus: "escalated", wantTransferedTo: "supervisor"},
		{name: "far above threshold escalates to supervisor", amount: 50000, wantStatus: "escalated", wantTransferedTo: "supervisor"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := &fakeContext{}
			got, err := executeInvestment(ctx, ExecuteInvestmentArgs{Amount: tt.amount})
			if err != nil {
				t.Fatalf("executeInvestment() error = %v", err)
			}
			if got.Status != tt.wantStatus {
				t.Errorf("Status = %q, want %q", got.Status, tt.wantStatus)
			}
			if gotTransfer := ctx.actions.TransferToAgent; gotTransfer != tt.wantTransferedTo {
				t.Errorf("Actions().TransferToAgent = %q, want %q", gotTransfer, tt.wantTransferedTo)
			}
		})
	}
}
