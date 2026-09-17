package documentprocessor

import (
	"errors"
	"fmt"
	"io/fs"
	"strings"

	"google.golang.org/adk/v2/agent"
	"google.golang.org/genai"
)

// dummyPNGBytes is a hardcoded 1x1 PNG, matching Python's own lab skeleton
// ("Simulating a binary PNG image" — a real app would use a charting
// library instead; the point here is the artifact mechanism, not rendering).
var dummyPNGBytes = []byte{
	0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a, 0x00, 0x00, 0x00, 0x0d,
	0x49, 0x48, 0x44, 0x52, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01,
	0x08, 0x06, 0x00, 0x00, 0x00, 0x1f, 0x15, 0xc4, 0x89, 0x00, 0x00, 0x00,
	0x0a, 0x49, 0x44, 0x41, 0x54, 0x78, 0x9c, 0x63, 0x00, 0x01, 0x00, 0x00,
	0x05, 0x00, 0x01, 0x0d, 0x0a, 0x2d, 0xb4, 0x00, 0x00, 0x00, 0x00, 0x49,
	0x45, 0x4e, 0x44, 0xae, 0x42, 0x60, 0x82,
}

// ===== extract_text =====

// ExtractTextArgs holds the source document name.
type ExtractTextArgs struct {
	DocumentName string `json:"document_name" jsonschema:"the source document name"`
}

// ExtractTextResult is extract_text's result.
type ExtractTextResult struct {
	Status  string `json:"status"`
	Version int    `json:"version"`
	Message string `json:"message"`
}

// extractText simulates extracting and cleaning a document's text, saving
// it as a new artifact version via ctx.Artifacts() — Go's unified
// equivalent of Python's tool_context.save_artifact.
func extractText(ctx agent.Context, args ExtractTextArgs) (ExtractTextResult, error) {
	content := fmt.Sprintf("EXTRACTED AND CLEANED TEXT FROM DOCUMENT: %s", args.DocumentName)
	name := extractedArtifactName(args.DocumentName)

	resp, err := ctx.Artifacts().Save(ctx, name, genai.NewPartFromText(content))
	if err != nil {
		return ExtractTextResult{}, err
	}

	return ExtractTextResult{
		Status:  "success",
		Version: int(resp.Version),
		Message: fmt.Sprintf("Extracted text saved as %s (version %d)", name, resp.Version),
	}, nil
}

// ===== summarize_document =====

// SummarizeDocumentArgs holds the document name to summarize.
type SummarizeDocumentArgs struct {
	DocumentName string `json:"document_name" jsonschema:"the document name"`
}

// SummarizeDocumentResult is summarize_document's result.
type SummarizeDocumentResult struct {
	Status  string `json:"status"`
	Version int    `json:"version,omitempty"`
	Message string `json:"message"`
}

// summarizeDocument reads back extract_text's own artifact and saves a
// summary as a new one. A missing extracted-text artifact is reported as a
// normal, successful result with a helpful message — not a Go error —
// matching Python's own "return a helpful error message" tool contract.
func summarizeDocument(ctx agent.Context, args SummarizeDocumentArgs) (SummarizeDocumentResult, error) {
	extractedName := extractedArtifactName(args.DocumentName)

	if _, err := ctx.Artifacts().Load(ctx, extractedName); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return SummarizeDocumentResult{
				Status:  "not_found",
				Message: fmt.Sprintf("No extracted text found for %q — run extract_text first.", args.DocumentName),
			}, nil
		}
		return SummarizeDocumentResult{}, err
	}

	summary := fmt.Sprintf("This is a concise summary of the document %q.", args.DocumentName)
	name := summaryArtifactName(args.DocumentName)

	resp, err := ctx.Artifacts().Save(ctx, name, genai.NewPartFromText(summary))
	if err != nil {
		return SummarizeDocumentResult{}, err
	}

	return SummarizeDocumentResult{
		Status:  "success",
		Version: int(resp.Version),
		Message: fmt.Sprintf("Summary saved as %s (version %d)", name, resp.Version),
	}, nil
}

// ===== generate_chart =====

// GenerateChartArgs holds the document name to chart.
type GenerateChartArgs struct {
	DocumentName string `json:"document_name" jsonschema:"the document name"`
}

// GenerateChartResult is generate_chart's result.
type GenerateChartResult struct {
	Status  string `json:"status"`
	Version int    `json:"version"`
	Message string `json:"message"`
}

// generateChart saves a dummy PNG as a binary artifact — this module's one
// demonstration of genai.NewPartFromBytes, Go's direct equivalent of
// Python's types.Part.from_bytes(), always paired with a real MIME type.
func generateChart(ctx agent.Context, args GenerateChartArgs) (GenerateChartResult, error) {
	name := chartArtifactName(args.DocumentName)

	resp, err := ctx.Artifacts().Save(ctx, name, genai.NewPartFromBytes(dummyPNGBytes, "image/png"))
	if err != nil {
		return GenerateChartResult{}, err
	}

	return GenerateChartResult{
		Status:  "success",
		Version: int(resp.Version),
		Message: fmt.Sprintf("Chart saved as %s (version %d)", name, resp.Version),
	}, nil
}

// ===== create_report =====

// CreateReportArgs holds the document name to report on.
type CreateReportArgs struct {
	DocumentName string `json:"document_name" jsonschema:"the document name"`
}

// CreateReportResult is create_report's result.
type CreateReportResult struct {
	Status  string `json:"status"`
	Version int    `json:"version"`
	Message string `json:"message"`
}

// createReport lists every artifact, filters to this document's own files,
// and compiles a Markdown report distinguishing text content from image
// content by MIME type — the same branch Python's own skeleton describes
// ("if it's an image (mime_type starts with 'image/')").
func createReport(ctx agent.Context, args CreateReportArgs) (CreateReportResult, error) {
	listResp, err := ctx.Artifacts().List(ctx)
	if err != nil {
		return CreateReportResult{}, err
	}

	reportName := reportArtifactName(args.DocumentName)
	prefix := args.DocumentName + "_"

	var report strings.Builder
	fmt.Fprintf(&report, "# Final Report for: %s\n\n", args.DocumentName)

	for _, name := range listResp.FileNames {
		if !strings.HasPrefix(name, prefix) || name == reportName {
			continue
		}

		loadResp, err := ctx.Artifacts().Load(ctx, name)
		if err != nil {
			return CreateReportResult{}, err
		}

		part := loadResp.Part
		switch {
		case part.InlineData != nil && strings.HasPrefix(part.InlineData.MIMEType, "image/"):
			fmt.Fprintf(&report, "[Attached Image: %s]\n\n", name)
		default:
			fmt.Fprintf(&report, "%s\n\n", part.Text)
		}
	}

	resp, err := ctx.Artifacts().Save(ctx, reportName, genai.NewPartFromText(report.String()))
	if err != nil {
		return CreateReportResult{}, err
	}

	return CreateReportResult{
		Status:  "success",
		Version: int(resp.Version),
		Message: fmt.Sprintf("Report saved as %s (version %d)", reportName, resp.Version),
	}, nil
}

// ===== shared helpers =====

func extractedArtifactName(documentName string) string {
	return documentName + "_extracted.txt"
}

func summaryArtifactName(documentName string) string {
	return documentName + "_summary.txt"
}

func chartArtifactName(documentName string) string {
	return documentName + "_chart.png"
}

func reportArtifactName(documentName string) string {
	return documentName + "_FINAL_REPORT.md"
}
