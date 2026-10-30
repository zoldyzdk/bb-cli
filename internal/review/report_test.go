package review

import (
	"os"
	"strings"
	"testing"
)

func TestDraftDowngradesInvalidInlineAndPreservesRepeatedReports(t *testing.T) {
	diff := "diff --git a/src/App.tsx b/src/App.tsx\n--- a/src/App.tsx\n+++ b/src/App.tsx\n@@ -1,1 +1,2 @@\n old\n+new\n"
	draft := Draft{Coverage: "complete", Summary: "Análise concluída", Findings: []Finding{
		{Severity: "alta", Path: "src/App.tsx", Line: 2, Impact: "Estado incorreto", Evidence: "A linha nova usa estado antigo", Recommendation: "Atualize o estado"},
		{Severity: "média", Path: "src/App.tsx", Line: 999, Impact: "Duplicação", Evidence: "Função existente", Recommendation: "Reutilize a função"},
	}}
	if err := ValidateDraft(&draft, diff); err != nil {
		t.Fatal(err)
	}
	if draft.Findings[0].Line != 2 || draft.Findings[1].Line != 0 {
		t.Fatalf("invalid anchors: %+v", draft.Findings)
	}
	root := t.TempDir()
	report := Report{Workspace: "team", Repository: "app", PR: 12, SourceCommit: strings.Repeat("a", 40), Draft: draft}
	first, err := SaveReport(root, &report)
	if err != nil {
		t.Fatal(err)
	}
	second, err := SaveReport(root, &report)
	if err != nil {
		t.Fatal(err)
	}
	if first == second {
		t.Fatal("manual repetition overwrote previous report")
	}
	markdown, err := os.ReadFile(first + ".md")
	if err != nil || !strings.Contains(string(markdown), "Estado incorreto") {
		t.Fatalf("missing Markdown finding: %v", err)
	}
}
