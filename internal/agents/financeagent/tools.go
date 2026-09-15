package financeagent

import "google.golang.org/adk/v2/agent"

// escalationThreshold is the amount above which an investment requires
// supervisor review, in addition to the human confirmation every investment
// already requires.
const escalationThreshold = 10000

// ExecuteInvestmentArgs is the input to executeInvestment.
type ExecuteInvestmentArgs struct {
	Amount float64 `json:"amount"`
}

// ExecuteInvestmentResult is the output of executeInvestment.
type ExecuteInvestmentResult struct {
	Status string `json:"status"`
}

// executeInvestment executes a long-term investment. The caller (via
// functiontool.Config{RequireConfirmation: true} on the wrapping tool) has
// already guaranteed a human approved this exact call before this function
// ever runs — see agent.go. Amounts over escalationThreshold additionally
// hand the conversation off to the supervisor agent by setting
// ctx.Actions().TransferToAgent, confirmed live to trigger the transfer
// itself (via the framework's own auto-injected transfer_to_agent tool,
// which the model calls after seeing the "escalated" status — see the
// Phase 3 build log for the confirmed sequence).
func executeInvestment(ctx agent.Context, args ExecuteInvestmentArgs) (ExecuteInvestmentResult, error) {
	if args.Amount > escalationThreshold {
		ctx.Actions().TransferToAgent = "supervisor"
		return ExecuteInvestmentResult{Status: "escalated"}, nil
	}
	return ExecuteInvestmentResult{Status: "success"}, nil
}
