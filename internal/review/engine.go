package review

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/zoldyzdk/bb-cli/internal/api"
	"github.com/zoldyzdk/bb-cli/internal/models"
)

type ReviewAPI interface {
	GetPinnedPullRequest(string, string, int) (*models.PullRequest, error)
	GetSourceFile(string, string, string, string) (string, error)
}

type Request struct {
	Enrollment                          Enrollment
	PR                                  *models.PullRequest
	Policy                              *Policy
	Checkout, Control, Diff, ProjectDir string
	Desktop                             *DesktopProgress
}
type Runner interface {
	Run(context.Context, Request) (Draft, string, error)
}
type RunnerFunc func(context.Context, Request) (Draft, string, error)

func (f RunnerFunc) Run(ctx context.Context, r Request) (Draft, string, error) { return f(ctx, r) }

type Engine struct {
	Project    *Project
	API        ReviewAPI
	Snapshot   Snapshotter
	Runner     Runner
	RetryDelay time.Duration
}

func (e *Engine) Review(parent context.Context, enrollment Enrollment, id int, token string) (prefix string, resultErr error) {
	ctx, cancel := context.WithTimeout(parent, 10*time.Minute)
	defer cancel()
	progress := newDesktopProgress(e.Project, enrollment, id)
	progress.update("preparing")
	report := &Report{Workspace: enrollment.Workspace, Repository: enrollment.Repository, PR: id, Profile: enrollment.Profile, Started: time.Now().UTC(), State: "failed", Skills: map[string]string{}, Draft: Draft{Coverage: "incomplete", Summary: "A revisão não pôde ser concluída.", NotExamined: []string{"Revisão integral do PR"}}}
	defer func() {
		report.Finished = time.Now().UTC()
		if resultErr != nil {
			report.Error = safeError(resultErr)
		}
		path, err := SaveReport(e.Project.ReportsDir, report)
		progress.Report = path
		progress.Summary = report.Draft.Summary
		progress.Coverage = report.Draft.Coverage
		progress.SessionID = report.SessionID
		if resultErr != nil {
			progress.Error = safeError(resultErr)
		}
		if err != nil {
			progress.Error = "Não foi possível salvar o relatório"
		}
		if bridgeErr := publishDesktopSession(e.Project, report); bridgeErr != nil {
			progress.Error = "Histórico salvo, mas a publicação da sessão no Desktop falhou"
		}
		progress.update(report.State)
		prefix = path
		if err != nil {
			resultErr = fmt.Errorf("saving private report: %w", err)
		}
	}()
	var pr *models.PullRequest
	var policy *Policy
	var snapshot *Snapshot
	delay := e.RetryDelay
	if delay == 0 {
		delay = 30 * time.Second
	}
	for attempt := 1; attempt <= 3; attempt++ {
		report.Attempts = attempt
		var err error
		if pr == nil {
			pr, err = e.API.GetPinnedPullRequest(enrollment.Workspace, enrollment.Repository, id)
			if err == nil {
				report.Title = pr.Title
				progress.Title = pr.Title
				progress.update("preparing")
				report.SourceCommit = pr.Source.Commit.Hash
				report.DestinationCommit = pr.Destination.Commit.Hash
			}
		}
		if err == nil && policy == nil {
			var raw string
			report.PolicySource = "repository:.hermes/review.yaml"
			raw, err = e.API.GetSourceFile(enrollment.Workspace, enrollment.Repository, pr.Destination.Commit.Hash, ".hermes/review.yaml")
			var httpErr *api.HTTPError
			if errors.As(err, &httpErr) && httpErr.StatusCode == http.StatusNotFound {
				report.PolicySource = "local:config/default-review.yaml"
				var data []byte
				data, err = os.ReadFile(filepath.Join(e.Project.Dir, "config", "default-review.yaml"))
				if err != nil {
					err = fmt.Errorf("reading local default review policy: %w", err)
				}
				raw = string(data)
			}
			if err == nil {
				policy, err = ParsePolicy(raw, e.Project)
				report.PolicySHA256 = fmt.Sprintf("%x", sha256.Sum256([]byte(raw)))
			}
		}
		if err == nil && snapshot == nil {
			progress.update("checkout")
			snapshot, err = e.Snapshot.Prepare(ctx, enrollment, pr, token)
		}
		if err == nil {
			progress.update("reviewing")
			report.Draft, report.SessionID, err = e.Runner.Run(ctx, Request{Enrollment: enrollment, PR: pr, Policy: policy, Checkout: snapshot.Checkout, Control: snapshot.Control, Diff: snapshot.Diff, ProjectDir: e.Project.Dir, Desktop: progress})
			if err == nil {
				err = ValidateDraft(&report.Draft, snapshot.Diff)
			}
			if err == nil {
				if snapshot.Truncated {
					report.Draft.Coverage = "incomplete"
					report.Draft.NotExamined = append(report.Draft.NotExamined, "Diff excedeu 1 MiB; apenas parte foi fornecida")
				}
				report.State = report.Draft.Coverage
				if snapshot != nil {
					defer os.RemoveAll(snapshot.Root)
				}
				report.Skills = installedSkillHashes(enrollment.Profile, policy.Skills)
				return "", nil
			}
		}
		resultErr = err
		if ctx.Err() != nil {
			resultErr = ctx.Err()
			break
		}
		if attempt == 3 || !retryable(err) {
			break
		}
		wait := delay * time.Duration(1<<(attempt-1))
		var httpErr *api.HTTPError
		if errors.As(err, &httpErr) && httpErr.RetryAfter > wait {
			wait = httpErr.RetryAfter
		}
		select {
		case <-ctx.Done():
			resultErr = ctx.Err()
			attempt = 3
		case <-time.After(wait):
		}
	}
	if snapshot != nil {
		defer os.RemoveAll(snapshot.Root)
	}
	if report.Draft.Coverage == "" {
		report.Draft = Draft{Coverage: "incomplete", Summary: "A revisão falhou.", NotExamined: []string{"Revisão integral do PR"}}
	}
	if resultErr != nil && report.Draft.Coverage == "complete" {
		report.Draft = Draft{Coverage: "incomplete", Summary: "A resposta do Hermes não passou na validação da prévia.", NotExamined: []string{"Resultado utilizável da revisão"}}
	}
	return "", resultErr
}

