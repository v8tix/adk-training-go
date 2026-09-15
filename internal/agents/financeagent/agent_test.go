package financeagent

import (
	"context"
	"errors"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/v8tix/adk-training-go/internal/infrastructure/llm"
	"google.golang.org/adk/v2/agent"
	"google.golang.org/adk/v2/model"
	"google.golang.org/adk/v2/runner"
	"google.golang.org/adk/v2/tool/toolconfirmation"
	"google.golang.org/genai"
)

var (
	testConfig      llm.Config
	ollamaReachable bool
)

// errNoConfirmationObserved indicates askInvestment's first turn never
// produced the expected adk_request_confirmation FunctionCall — a real
// test-setup failure, not an expected outcome.
var errNoConfirmationObserved = errors.New("no confirmation FunctionCall observed")

func TestMain(m *testing.M) {
	testConfig = llm.LoadConfig()
	ollamaReachable = llm.OllamaReachable(testConfig, 2*time.Second)
	os.Exit(m.Run())
}

// investmentOutcome is what askInvestment observes from driving the full
// confirmation round-trip: whether the real tool ever ran, its final
// status (if it did), whether a rejection was honored (the real handler
// never ran), which agent authored the last event (ends up "supervisor"
// only after a real transfer), and whether the model itself called the
// framework's own transfer_to_agent tool — the confirmed, non-obvious extra
// step between a confirmed escalating call and the actual agent switch
// (see the Phase 3 build log).
type investmentOutcome struct {
	toolRan            bool
	toolStatus         string
	rejectionError     string
	lastAuthor         string
	sawTransferToAgent bool
}

// askInvestment drives BuildRootAgent through two turns: the initial
// request, then a synthesized confirmation FunctionResponse (approve or
// reject per approve). It iterates the full second-turn event stream to
// the end, since — confirmed live (Phase 3 build log) — a transfer
// following a confirmed, escalating call takes one additional model turn
// (the model itself calling the framework's own transfer_to_agent tool),
// not an immediate switch on the same event as the tool's own response.
func askInvestment(ctx context.Context, llmModel model.LLM, appName string, amount float64, approve bool) (investmentOutcome, error) {
	financeAgent, err := BuildRootAgent(llmModel)
	if err != nil {
		return investmentOutcome{}, err
	}

	r, err := runner.NewInMemory(appName, financeAgent)
	if err != nil {
		return investmentOutcome{}, err
	}

	question := "Invest $" + strconv.FormatFloat(amount, 'f', -1, 64) + " for me."
	msg := genai.NewContentFromText(question, genai.RoleUser)
	var confirmCallID string
	for event, runErr := range r.Run(ctx, "test_user", "test_session", msg, agent.RunConfig{}) {
		if runErr != nil {
			return investmentOutcome{}, runErr
		}
		if event.Content == nil {
			continue
		}
		for _, p := range event.Content.Parts {
			if p.FunctionCall != nil && p.FunctionCall.Name == toolconfirmation.FunctionCallName {
				confirmCallID = p.FunctionCall.ID
			}
		}
	}

	if confirmCallID == "" {
		return investmentOutcome{}, errNoConfirmationObserved
	}

	confirmResp := &genai.Content{
		Role: string(genai.RoleUser),
		Parts: []*genai.Part{{
			FunctionResponse: &genai.FunctionResponse{
				Name:     toolconfirmation.FunctionCallName,
				ID:       confirmCallID,
				Response: map[string]any{"confirmed": approve},
			},
		}},
	}

	var out investmentOutcome
	for event, runErr := range r.Run(ctx, "test_user", "test_session", confirmResp, agent.RunConfig{}) {
		if runErr != nil {
			return investmentOutcome{}, runErr
		}
		out.lastAuthor = event.Author
		if event.Content == nil {
			continue
		}
		for _, p := range event.Content.Parts {
			if p.FunctionCall != nil && p.FunctionCall.Name == "transfer_to_agent" {
				out.sawTransferToAgent = true
			}
			if p.FunctionResponse == nil || p.FunctionResponse.Name != "execute_investment" {
				continue
			}
			if errMsg, ok := p.FunctionResponse.Response["error"].(string); ok {
				out.rejectionError = errMsg
				continue
			}
			out.toolRan = true
			if status, ok := p.FunctionResponse.Response["status"].(string); ok {
				out.toolStatus = status
			}
		}
	}

	return out, nil
}

func skipOnQuota(t *testing.T, err error) {
	t.Helper()
	if err != nil && strings.Contains(err.Error(), "RESOURCE_EXHAUSTED") {
		t.Skipf("skipping: Gemini free-tier quota exhausted for today (%v) — external rate limit, not a code defect", err)
	}
}

// skipIfNoOllama skips the test if TEST_BACKEND excludes ollama or no local
// server is reachable, returning a config forced to the Ollama backend
// otherwise. Shared by every _Ollama test pair below — extracted during
// Phase 6 simplify after the same six lines appeared three times.
func skipIfNoOllama(t *testing.T) llm.Config {
	t.Helper()
	if !llm.SelectedTestBackend().IncludesOllama() {
		t.Skip("skipping: TEST_BACKEND excludes ollama")
	}
	if !ollamaReachable {
		t.Skip("skipping: local Ollama server (" + testConfig.OllamaBaseURL + ") is not reachable")
	}
	cfg := testConfig
	cfg.ModelType = llm.ModelTypeOllama
	return cfg
}

