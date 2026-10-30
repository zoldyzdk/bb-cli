package review

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/zoldyzdk/bb-cli/internal/models"
)

// Optional inference smoke test, never runs during ordinary go test.
func TestLiveHermesPrivateReview(t *testing.T) {
	if os.Getenv("BB_REVIEW_LIVE_TEST") != "1" {
		t.Skip("set BB_REVIEW_LIVE_TEST=1 to use the configured LLM")
	}
	project, err := LoadProject(os.Getenv("BB_REVIEW_PROJECT_DIR"))
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	source := filepath.Join(root, "source")
	control := filepath.Join(root, "control")
	os.MkdirAll(filepath.Join(source, "src"), 0700)
	os.Mkdir(control, 0700)
	app := "import { fetchProfile, fetchSettings } from './api';\nexport async function loadDashboard() {\n  const profile = await fetchProfile();\n  const settings = await fetchSettings();\n  return { profile, settings };\n}\n"
	os.WriteFile(filepath.Join(source, "src/App.tsx"), []byte(app), 0600)
	os.WriteFile(filepath.Join(source, "src/api.ts"), []byte("export const fetchProfile = () => fetch('/api/profile').then(r => r.json());\nexport const fetchSettings = () => fetch('/api/settings').then(r => r.json());\n"), 0600)
	diff := "diff --git a/src/App.tsx b/src/App.tsx\nnew file mode 100644\n--- /dev/null\n+++ b/src/App.tsx\n@@ -0,0 +1,6 @@\n+" + strings.ReplaceAll(strings.TrimSuffix(app, "\n"), "\n", "\n+") + "\n"
	os.WriteFile(filepath.Join(control, "diff.patch"), []byte(diff), 0600)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	pr := &models.PullRequest{ID: 1, Title: "Fixture local: carregamento do dashboard", Source: models.Endpoint{Commit: models.MergeCommit{Hash: strings.Repeat("a", 40)}}, Destination: models.Endpoint{Commit: models.MergeCommit{Hash: strings.Repeat("b", 40)}}}
	started := time.Now().UTC()
	draft, session, err := (HermesRunner{}).Run(ctx, Request{Enrollment: Enrollment{Workspace: "local", Repository: "fixture", Profile: "frontend-review"}, PR: pr, Policy: &Policy{Version: 1}, Checkout: source, Control: control, Diff: diff, ProjectDir: project.Dir})
	if err != nil {
		t.Fatal(err)
	}
	if err := ValidateDraft(&draft, diff); err != nil {
		t.Fatal(err)
	}
	if session == "" {
		t.Fatal("review session was not persisted")
	}
	path, err := SaveReport(project.ReportsDir, &Report{Workspace: "local", Repository: "fixture", PR: 1, Title: pr.Title, SourceCommit: pr.Source.Commit.Hash, DestinationCommit: pr.Destination.Commit.Hash, Profile: "frontend-review", SessionID: session, Started: started, Finished: time.Now().UTC(), Attempts: 1, State: draft.Coverage, Skills: installedSkillHashes("frontend-review", nil), Draft: draft})
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("Live fixture report: %s.md", path)
}
