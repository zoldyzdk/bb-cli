package review

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type Finding struct {
	ID             string `json:"id"`
	Severity       string `json:"severity"`
	Path           string `json:"path,omitempty"`
	Line           int    `json:"line,omitempty"`
	Impact         string `json:"impact"`
	Evidence       string `json:"evidence"`
	Recommendation string `json:"recommendation"`
}

type Draft struct {
	Coverage    string    `json:"coverage"`
	Summary     string    `json:"summary"`
	Examined    []string  `json:"examined"`
	NotExamined []string  `json:"not_examined"`
	Findings    []Finding `json:"findings"`
}

type Report struct {
	Version           int               `json:"version"`
	Workspace         string            `json:"workspace"`
	Repository        string            `json:"repository"`
	PR                int               `json:"pr"`
	Title             string            `json:"title"`
	SourceCommit      string            `json:"source_commit"`
	DestinationCommit string            `json:"destination_commit"`
	Profile           string            `json:"profile"`
	SessionID         string            `json:"session_id,omitempty"`
	PolicySource      string            `json:"policy_source,omitempty"`
	PolicySHA256      string            `json:"policy_sha256"`
	Skills            map[string]string `json:"skills_sha256"`
	Run               int               `json:"run"`
	Attempts          int               `json:"attempts"`
	Started           time.Time         `json:"started"`
	Finished          time.Time         `json:"finished"`
	State             string            `json:"state"`
	Error             string            `json:"error,omitempty"`
	Draft             Draft             `json:"draft"`
}

var hunk = regexp.MustCompile(`^@@ -\d+(?:,\d+)? \+(\d+)(?:,\d+)? @@`)
var commitHash = regexp.MustCompile(`^[a-fA-F0-9]{7,64}$`)

func ChangedLines(diff string) map[string]map[int]bool {
	result := map[string]map[int]bool{}
	path, line := "", 0
	for _, text := range strings.Split(diff, "\n") {
		if strings.HasPrefix(text, "diff --git ") {
			path = ""
			line = 0
			continue
		}
		if strings.HasPrefix(text, "+++ b/") {
			path = strings.TrimPrefix(text, "+++ b/")
			result[path] = map[int]bool{}
			continue
		}
		if match := hunk.FindStringSubmatch(text); match != nil {
			line, _ = strconv.Atoi(match[1])
			continue
		}
		if path == "" || line == 0 {
			continue
		}
		if strings.HasPrefix(text, "+") {
			result[path][line] = true
			line++
		} else if strings.HasPrefix(text, " ") {
			line++
		}
	}
	return result
}

func ValidateDraft(d *Draft, diff string) error {
	if d.Coverage != "complete" && d.Coverage != "incomplete" {
		return fmt.Errorf("review coverage must be complete or incomplete")
	}
	if strings.TrimSpace(d.Summary) == "" {
		return fmt.Errorf("review summary is required")
	}
	if len(d.Findings) > 10 {
		return fmt.Errorf("review exceeds ten findings")
	}
	if d.Coverage == "incomplete" && len(d.NotExamined) == 0 {
		return fmt.Errorf("incomplete review must explain missing coverage")
	}
	lines := ChangedLines(diff)
	for i := range d.Findings {
		f := &d.Findings[i]
		if f.Severity != "alta" && f.Severity != "média" && f.Severity != "baixa" {
			return fmt.Errorf("invalid finding severity")
		}
		if strings.TrimSpace(f.Impact) == "" || strings.TrimSpace(f.Evidence) == "" || strings.TrimSpace(f.Recommendation) == "" {
			return fmt.Errorf("finding needs impact, evidence and recommendation")
		}
		if f.Path != "" && (filepath.IsAbs(f.Path) || filepath.Clean(f.Path) == ".." || strings.HasPrefix(filepath.Clean(f.Path), "../")) {
			return fmt.Errorf("finding path escapes repository")
		}
		f.ID = fmt.Sprintf("F%02d", i+1)
		if f.Line < 1 || !lines[f.Path][f.Line] {
			f.Line = 0
		}
	}
	return nil
}

func SaveReport(root string, r *Report) (string, error) {
	if !ValidRepository(r.Workspace, r.Repository) || r.PR < 1 {
		return "", fmt.Errorf("invalid report target")
	}
	commit := r.SourceCommit
	if !commitHash.MatchString(commit) {
		commit = "unknown"
	}
	dir := filepath.Join(root, r.Workspace, r.Repository, fmt.Sprintf("pr-%d", r.PR), commit)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return "", err
	}
	var prefix string
	for n := 1; ; n++ {
		prefix = filepath.Join(dir, fmt.Sprintf("run-%03d", n))
		f, err := os.OpenFile(prefix+".reserved", os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
		if os.IsExist(err) {
			continue
		}
		if err != nil {
			return "", err
		}
		f.Close()
		r.Run = n
		break
	}
	r.Version = 1
	if err := writeJSON(prefix+".json", r); err != nil {
		return "", err
	}
	if err := writeAtomic(prefix+".md", []byte(RenderMarkdown(r))); err != nil {
		return "", err
	}
	return prefix, nil
}

func RenderMarkdown(r *Report) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# Revisão privada: %s/%s, PR #%d\n\n%s\n\n", r.Workspace, r.Repository, r.PR, r.Title)
	fmt.Fprintf(&b, "Estado: **%s**. Cobertura: **%s**. Tentativa manual: %d.\n\nOrigem: `%s`\n\nDestino: `%s`\n\nProfile: `%s`. Sessão Hermes: `%s`.\n\n", r.State, r.Draft.Coverage, r.Run, r.SourceCommit, r.DestinationCommit, r.Profile, r.SessionID)
	if r.PolicySource != "" {
		fmt.Fprintf(&b, "Política: `%s`. SHA256: `%s`.\n\n", r.PolicySource, r.PolicySHA256)
	}
	fmt.Fprintf(&b, "## Resumo\n\n%s\n\n", r.Draft.Summary)
	if r.Error != "" {
		fmt.Fprintf(&b, "Erro: %s\n\n", r.Error)
	}
	for _, section := range []struct {
		name  string
		paths []string
	}{{"Examinado", r.Draft.Examined}, {"Não examinado", r.Draft.NotExamined}} {
		if len(section.paths) > 0 {
			fmt.Fprintf(&b, "## %s\n\n", section.name)
			for _, p := range section.paths {
				fmt.Fprintf(&b, "- %s\n", p)
			}
			b.WriteString("\n")
		}
	}
	for _, f := range r.Draft.Findings {
		fmt.Fprintf(&b, "## %s, severidade %s\n\n", f.ID, f.Severity)
		if f.Line > 0 {
			fmt.Fprintf(&b, "Local: `%s:%d`\n\n", f.Path, f.Line)
		} else if f.Path != "" {
			fmt.Fprintf(&b, "Arquivo: `%s`. Achado geral, sem linha nova apropriada.\n\n", f.Path)
		}
		fmt.Fprintf(&b, "%s\n\nEvidência: %s\n\nCorreção sugerida: %s\n\n", f.Impact, f.Evidence, f.Recommendation)
	}
	if len(r.Draft.Findings) == 0 {
		b.WriteString("Nenhum achado registrado. Isso não garante que o PR esteja correto.\n\n")
	}
	b.WriteString("## Feedback manual\n\nRegistre separadamente achados úteis, incorretos, repetitivos e problemas não encontrados. Nada foi publicado no Bitbucket.\n")
	return b.String()
}
