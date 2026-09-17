package contentmoderator

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"sync"

	"google.golang.org/adk/v2/agent"
	"google.golang.org/adk/v2/model"
	"google.golang.org/adk/v2/session"
	"google.golang.org/adk/v2/tool"
	"google.golang.org/genai"
)

// blockedWords are checked case-insensitively, as whole words, by
// blockedWordPattern below — matching Python's own lab's simple
// substring-based guardrail, just expressed with a word boundary so
// "unsafe" doesn't also flag a word like "unsafely" mid-match.
var blockedWords = []string{"unsafe", "offensive"}

var blockedWordPattern = compileBlockedWordPattern(blockedWords)

func compileBlockedWordPattern(words []string) *regexp.Regexp {
	escaped := make([]string, len(words))
	for i, w := range words {
		escaped[i] = regexp.QuoteMeta(w)
	}
	return regexp.MustCompile(`(?i)\b(` + strings.Join(escaped, "|") + `)\b`)
}

// emailPattern matches a plain, simple email-shaped string — good enough
// for this lab's redaction demo, the same scope Python's own re.sub
// approach uses.
var emailPattern = regexp.MustCompile(`\b[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}\b`)

// wordCountLimit is the maximum word_count generate_text will accept
// before beforeToolCallback blocks the call, matching Python's own lab
// threshold.
const wordCountLimit = 5000

// generateTextToolName names the one tool this agent exposes — shared by
// beforeToolCallback, afterToolCallback, and agent.go's functiontool.Config
// so the three can't silently drift apart on a future rename.
const generateTextToolName = "generate_text"

// cacheKey derives a per-question cache key from the CURRENT turn's user
// input — ctx.UserContent() is the direct equivalent of Python's
// callback_context.get_invocation_context().user_content, already
// available on the one unified agent.Context every callback receives. A
// cache keyed by anything less specific than the actual question (e.g. one
// global key) would return the same cached answer no matter what's asked
// next.
//
// A turn with no text parts (e.g. every blocked-word refusal, whose
// UserContent's own text is what tripped blockedWordPattern) hashes to the
// same sha256("") key as any other empty-text turn, so a second identical
// blocked-word ask is served straight from cache — bypassing
// beforeModelCallback's own guardrail entirely. This is harmless today (the
// cached answer is itself the refusal), but it means neither the cache nor
// the guardrail re-evaluates blockedWords if that list changes later; a real
// deployment would want the cache to expire or the guardrail to run
// regardless of a cache hit.
func cacheKey(ctx agent.Context) string {
	var sb strings.Builder
	if content := ctx.UserContent(); content != nil {
		for _, part := range content.Parts {
			sb.WriteString(part.Text)
		}
	}
	sum := sha256.Sum256([]byte(sb.String()))
	return fmt.Sprintf("cache:%x", sum)
}

// responseCache implements the caching pair (beforeAgentCallback,
// afterAgentCallback) as methods rather than free functions, so a caller —
// this package's own live test included — can read hitCount directly
// instead of inferring a cache hit from the model's own text (which could
// coincidentally repeat itself even on a real, uncached call). mu guards
// hitCount for the same reason modules 25/25.5's own plugin state needed
// one: multiple tool/agent calls within a turn, or across concurrent
// sessions under a multi-user launcher, are real, not hypothetical.
type responseCache struct {
	mu       sync.Mutex
	hitCount int
}