func retryable(err error) bool {
	var httpErr *api.HTTPError
	if errors.As(err, &httpErr) {
		return httpErr.StatusCode == 429 || httpErr.StatusCode >= 500
	}
	var runErr *RunError
	if errors.As(err, &runErr) {
		return runErr.Retryable
	}
	var network interface{ Timeout() bool }
	return errors.As(err, &network) && network.Timeout()
}

func safeError(err error) string {
	if errors.Is(err, context.DeadlineExceeded) {
		return "Orçamento de dez minutos esgotado"
	}
	if errors.Is(err, context.Canceled) {
		return "Execução interrompida"
	}
	var httpErr *api.HTTPError
	if errors.As(err, &httpErr) {
		return fmt.Sprintf("Bitbucket HTTP %d", httpErr.StatusCode)
	}
	var runErr *RunError
	if errors.As(err, &runErr) {
		return runErr.Message
	}
	// Configuration/Git errors do not contain HTTP bodies, tokens or source text.
	return err.Error()
}

func installedSkillHashes(profile string, extras []string) map[string]string {
	home, _ := os.UserHomeDir()
	root := filepath.Join(home, ".hermes", "profiles", profile, "skills")
	result := map[string]string{}
	for _, name := range reviewSkills(profile, extras) {
		hash := sha256.New()
		err := filepath.WalkDir(filepath.Join(root, name), func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				return nil
			}
			if d.Type()&os.ModeSymlink != 0 {
				return fmt.Errorf("skill symlink refused")
			}
			data, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			rel, _ := filepath.Rel(root, path)
			hash.Write([]byte(rel))
			hash.Write(data)
			return nil
		})
		if err == nil {
			result[name] = fmt.Sprintf("%x", hash.Sum(nil))
		}
	}
	return result
}
