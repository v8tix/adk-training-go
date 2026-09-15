package researchassistant

import (
	"reflect"
	"strings"
	"testing"

	"google.golang.org/adk/v2/agent"
)

func TestExtractKeyFacts(t *testing.T) {
	tests := []struct {
		name string
		args ExtractKeyFactsArgs
		want []string
	}{
		{
			name: "splits on periods and trims whitespace",
			args: ExtractKeyFactsArgs{Text: "First sentence here. Second sentence here. Third.", NumFacts: 5},
			want: []string{"First sentence here", "Second sentence here"},
		},
		{
			name: "drops sentences at or below the minimum length",
			args: ExtractKeyFactsArgs{Text: "Ok. This one is long enough to count.", NumFacts: 5},
			want: []string{"This one is long enough to count"},
		},
		{
			name: "truncates to num_facts",
			args: ExtractKeyFactsArgs{Text: "First long sentence. Second long sentence. Third long sentence.", NumFacts: 2},
			want: []string{"First long sentence", "Second long sentence"},
		},
		{
			name: "defaults num_facts to 5 when zero",
			args: ExtractKeyFactsArgs{Text: "Only one long sentence here.", NumFacts: 0},
			want: []string{"Only one long sentence here"},
		},
		{
			name: "empty text yields no facts",
			args: ExtractKeyFactsArgs{Text: "", NumFacts: 5},
			want: []string{},
		},
		{
			name: "text with no periods is treated as one sentence",
			args: ExtractKeyFactsArgs{Text: "A single sentence with no terminal period", NumFacts: 5},
			want: []string{"A single sentence with no terminal period"},
		},
		{
			name: "num_facts larger than the available sentences returns all of them",
			args: ExtractKeyFactsArgs{Text: "Only one long sentence here.", NumFacts: 100},
			want: []string{"Only one long sentence here"},
		},
		{
			name: "negative num_facts also defaults to 5",
			args: ExtractKeyFactsArgs{Text: "Only one long sentence here.", NumFacts: -1},
			want: []string{"Only one long sentence here"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockCtx := agent.NewStrictContextMock(t.Context())
			got, err := extractKeyFacts(&mockCtx, tt.args)
			if err != nil {
				t.Fatalf("extractKeyFacts() error = %v", err)
			}
			if got.Status != "success" {
				t.Errorf("Status = %q, want %q", got.Status, "success")
			}
			if !reflect.DeepEqual(got.Facts, tt.want) {
				t.Errorf("Facts = %#v, want %#v", got.Facts, tt.want)
			}
		})
	}
}

func TestFormatResearchNotes(t *testing.T) {
	tests := []struct {
		name    string
		args    FormatResearchNotesArgs
		wantDoc []string // substrings the document must contain
	}{
		{
			name: "typical topic and findings",
			args: FormatResearchNotesArgs{Topic: "quantum computing", Findings: "Researchers made progress."},
			wantDoc: []string{
				"# Research Report: quantum computing",
				"## Findings\nResearchers made progress.",
			},
		},
		{
			name:    "empty topic",
			args:    FormatResearchNotesArgs{Topic: "", Findings: "Researchers made progress."},
			wantDoc: []string{"# Research Report: \n", "## Findings\nResearchers made progress."},
		},
		{
			name:    "empty findings",
			args:    FormatResearchNotesArgs{Topic: "quantum computing", Findings: ""},
			wantDoc: []string{"# Research Report: quantum computing", "## Findings\n"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockCtx := agent.NewStrictContextMock(t.Context())
			got, err := formatResearchNotes(&mockCtx, tt.args)
			if err != nil {
				t.Fatalf("formatResearchNotes() error = %v", err)
			}
			if got.Status != "success" {
				t.Errorf("Status = %q, want %q", got.Status, "success")
			}
			for _, want := range tt.wantDoc {
				if !strings.Contains(got.Document, want) {
					t.Errorf("Document = %q, want it to contain %q", got.Document, want)
				}
			}
			if !strings.Contains(got.Document, "Generated: ") {
				t.Errorf("Document = %q, want it to contain a generated timestamp", got.Document)
			}
		})
	}
}