// beforeAgentCallback checks the cache for this exact question. A cache
// hit returns a *genai.Content directly — agent.BeforeAgentCallback's own
// contract skips the agent's run entirely and uses this as the final
// result, the direct equivalent of Python's before_agent_callback
// returning types.Content to bypass the LLM.
func (c *responseCache) beforeAgentCallback(ctx agent.Context) (*genai.Content, error) {
	key := cacheKey(ctx)
	val, err := ctx.State().Get(key)
	if errors.Is(err, session.ErrStateKeyNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	text, _ := val.(string)

	c.mu.Lock()
	c.hitCount++
	count := c.hitCount
	c.mu.Unlock()

	fmt.Printf("💾 [CACHE] Hit #%d for %s — skipping the model call\n", count, key)
	return genai.NewContentFromText(text, genai.RoleModel), nil
}

// outputKey is where llmagent.Config's own OutputKey mechanism saves this
// agent's real final-response text — see agent.go. This is the real,
// sanctioned Go equivalent of Python's own
// "callback_context.session.events, walked in reverse" approach: a
// callback context deliberately does NOT support Session() at all
// (confirmed live — internal/agent/callback_context_wrapper.go logs
// "Session() is not supported for callback context" and returns nil,
// which panics the instant anything calls a method on it). OutputKey's own
// doc comment names exactly this use case ("Extracts agent reply for
// later use, such as in tools, callbacks, etc."), and its real
// implementation (llmagent.go's maybeSaveOutputToState) already
// concatenates every non-Thought part's text for you — the same
// Parts[0]-isn't-the-answer lesson module-25.5 learned the hard way, here
// already handled by the framework itself.
const outputKey = "content_moderator:last_response"

// afterAgentCallback saves this turn's real answer (already written to
// state under outputKey by the time this callback runs) under the
// per-question key beforeAgentCallback reads.
func (c *responseCache) afterAgentCallback(ctx agent.Context) (*genai.Content, error) {
	val, err := ctx.State().Get(outputKey)
	if errors.Is(err, session.ErrStateKeyNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	text, _ := val.(string)
	if text == "" {
		return nil, nil
	}
	return nil, ctx.State().Set(cacheKey(ctx), text)
}

// beforeModelCallback is the input guardrail: if the CURRENT turn's own
// message contains a blocked word, it returns a refusal *model.LLMResponse
// directly instead of letting the real model call happen at all — the
// direct equivalent of Python's before_model_callback returning an
// LlmResponse to skip the model.
//
// Only llmRequest.Contents' own last entry is checked, deliberately —
// found live building this module: llmRequest.Contents carries the WHOLE
// conversation history (every past turn, not just this one), since that's
// what actually gets sent to the model. Checking every entry means a
// single blocked word anywhere in a session's past permanently refuses
// every later turn in that same session, including perfectly ordinary
// follow-up questions — confirmed by watching a live multi-turn test fail
// exactly that way. An input guardrail should judge the current input, not
// something the user already tried and moved on from.
func beforeModelCallback(_ agent.Context, llmRequest *model.LLMRequest) (*model.LLMResponse, error) {
	if len(llmRequest.Contents) == 0 {
		return nil, nil
	}
	currentTurn := llmRequest.Contents[len(llmRequest.Contents)-1]

	var sb strings.Builder
	for _, part := range currentTurn.Parts {
		sb.WriteString(part.Text)
		sb.WriteString(" ")
	}
	match := blockedWordPattern.FindString(sb.String())
	if match == "" {
		return nil, nil
	}
	fmt.Printf("⚠️  [GUARDRAIL] Blocked word %q detected in the request — refusing before calling the model\n", match)
	return &model.LLMResponse{
		Content: genai.NewContentFromText("I'm sorry, but I can't help with that request.", genai.RoleModel),
	}, nil
}

// afterModelCallback redacts email addresses from the model's own
// response. Every non-Thought part is checked, never just Parts[0] — the
// same module-25.5 lesson afterAgentCallback above already applies. A nil
// llmResponseError still gets passed through unchanged when there's
// nothing to redact; a genuine llmResponseError, or a response with no
// content at all (e.g. a pure function-call turn), is left completely
// alone.
func afterModelCallback(_ agent.Context, llmResponse *model.LLMResponse, llmResponseError error) (*model.LLMResponse, error) {
	if llmResponseError != nil || llmResponse == nil || llmResponse.Content == nil {
		return nil, nil
	}

	var redacted bool
	newParts := make([]*genai.Part, len(llmResponse.Content.Parts))
	for i, part := range llmResponse.Content.Parts {
		if part.Thought || part.Text == "" || !emailPattern.MatchString(part.Text) {
			newParts[i] = part
			continue
		}
		newPart := *part
		newPart.Text = emailPattern.ReplaceAllString(part.Text, "[REDACTED EMAIL]")
		newParts[i] = &newPart
		redacted = true
	}
	if !redacted {
		return nil, nil
	}

	fmt.Println("🔒 [FILTER] Redacted an email address from the model's response")
	newContent := *llmResponse.Content
	newContent.Parts = newParts
	newResponse := *llmResponse
	newResponse.Content = &newContent
	return &newResponse, nil
}

// beforeToolCallback validates generate_text's own arguments, blocking an
// excessive word_count before the tool ever runs — the direct equivalent
// of Python's before_tool_callback returning an error dict.
func beforeToolCallback(_ agent.Context, t tool.Tool, args map[string]any) (map[string]any, error) {
	if t.Name() != generateTextToolName {
		return nil, nil
	}
	wordCount, ok := toInt(args["word_count"])
	if !ok || wordCount <= wordCountLimit {
		return nil, nil
	}
	fmt.Printf("⚠️  [VALIDATION] Blocked generate_text: word_count %d exceeds the %d limit\n", wordCount, wordCountLimit)
	return map[string]any{
		"status":  "error",
		"message": fmt.Sprintf("word_count %d exceeds the maximum of %d", wordCount, wordCountLimit),
	}, nil
}

// afterToolCallback audits generate_text's own output as defense-in-depth
// against the model injecting a blocked word via its own generated text
// (not just via the arguments beforeToolCallback already checked),
// redacting any match. A clean result just gets an audit log line.
func afterToolCallback(_ agent.Context, t tool.Tool, _ map[string]any, result map[string]any, err error) (map[string]any, error) {
	if err != nil || t.Name() != generateTextToolName {
		return nil, nil
	}
	text, _ := result["text"].(string)
	if !blockedWordPattern.MatchString(text) {
		fmt.Printf("📋 [AUDIT] tool=%s status=%v\n", t.Name(), result["status"])
		return nil, nil
	}

	fmt.Println("⚠️  [AUDIT] Redacted a blocked word from generate_text's own output")
	redacted := make(map[string]any, len(result))
	for k, v := range result {
		redacted[k] = v
	}
	redacted["text"] = blockedWordPattern.ReplaceAllString(text, "***")
	return redacted, nil
}

// toInt accepts either a Go int (as unit tests construct directly) or a
// float64 (as tool arguments decode from the model's own JSON call) —
// mirroring the same any-typed-numeric-argument handling this repo's
// established since module-22's own state-value reads.
func toInt(v any) (int, bool) {
	switch n := v.(type) {
	case int:
		return n, true
	case float64:
		return int(n), true
	default:
		return 0, false
	}
}
