package review

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/zoldyzdk/bb-cli/internal/models"
)

func TestHermesRunnerPreservesValidPartialResult(t *testing.T) {
	for _, tc := range []struct {
		name, text, providerError string
		code                      int
		wantError                 bool
	}{
		{"completed", `{"coverage":"complete","summary":"Reviewed","findings":[]}`, "", 0, false},
		{"partial", `{"coverage":"incomplete","summary":"Reviewed part","not_examined":["Other files"],"findings":[]}`, "", 1, false},
		{"partial cannot claim complete", `{"coverage":"complete","summary":"Reviewed","findings":[]}`, "", 1, false},
		{"provider failure", `{"coverage":"incomplete","summary":"Partial"}`, "503", 1, true},
		{"invalid partial", "Not a review", "", 1, true},
		{"interrupted", `{"coverage":"complete","summary":"Reviewed"}`, "", 130, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			event, err := json.Marshal(map[string]any{"type": "result", "session_id": "fixture-session", "text": tc.text, "error": tc.providerError, "exit_code": tc.code})
			if err != nil {
				t.Fatal(err)
			}
			progress := newDesktopProgress(&Project{ReportsDir: dir}, Enrollment{Workspace: "team", Repository: "app", Profile: "frontend-review"}, 1)
			executable := filepath.Join(dir, "hermes-fixture")
			script := fmt.Sprintf("#!/usr/bin/python3\nimport sys\nsys.stdout.write(%s)\nsys.exit(%d)\n", strconv.Quote("{\"type\":\"tool_use\",\"name\":\"mcp__review_files__read_file\",\"input\":{\"secret\":\"never-persist\"}}\n"+string(event)+"\n"), tc.code)
			if err := os.WriteFile(executable, []byte(script), 0700); err != nil {
				t.Fatal(err)
			}
			draft, session, err := (HermesRunner{Executable: executable}).Run(context.Background(), Request{Enrollment: Enrollment{Profile: "frontend-review"}, PR: &models.PullRequest{ID: 1}, Policy: &Policy{Version: 1}, Control: dir, Desktop: progress})
			if progress.ToolCalls != 1 || progress.LastTool != "mcp__review_files__read_file" {
				t.Fatalf("progress not captured: %+v", progress)
			}
			progressData, readErr := os.ReadFile(progress.path)
			if readErr != nil || strings.Contains(string(progressData), "never-persist") {
				t.Fatalf("unsafe progress: %s, %v", progressData, readErr)
			}
			if tc.wantError {
				if err == nil {
					t.Fatal("failed execution accepted")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if session != "fixture-session" {
				t.Fatalf("session lost: %s", session)
			}
			if tc.code != 0 && (draft.Coverage != "incomplete" || len(draft.NotExamined) == 0) {
				t.Fatalf("partial result not marked incomplete: %+v", draft)
			}
			if err := ValidateDraft(&draft, ""); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestHermesRunnerSelectsSkillsByProfile(t *testing.T) {
	t.Setenv("HERMES_REVIEW_ENABLED_SKILLS", "must-not-leak")
	for _, tc := range []struct {
		profile, wantSkills string
	}{
		{"code-review", ""},
		{"frontend-review", "vercel-react-best-practices"},
	} {
		t.Run(tc.profile, func(t *testing.T) {
			dir := t.TempDir()
			record := filepath.Join(dir, "invocation.json")
			executable := filepath.Join(dir, "hermes-fixture")
			script := fmt.Sprintf(`#!/usr/bin/python3
import json, os, sys
with open(%s, 'w') as f:
    json.dump({'args': sys.argv[1:], 'skills': os.environ.get('HERMES_REVIEW_ENABLED_SKILLS')}, f)
print(json.dumps({'type':'result','exit_code':0,'session_id':'fixture','text':'{"coverage":"complete","summary":"Reviewed","findings":[]}'}))
`, strconv.Quote(record))
			if err := os.WriteFile(executable, []byte(script), 0700); err != nil {
				t.Fatal(err)
			}
			_, _, err := (HermesRunner{Executable: executable}).Run(context.Background(), Request{Enrollment: Enrollment{Profile: tc.profile}, PR: &models.PullRequest{ID: 32}, Policy: &Policy{Version: 1}, Control: dir})
			if err != nil {
				t.Fatal(err)
			}
			data, err := os.ReadFile(record)
			if err != nil {
				t.Fatal(err)
			}
			var invocation struct {
				Args   []string `json:"args"`
				Skills string   `json:"skills"`
			}
			if err := json.Unmarshal(data, &invocation); err != nil {
				t.Fatal(err)
			}
			if invocation.Skills != tc.wantSkills {
				t.Fatalf("enabled skills = %q, want %q", invocation.Skills, tc.wantSkills)
			}
			preload := ""
			for i, arg := range invocation.Args {
				if arg == "--max-turns" || arg == "--run-budget" {
					t.Fatalf("profile budget overridden by CLI flag %s", arg)
				}
				if arg == "-s" {
					preload = invocation.Args[i+1]
					if tc.wantSkills == "" {
						t.Fatal("generic review unexpectedly passed a skill preload flag")
					}
				}
			}
			if preload != tc.wantSkills {
				t.Fatalf("preloaded skills = %q, want %q", preload, tc.wantSkills)
			}
			if tc.profile == "code-review" && len(installedSkillHashes(tc.profile, nil)) != 0 {
				t.Fatal("generic report includes implicit skills")
			}
		})
	}
}
