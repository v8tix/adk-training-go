// Package financeagent defines a secure finance agent demonstrating two
// ToolContext capabilities beyond simple state access (module-10): Human-in-
// the-Loop (HITL) confirmation and dynamic agent transfer.
//
// functiontool.Config.RequireConfirmation wraps executeInvestment so the
// framework pauses for human approval before it ever runs — confirmed live
// this module via the SDK's own toolconfirmation package: the first call
// returns a "requires confirmation" error and a special FunctionCall event
// (toolconfirmation.FunctionCallName), and only a matching FunctionResponse
// (Response: {"confirmed": bool}) resumes the original call.
//
// executeInvestment additionally escalates to a supervisor sub-agent for
// amounts over escalationThreshold, by setting ctx.Actions().TransferToAgent
// — confirmed live that no Workflow wrapper is needed, a plain llmagent with
// SubAgents is sufficient. Confirmed also (a real, non-obvious finding) that
// this transfer does NOT happen in the same turn the confirmed call
// resolves: the model gets one more turn and must itself call the
// framework's own auto-injected transfer_to_agent tool after seeing the
// "escalated" status, before the active agent actually changes. See
// docs/module-13/README.md for the full finding.
package financeagent

import (
	"embed"
	"io/fs"
	"log"

	"github.com/v8tix/adk-training-go/internal/infrastructure/prompts"
	"google.golang.org/adk/v2/agent"
	"google.golang.org/adk/v2/agent/llmagent"
	"google.golang.org/adk/v2/model"
	"google.golang.org/adk/v2/tool"
	"google.golang.org/adk/v2/tool/functiontool"
)

// PromptNamespace identifies this package's entries in the shared prompts
// cache (internal/infrastructure/prompts).
const PromptNamespace = "finance-agent"

//go:embed prompts/*.md
var promptFS embed.FS

func init() {
	promptFiles, err := fs.Sub(promptFS, "prompts")
	if err != nil {
		log.Fatalf("resolving prompts directory: %v", err)
	}
	if err := prompts.Register(PromptNamespace, promptFiles, ".md"); err != nil {
		log.Fatalf("registering prompts: %v", err)
	}
}

// BuildRootAgent constructs the finance agent: a single execute_investment
// tool requiring human confirmation, and a supervisor sub-agent it can
// escalate to.
func BuildRootAgent(llmModel model.LLM) (agent.Agent, error) {
	financeInstruction, err := prompts.Get(PromptNamespace + "/finance_instruction")
	if err != nil {
		return nil, err
	}
	supervisorInstruction, err := prompts.Get(PromptNamespace + "/supervisor_instruction")
	if err != nil {
		return nil, err
	}

	investmentTool, err := functiontool.New(functiontool.Config{
		Name:                "execute_investment",
		Description:         "Executes a long-term investment. Use this tool only when the user explicitly asks to invest or buy.",
		RequireConfirmation: true,
	}, executeInvestment)
	if err != nil {
		return nil, err
	}

	supervisorAgent, err := llmagent.New(llmagent.Config{
		Name:        "supervisor",
		Model:       llmModel,
		Description: "Reviews escalated investment requests and gives a final verdict.",
		Instruction: supervisorInstruction,
	})
	if err != nil {
		return nil, err
	}

	return llmagent.New(llmagent.Config{
		Name:        "finance_agent",
		Model:       llmModel,
		Description: "Helps users with their investments, requiring human approval for every trade.",
		Instruction: financeInstruction,
		Tools:       []tool.Tool{investmentTool},
		SubAgents:   []agent.Agent{supervisorAgent},
	})
}
