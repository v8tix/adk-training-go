package personaltutor

import (
	"testing"

	"google.golang.org/adk/v2/memory"
	"google.golang.org/adk/v2/session"
	"google.golang.org/genai"
)

// TestMemoryService_AddAndSearch proves the real memory.Service mechanism
// genuinely works — no LLM call needed. This module's own search_past_lessons
// tool only simulates a search over user:topics (matching Python's own lab
// scope, which notes "in production, this would use
// MemoryService.search_memory()"), so this test is what actually exercises
// AddSessionToMemory/SearchMemory instead of leaving them purely theoretical.
func TestMemoryService_AddAndSearch(t *testing.T) {
	const (
		appName = "personal_tutor_memory_test"
		userID  = "test_user"
	)

	sessionService := session.InMemoryService()
	createResp, err := sessionService.Create(t.Context(), &session.CreateRequest{
		AppName:   appName,
		UserID:    userID,
		SessionID: "past_session",
	})
	if err != nil {
		t.Fatalf("sessionService.Create() error = %v", err)
	}

	event := session.NewEvent(t.Context(), "test_invocation")
	event.Author = "model"
	event.Content = genai.NewContentFromText(
		"Goroutines are lightweight threads managed by the Go runtime, started with the go keyword.",
		genai.RoleModel,
	)
	if err := sessionService.AppendEvent(t.Context(), createResp.Session, event); err != nil {
		t.Fatalf("sessionService.AppendEvent() error = %v", err)
	}

	getResp, err := sessionService.Get(t.Context(), &session.GetRequest{
		AppName:   appName,
		UserID:    userID,
		SessionID: "past_session",
	})
	if err != nil {
		t.Fatalf("sessionService.Get() error = %v", err)
	}

	memoryService := memory.InMemoryService()
	if err := memoryService.AddSessionToMemory(t.Context(), getResp.Session); err != nil {
		t.Fatalf("AddSessionToMemory() error = %v", err)
	}

	searchResp, err := memoryService.SearchMemory(t.Context(), &memory.SearchRequest{
		Query:   "goroutines",
		UserID:  userID,
		AppName: appName,
	})
	if err != nil {
		t.Fatalf("SearchMemory() error = %v", err)
	}

	if len(searchResp.Memories) == 0 {
		t.Fatal("SearchMemory() returned no memories, want at least one match for a query word actually present in the stored session")
	}
	entry := searchResp.Memories[0]
	if entry.Content == nil || len(entry.Content.Parts) == 0 || entry.Content.Parts[0].Text == "" {
		t.Fatalf("SearchMemory() first result has no text content: %+v", entry)
	}
	if entry.Author != "model" {
		t.Errorf("SearchMemory() first result Author = %q, want %q", entry.Author, "model")
	}
}

// TestMemoryService_SearchWithNoMatch confirms a query with no matching
// words returns no results, rather than a false positive — the negative
// case that proves SearchMemory can actually discriminate.
func TestMemoryService_SearchWithNoMatch(t *testing.T) {
	const (
		appName = "personal_tutor_memory_test_empty"
		userID  = "test_user"
	)

	sessionService := session.InMemoryService()
	createResp, err := sessionService.Create(t.Context(), &session.CreateRequest{
		AppName:   appName,
		UserID:    userID,
		SessionID: "past_session",
	})
	if err != nil {
		t.Fatalf("sessionService.Create() error = %v", err)
	}

	event := session.NewEvent(t.Context(), "test_invocation")
	event.Author = "model"
	event.Content = genai.NewContentFromText("Channels coordinate communication between goroutines.", genai.RoleModel)
	if err := sessionService.AppendEvent(t.Context(), createResp.Session, event); err != nil {
		t.Fatalf("sessionService.AppendEvent() error = %v", err)
	}

	getResp, err := sessionService.Get(t.Context(), &session.GetRequest{
		AppName:   appName,
		UserID:    userID,
		SessionID: "past_session",
	})
	if err != nil {
		t.Fatalf("sessionService.Get() error = %v", err)
	}

	memoryService := memory.InMemoryService()
	if err := memoryService.AddSessionToMemory(t.Context(), getResp.Session); err != nil {
		t.Fatalf("AddSessionToMemory() error = %v", err)
	}

	searchResp, err := memoryService.SearchMemory(t.Context(), &memory.SearchRequest{
		Query:   "recursion",
		UserID:  userID,
		AppName: appName,
	})
	if err != nil {
		t.Fatalf("SearchMemory() error = %v", err)
	}
	if len(searchResp.Memories) != 0 {
		t.Errorf("SearchMemory() with an unrelated query returned %d memories, want 0", len(searchResp.Memories))
	}
}
