package documentprocessor

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"sort"
	"strings"
	"testing"

	"google.golang.org/adk/v2/agent"
	"google.golang.org/adk/v2/artifact"
	"google.golang.org/genai"
)

// fakeArtifacts is a minimal, map-backed agent.Artifacts double — 1-indexed
// versions (index 0 of the slice is version 1), matching the real
// InMemoryService/gcsartifact backends exactly (confirmed in Phase 1),
// so a unit test can't encode a version-numbering assumption the real
// service would also get wrong.
type fakeArtifacts struct {
	data map[string][]*genai.Part
}

func (a *fakeArtifacts) Save(_ context.Context, name string, part *genai.Part) (*artifact.SaveResponse, error) {
	if a.data == nil {
		a.data = make(map[string][]*genai.Part)
	}
	a.data[name] = append(a.data[name], part)
	return &artifact.SaveResponse{Version: int64(len(a.data[name]))}, nil
}

func (a *fakeArtifacts) Load(ctx context.Context, name string) (*artifact.LoadResponse, error) {
	parts, ok := a.data[name]
	if !ok || len(parts) == 0 {
		return nil, fmt.Errorf("artifact not found: %w", fs.ErrNotExist)
	}
	return &artifact.LoadResponse{Part: parts[len(parts)-1]}, nil
}

func (a *fakeArtifacts) LoadVersion(_ context.Context, name string, version int) (*artifact.LoadResponse, error) {
	parts, ok := a.data[name]
	if !ok || version < 1 || version > len(parts) {
		return nil, fmt.Errorf("artifact not found: %w", fs.ErrNotExist)
	}
	return &artifact.LoadResponse{Part: parts[version-1]}, nil
}

func (a *fakeArtifacts) List(context.Context) (*artifact.ListResponse, error) {
	names := make([]string, 0, len(a.data))
	for name := range a.data {
		names = append(names, name)
	}
	sort.Strings(names)
	return &artifact.ListResponse{FileNames: names}, nil
}

// erroringArtifacts always fails with a genuine (non-not-found) error — used
// to prove each tool propagates a real artifact.Service failure unchanged,
// distinct from the fs.ErrNotExist "not found" case fakeArtifacts covers.
type erroringArtifacts struct {
	err error
}

func (a *erroringArtifacts) Save(context.Context, string, *genai.Part) (*artifact.SaveResponse, error) {
	return nil, a.err
}
func (a *erroringArtifacts) Load(context.Context, string) (*artifact.LoadResponse, error) {
	return nil, a.err
}
func (a *erroringArtifacts) LoadVersion(context.Context, string, int) (*artifact.LoadResponse, error) {
	return nil, a.err
}
func (a *erroringArtifacts) List(context.Context) (*artifact.ListResponse, error) {
	return nil, a.err
}

// saveErroringArtifacts wraps a working fakeArtifacts but fails every Save
// call — proves a tool propagates a genuine write failure that occurs
// after its own earlier reads already succeeded, distinct from
// erroringArtifacts which fails immediately on the first call.
type saveErroringArtifacts struct {
	*fakeArtifacts
	err error
}

func (a *saveErroringArtifacts) Save(context.Context, string, *genai.Part) (*artifact.SaveResponse, error) {
	return nil, a.err
}

// fakeContext embeds the SDK's own agent.StrictContextMock and overrides
// only Artifacts() — every other Context method panics if a test
// accidentally calls it. artifacts is the agent.Artifacts interface, not
// the concrete *fakeArtifacts, so tests can swap in erroringArtifacts
// without a second context type.
type fakeContext struct {
	agent.StrictContextMock
	artifacts agent.Artifacts
}

func (c *fakeContext) Artifacts() agent.Artifacts {
	return c.artifacts
}

func newFakeContext() *fakeContext {
	return &fakeContext{artifacts: &fakeArtifacts{}}
}

func newFakeContextWithArtifacts(a agent.Artifacts) *fakeContext {
	return &fakeContext{artifacts: a}
}

func TestExtractText_VersionsIncrementAcrossCalls(t *testing.T) {
	ctx := newFakeContext()

	first, err := extractText(ctx, ExtractTextArgs{DocumentName: "Report"})
	if err != nil {
		t.Fatalf("extractText() first call error = %v", err)
	}
	if first.Version != 1 {
		t.Errorf("first extractText() Version = %d, want 1", first.Version)
	}

	second, err := extractText(ctx, ExtractTextArgs{DocumentName: "Report"})
	if err != nil {
		t.Fatalf("extractText() second call error = %v", err)
	}
	if second.Version != 2 {
		t.Errorf("second extractText() Version = %d, want 2 (a re-extract must not overwrite the prior version)", second.Version)
	}
}

