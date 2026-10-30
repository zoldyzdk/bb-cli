package review

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/zoldyzdk/bb-cli/internal/api"
)

func TestReviewRunsAgainstPinnedGitSnapshotAndProducesPrivateDraft(t *testing.T) {
	for _, tc := range []struct {
		name                               string
		status                             int
		policy, instructions, policySource string
		wantError                          bool
		localMode                          string
	}{
		{"repository policy", 200, "version: 1\nskills: []\ninstructions: Use existing APIs\n", "Use existing APIs", "repository:.hermes/review.yaml", false, ""},
		{"missing repository policy", 404, "", "Local baseline", "local:config/default-review.yaml", false, ""},
		{"forbidden repository policy", 403, "", "", "", true, ""},
		{"unauthorized repository policy", 401, "", "", "", true, ""},
		{"invalid repository policy", 200, "version: 999\n", "", "", true, ""},
		{"missing local policy", 404, "", "", "", true, "missing"},
		{"invalid local policy", 404, "", "", "", true, "invalid"},
		{"repository policy needs no local file", 200, "version: 1\nskills: []\ninstructions: Use existing APIs\n", "Use existing APIs", "repository:.hermes/review.yaml", false, "missing"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			repo := t.TempDir()
			git := func(args ...string) string {
				t.Helper()
				cmd := exec.Command("git", args...)
				cmd.Dir = repo
				cmd.Env = append(os.Environ(), "GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_NOSYSTEM=1")
				output, err := cmd.CombinedOutput()
				if err != nil {
					t.Fatalf("git %v: %s", args, output)
				}
				return strings.TrimSpace(string(output))
			}
			git("init", "-q")
			git("config", "user.name", "Fixture")
			git("config", "user.email", "fixture@example.invalid")
			os.WriteFile(filepath.Join(repo, "App.tsx"), []byte("export const count = 1;\n"), 0600)
			git("add", ".")
			git("commit", "-qm", "base")
			destination := git("rev-parse", "HEAD")
			os.WriteFile(filepath.Join(repo, "App.tsx"), []byte("export const count = 2;\n"), 0600)
			git("commit", "-qam", "change")
			source := git("rev-parse", "HEAD")
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != "GET" {
					t.Error("review attempted external write")
				}
				if strings.Contains(r.URL.Path, "/src/") {
					w.WriteHeader(tc.status)
					fmt.Fprint(w, tc.policy)
					return
				}
				fmt.Fprintf(w, `{"id":7,"title":"Change count","source":{"commit":{"hash":"%s"}},"destination":{"commit":{"hash":"%s"}}}`, source, destination)
			}))
			defer server.Close()
			project := &Project{Dir: t.TempDir(), ReportsDir: t.TempDir()}
			if err := os.MkdirAll(filepath.Join(project.Dir, "config"), 0700); err != nil {
				t.Fatal(err)
			}
			if tc.localMode != "missing" {
				localPolicy := "version: 1\nskills: []\ninstructions: Local baseline\n"
				if tc.localMode == "invalid" {
					localPolicy = "version: 999\n"
				}
				if err := os.WriteFile(filepath.Join(project.Dir, "config", "default-review.yaml"), []byte(localPolicy), 0600); err != nil {
					t.Fatal(err)
				}
			}
			runnerCalled := false
			runner := RunnerFunc(func(ctx context.Context, request Request) (Draft, string, error) {
				runnerCalled = true
				if request.Policy.Instructions != tc.instructions {
					t.Fatalf("wrong policy: %+v", request.Policy)
				}
				content, err := os.ReadFile(filepath.Join(request.Checkout, "App.tsx"))
				if err != nil || !strings.Contains(string(content), "2") {
					t.Fatal("wrong source checkout")
				}
				if !strings.Contains(request.Diff, "+export const count = 2;") {
					t.Fatal("wrong pinned diff")
				}
				return Draft{Coverage: "complete", Summary: "Análise concluída", Examined: []string{"App.tsx"}}, "fixture-session", nil
			})
			engine := Engine{Project: project, API: api.NewRepositoryClient("test-token").WithEndpoint(server.URL), Snapshot: GitSnapshotter{LocalURL: repo}, Runner: runner}
			result, err := engine.Review(context.Background(), Enrollment{Workspace: "team", Repository: "app", Profile: "frontend-review"}, 7, "test-token")
			if tc.wantError {
				if err == nil || runnerCalled {
					t.Fatalf("policy error must stop review: %v, runner=%v", err, runnerCalled)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			data, err := os.ReadFile(result + ".json")
			if err != nil || !strings.Contains(string(data), source) || !strings.Contains(string(data), "fixture-session") {
				t.Fatalf("bad report: %s, %v", data, err)
			}
			if !strings.Contains(string(data), tc.policySource) {
				t.Fatalf("policy source missing: %s", data)
			}
			markdown, err := os.ReadFile(result + ".md")
			if err != nil || !strings.Contains(string(markdown), tc.policySource) {
				t.Fatalf("policy source missing from markdown: %s, %v", markdown, err)
			}
			if git("rev-parse", "HEAD") != source {
				t.Fatal("operator checkout was modified")
			}
		})
	}
}

func TestTemporaryAPIFailureStopsAfterThreeAttempts(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		w.WriteHeader(http.StatusServiceUnavailable)
		fmt.Fprint(w, "untrusted provider body")
	}))
	defer server.Close()
	engine := Engine{Project: &Project{ReportsDir: t.TempDir()}, API: api.NewRepositoryClient("fixture-token").WithEndpoint(server.URL), RetryDelay: time.Millisecond}
	path, err := engine.Review(context.Background(), Enrollment{Workspace: "team", Repository: "app", Profile: "frontend-review"}, 1, "fixture-token")
	if err == nil || requests != 3 {
		t.Fatalf("unexpected retry count: %d, %v", requests, err)
	}
	data, readErr := os.ReadFile(path + ".json")
	if readErr != nil || !strings.Contains(string(data), `"attempts": 3`) || strings.Contains(string(data), "untrusted provider body") {
		t.Fatalf("invalid retry report: %s, %v", data, readErr)
	}
}

func TestAuthenticationFailureProducesPrivateFailureReportWithoutRetry(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprint(w, "repository-secret")
	}))
	defer server.Close()
	project := &Project{ReportsDir: t.TempDir()}
	engine := Engine{Project: project, API: api.NewRepositoryClient("repository-secret").WithEndpoint(server.URL)}
	path, err := engine.Review(context.Background(), Enrollment{Workspace: "team", Repository: "app", Profile: "frontend-review"}, 1, "repository-secret")
	if err == nil || requests != 1 {
		t.Fatalf("unexpected authentication retries: %d, %v", requests, err)
	}
	data, err := os.ReadFile(path + ".json")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "repository-secret") || !strings.Contains(string(data), `"state": "failed"`) {
		t.Fatalf("invalid failure report: %s", data)
	}
}
