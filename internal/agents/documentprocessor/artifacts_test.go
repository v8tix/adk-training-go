package documentprocessor

import (
	"testing"

	"google.golang.org/adk/v2/artifact"
	"google.golang.org/genai"
)

// TestArtifactService_VersionsStartAtOne proves, directly against the real
// artifact.InMemoryService(), the version-numbering finding from Phase 1:
// a file's first save is version 1, not version 0 as Python's own docs
// describe — traced in the SDK's own Save implementation, and proven live
// here rather than just cited.
func TestArtifactService_VersionsStartAtOne(t *testing.T) {
	svc := artifact.InMemoryService()
	const (
		appName   = "doc_processor_test_app"
		userID    = "test_user"
		sessionID = "test_session"
		fileName  = "report.txt"
	)

	first, err := svc.Save(t.Context(), &artifact.SaveRequest{
		AppName: appName, UserID: userID, SessionID: sessionID, FileName: fileName,
		Part: genai.NewPartFromText("v1 content"),
	})
	if err != nil {
		t.Fatalf("Save() first call error = %v", err)
	}
	if first.Version != 1 {
		t.Errorf("first Save() Version = %d, want 1", first.Version)
	}

	second, err := svc.Save(t.Context(), &artifact.SaveRequest{
		AppName: appName, UserID: userID, SessionID: sessionID, FileName: fileName,
		Part: genai.NewPartFromText("v2 content"),
	})
	if err != nil {
		t.Fatalf("Save() second call error = %v", err)
	}
	if second.Version != 2 {
		t.Errorf("second Save() Version = %d, want 2", second.Version)
	}

	loadResp, err := svc.Load(t.Context(), &artifact.LoadRequest{
		AppName: appName, UserID: userID, SessionID: sessionID, FileName: fileName,
	})
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if loadResp.Part.Text != "v2 content" {
		t.Errorf("Load() with no version specified returned %q, want the latest (%q)", loadResp.Part.Text, "v2 content")
	}
}

// TestArtifactService_UserPrefixScopesAcrossSessions proves the user:
// filename-scoping claim directly against the real service: an artifact
// saved under one session ID, with a user:-prefixed name, is visible from a
// completely different session ID for the same app/user — the real
// cross-session persistence Python's docs describe, confirmed live here
// rather than just cited.
func TestArtifactService_UserPrefixScopesAcrossSessions(t *testing.T) {
	svc := artifact.InMemoryService()
	const (
		appName  = "doc_processor_test_app"
		userID   = "test_user"
		fileName = "user:settings.json"
	)

	if _, err := svc.Save(t.Context(), &artifact.SaveRequest{
		AppName: appName, UserID: userID, SessionID: "session_one", FileName: fileName,
		Part: genai.NewPartFromText(`{"theme":"dark"}`),
	}); err != nil {
		t.Fatalf("Save() from session_one error = %v", err)
	}

	loadResp, err := svc.Load(t.Context(), &artifact.LoadRequest{
		AppName: appName, UserID: userID, SessionID: "session_two", FileName: fileName,
	})
	if err != nil {
		t.Fatalf("Load() from session_two error = %v, want the user:-prefixed artifact to be visible across sessions", err)
	}
	if loadResp.Part.Text != `{"theme":"dark"}` {
		t.Errorf("Load() from session_two = %q, want the content session_one saved", loadResp.Part.Text)
	}
}

// TestArtifactService_PlainFilenameIsSessionScoped is the negative case:
// a filename with no user: prefix is NOT visible from a different session,
// proving the scoping distinction actually discriminates rather than every
// filename being cross-session-visible regardless of prefix.
func TestArtifactService_PlainFilenameIsSessionScoped(t *testing.T) {
	svc := artifact.InMemoryService()
	const (
		appName  = "doc_processor_test_app"
		userID   = "test_user"
		fileName = "notes.txt"
	)

	if _, err := svc.Save(t.Context(), &artifact.SaveRequest{
		AppName: appName, UserID: userID, SessionID: "session_one", FileName: fileName,
		Part: genai.NewPartFromText("session one's own notes"),
	}); err != nil {
		t.Fatalf("Save() from session_one error = %v", err)
	}

	if _, err := svc.Load(t.Context(), &artifact.LoadRequest{
		AppName: appName, UserID: userID, SessionID: "session_two", FileName: fileName,
	}); err == nil {
		t.Error("Load() from session_two unexpectedly succeeded — a plain filename must stay session-scoped")
	}
}