// skipIfNoGemini is skipIfNoOllama's Gemini counterpart.
func skipIfNoGemini(t *testing.T) llm.Config {
	t.Helper()
	if !llm.SelectedTestBackend().IncludesGemini() {
		t.Skip("skipping: TEST_BACKEND excludes gemini")
	}
	if testConfig.GoogleAPIKey == "" {
		t.Skip("skipping: GOOGLE_AI_STUDIO_API_KEY is not set")
	}
	cfg := testConfig
	cfg.ModelType = llm.ModelTypeGemini
	return cfg
}

// TestInvestment_SmallAmountApproved_Ollama confirms the full confirmation
// round-trip on the local backend: approve, no escalation, the real tool
// genuinely ran only after approval.
func TestInvestment_SmallAmountApproved_Ollama(t *testing.T) {
	assertSmallAmountApproved(t, skipIfNoOllama(t))
}

// TestInvestment_SmallAmountApproved_Gemini is the same behavior against
// real Gemini — confirmed live this module that both backends genuinely
// support the confirmation round-trip, no forcing needed.
func TestInvestment_SmallAmountApproved_Gemini(t *testing.T) {
	assertSmallAmountApproved(t, skipIfNoGemini(t))
}

func assertSmallAmountApproved(t *testing.T, cfg llm.Config) {
	t.Helper()
	llmModel, _, err := llm.BuildModel(t.Context(), cfg)
	if err != nil {
		t.Fatalf("BuildModel() error = %v", err)
	}

	out, err := askInvestment(t.Context(), llmModel, "finance_test_app_small", 500, true)
	skipOnQuota(t, err)
	if err != nil {
		t.Fatalf("askInvestment() error = %v", err)
	}
	if !out.toolRan {
		t.Fatal("execute_investment never ran after approval")
	}
	if out.toolStatus != "success" {
		t.Errorf("toolStatus = %q, want %q", out.toolStatus, "success")
	}
	if out.lastAuthor == "supervisor" {
		t.Error("a small, approved investment should never transfer to supervisor")
	}
}

// TestInvestment_LargeAmountApproved_Ollama confirms the full escalation
// flow: approve, the real tool escalates, and the conversation genuinely
// ends up transferred to supervisor — the confirmed, non-obvious sequence
// from the Phase 3 build log (one extra model turn between the tool's own
// "escalated" response and the actual transfer).
func TestInvestment_LargeAmountApproved_Ollama(t *testing.T) {
	assertLargeAmountEscalates(t, skipIfNoOllama(t))
}

// TestInvestment_LargeAmountApproved_Gemini is the same behavior against
// real Gemini.
func TestInvestment_LargeAmountApproved_Gemini(t *testing.T) {
	assertLargeAmountEscalates(t, skipIfNoGemini(t))
}

func assertLargeAmountEscalates(t *testing.T, cfg llm.Config) {
	t.Helper()
	llmModel, _, err := llm.BuildModel(t.Context(), cfg)
	if err != nil {
		t.Fatalf("BuildModel() error = %v", err)
	}

	out, err := askInvestment(t.Context(), llmModel, "finance_test_app_large", 50000, true)
	skipOnQuota(t, err)
	if err != nil {
		t.Fatalf("askInvestment() error = %v", err)
	}
	if !out.toolRan {
		t.Fatal("execute_investment never ran after approval")
	}
	if out.toolStatus != "escalated" {
		t.Errorf("toolStatus = %q, want %q", out.toolStatus, "escalated")
	}
	if !out.sawTransferToAgent {
		t.Error("no transfer_to_agent FunctionCall observed — the model never explicitly performed the transfer, only the tool's own \"escalated\" status")
	}
	if out.lastAuthor != "supervisor" {
		t.Errorf("lastAuthor = %q, want %q — the conversation should end up transferred to the supervisor", out.lastAuthor, "supervisor")
	}
}

// TestInvestment_Rejected_Ollama confirms a rejected confirmation is
// honored at the framework level: the real handler never runs, and (per
// finance_instruction.md's explicit guidance, added after a real prompt gap
// found live in Phase 3) the conversation never transfers to supervisor —
// a rejection is final, not an escalation trigger.
func TestInvestment_Rejected_Ollama(t *testing.T) {
	assertRejectionHonored(t, skipIfNoOllama(t))
}

// TestInvestment_Rejected_Gemini is the same behavior against real Gemini.
func TestInvestment_Rejected_Gemini(t *testing.T) {
	assertRejectionHonored(t, skipIfNoGemini(t))
}

func assertRejectionHonored(t *testing.T, cfg llm.Config) {
	t.Helper()
	llmModel, _, err := llm.BuildModel(t.Context(), cfg)
	if err != nil {
		t.Fatalf("BuildModel() error = %v", err)
	}

	out, err := askInvestment(t.Context(), llmModel, "finance_test_app_reject", 500, false)
	skipOnQuota(t, err)
	if err != nil {
		t.Fatalf("askInvestment() error = %v", err)
	}
	if out.toolRan {
		t.Fatal("execute_investment ran despite a rejected confirmation")
	}
	if out.rejectionError == "" {
		t.Fatal("no rejection error observed from execute_investment")
	}
	if !strings.Contains(out.rejectionError, "rejected") {
		t.Errorf("rejectionError = %q, want it to mention the call was rejected (tool.ErrConfirmationRejected), not some other tool error", out.rejectionError)
	}
	if out.sawTransferToAgent || out.lastAuthor == "supervisor" {
		t.Error("a rejected confirmation should never transfer to supervisor")
	}
}