func TestExtractText_PropagatesSaveError(t *testing.T) {
	wantErr := errors.New("boom")
	ctx := newFakeContextWithArtifacts(&erroringArtifacts{err: wantErr})

	_, err := extractText(ctx, ExtractTextArgs{DocumentName: "Report"})
	if !errors.Is(err, wantErr) {
		t.Fatalf("extractText() error = %v, want errors.Is(err, wantErr)", err)
	}
}

func TestSummarizeDocument_NotFoundYet(t *testing.T) {
	ctx := newFakeContext()

	got, err := summarizeDocument(ctx, SummarizeDocumentArgs{DocumentName: "Report"})
	if err != nil {
		t.Fatalf("summarizeDocument() error = %v, want no Go error for the not-yet-extracted case", err)
	}
	if got.Status != "not_found" {
		t.Errorf("Status = %q, want %q", got.Status, "not_found")
	}
}

func TestSummarizeDocument_SucceedsAfterExtraction(t *testing.T) {
	ctx := newFakeContext()
	if _, err := extractText(ctx, ExtractTextArgs{DocumentName: "Report"}); err != nil {
		t.Fatalf("extractText() error = %v", err)
	}

	got, err := summarizeDocument(ctx, SummarizeDocumentArgs{DocumentName: "Report"})
	if err != nil {
		t.Fatalf("summarizeDocument() error = %v", err)
	}
	if got.Status != "success" {
		t.Errorf("Status = %q, want %q", got.Status, "success")
	}
	if got.Version != 1 {
		t.Errorf("Version = %d, want 1", got.Version)
	}
}

func TestSummarizeDocument_PropagatesRealLoadError(t *testing.T) {
	wantErr := errors.New("boom")
	ctx := newFakeContextWithArtifacts(&erroringArtifacts{err: wantErr})

	_, err := summarizeDocument(ctx, SummarizeDocumentArgs{DocumentName: "Report"})
	if !errors.Is(err, wantErr) {
		t.Fatalf("summarizeDocument() error = %v, want errors.Is(err, wantErr) — a real failure must not be mistaken for the not-found case", err)
	}
}

func TestSummarizeDocument_PropagatesSaveError(t *testing.T) {
	fa := &fakeArtifacts{}
	if _, err := fa.Save(context.Background(), extractedArtifactName("Report"), genai.NewPartFromText("extracted")); err != nil {
		t.Fatalf("seeding extracted artifact error = %v", err)
	}
	wantErr := errors.New("boom")
	ctx := newFakeContextWithArtifacts(&saveErroringArtifacts{fakeArtifacts: fa, err: wantErr})

	_, err := summarizeDocument(ctx, SummarizeDocumentArgs{DocumentName: "Report"})
	if !errors.Is(err, wantErr) {
		t.Fatalf("summarizeDocument() error = %v, want errors.Is(err, wantErr) — the Load already succeeded, only the Save should fail here", err)
	}
}

func TestGenerateChart_SavesWithImageMimeType(t *testing.T) {
	ctx := newFakeContext()

	got, err := generateChart(ctx, GenerateChartArgs{DocumentName: "Report"})
	if err != nil {
		t.Fatalf("generateChart() error = %v", err)
	}
	if got.Version != 1 {
		t.Errorf("Version = %d, want 1", got.Version)
	}

	loadResp, err := ctx.artifacts.Load(context.Background(), chartArtifactName("Report"))
	if err != nil {
		t.Fatalf("Load(chart) error = %v", err)
	}
	if loadResp.Part.InlineData == nil || loadResp.Part.InlineData.MIMEType != "image/png" {
		t.Errorf("chart artifact InlineData.MIMEType = %+v, want %q", loadResp.Part.InlineData, "image/png")
	}
}

func TestGenerateChart_PropagatesSaveError(t *testing.T) {
	wantErr := errors.New("boom")
	ctx := newFakeContextWithArtifacts(&erroringArtifacts{err: wantErr})

	_, err := generateChart(ctx, GenerateChartArgs{DocumentName: "Report"})
	if !errors.Is(err, wantErr) {
		t.Fatalf("generateChart() error = %v, want errors.Is(err, wantErr)", err)
	}
}

func TestCreateReport_DistinguishesTextAndImage_AndOnlyOwnDocument(t *testing.T) {
	ctx := newFakeContext()

	if _, err := extractText(ctx, ExtractTextArgs{DocumentName: "Alpha"}); err != nil {
		t.Fatalf("extractText(Alpha) error = %v", err)
	}
	if _, err := summarizeDocument(ctx, SummarizeDocumentArgs{DocumentName: "Alpha"}); err != nil {
		t.Fatalf("summarizeDocument(Alpha) error = %v", err)
	}
	if _, err := generateChart(ctx, GenerateChartArgs{DocumentName: "Alpha"}); err != nil {
		t.Fatalf("generateChart(Alpha) error = %v", err)
	}
	// A second, unrelated document's own artifacts must not leak into
	// Alpha's report.
	if _, err := extractText(ctx, ExtractTextArgs{DocumentName: "Beta"}); err != nil {
		t.Fatalf("extractText(Beta) error = %v", err)
	}

	got, err := createReport(ctx, CreateReportArgs{DocumentName: "Alpha"})
	if err != nil {
		t.Fatalf("createReport() error = %v", err)
	}
	if got.Status != "success" {
		t.Errorf("Status = %q, want %q", got.Status, "success")
	}

	loadResp, err := ctx.artifacts.Load(context.Background(), reportArtifactName("Alpha"))
	if err != nil {
		t.Fatalf("Load(report) error = %v", err)
	}
	report := loadResp.Part.Text

	if !strings.Contains(report, "concise summary of the document \"Alpha\"") {
		t.Errorf("report missing Alpha's own summary text: %q", report)
	}
	if !strings.Contains(report, "[Attached Image: "+chartArtifactName("Alpha")+"]") {
		t.Errorf("report missing Alpha's own chart image reference: %q", report)
	}
	if strings.Contains(report, "Beta") {
		t.Errorf("report leaked Beta's own content: %q", report)
	}
}

func TestCreateReport_ExcludesItsOwnPriorVersion(t *testing.T) {
	ctx := newFakeContext()
	if _, err := extractText(ctx, ExtractTextArgs{DocumentName: "Alpha"}); err != nil {
		t.Fatalf("extractText() error = %v", err)
	}
	if _, err := createReport(ctx, CreateReportArgs{DocumentName: "Alpha"}); err != nil {
		t.Fatalf("createReport() first call error = %v", err)
	}

	got, err := createReport(ctx, CreateReportArgs{DocumentName: "Alpha"})
	if err != nil {
		t.Fatalf("createReport() second call error = %v", err)
	}
	if got.Version != 2 {
		t.Errorf("Version = %d, want 2", got.Version)
	}

	loadResp, err := ctx.artifacts.Load(context.Background(), reportArtifactName("Alpha"))
	if err != nil {
		t.Fatalf("Load(report) error = %v", err)
	}
	if n := strings.Count(loadResp.Part.Text, "# Final Report for: Alpha"); n != 1 {
		t.Errorf("report heading appears %d times, want 1 — the second report must not recursively embed the first report's own content: %q", n, loadResp.Part.Text)
	}
}

func TestCreateReport_PropagatesListError(t *testing.T) {
	wantErr := errors.New("boom")
	ctx := newFakeContextWithArtifacts(&erroringArtifacts{err: wantErr})

	_, err := createReport(ctx, CreateReportArgs{DocumentName: "Alpha"})
	if !errors.Is(err, wantErr) {
		t.Fatalf("createReport() error = %v, want errors.Is(err, wantErr)", err)
	}
}

func TestCreateReport_PropagatesSaveError(t *testing.T) {
	fa := &fakeArtifacts{}
	if _, err := fa.Save(context.Background(), extractedArtifactName("Alpha"), genai.NewPartFromText("extracted")); err != nil {
		t.Fatalf("seeding extracted artifact error = %v", err)
	}
	wantErr := errors.New("boom")
	ctx := newFakeContextWithArtifacts(&saveErroringArtifacts{fakeArtifacts: fa, err: wantErr})

	_, err := createReport(ctx, CreateReportArgs{DocumentName: "Alpha"})
	if !errors.Is(err, wantErr) {
		t.Fatalf("createReport() error = %v, want errors.Is(err, wantErr) — the List/Load already succeeded, only the final Save should fail here", err)
	}
}
