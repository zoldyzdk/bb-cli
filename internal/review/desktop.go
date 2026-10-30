package review

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

// DesktopProgress is a private, atomic snapshot. Only tool names are exposed,
// never raw provider events, tool inputs, output, credentials or reasoning.
type DesktopProgress struct {
	Version    int       `json:"version"`
	Profile    string    `json:"profile"`
	Workspace  string    `json:"workspace"`
	Repository string    `json:"repository"`
	PR         int       `json:"pr"`
	Title      string    `json:"title"`
	Stage      string    `json:"stage"`
	Started    time.Time `json:"started"`
	Updated    time.Time `json:"updated"`
	SessionID  string    `json:"session_id,omitempty"`
	ToolCalls  int       `json:"tool_calls"`
	LastTool   string    `json:"last_tool,omitempty"`
	Report     string    `json:"report,omitempty"`
	Summary    string    `json:"summary,omitempty"`
	Coverage   string    `json:"coverage,omitempty"`
	Error      string    `json:"error,omitempty"`
	path       string
}

func newDesktopProgress(project *Project, enrollment Enrollment, id int) *DesktopProgress {
	now := time.Now().UTC()
	return &DesktopProgress{Version: 1, Profile: enrollment.Profile, Workspace: enrollment.Workspace, Repository: enrollment.Repository, PR: id, Started: now, path: filepath.Join(project.ReportsDir, ".desktop", enrollment.Profile, fmt.Sprintf("%d.json", now.UnixNano()))}
}

func (p *DesktopProgress) update(stage string) {
	if p == nil {
		return
	}
	p.Stage = stage
	p.Updated = time.Now().UTC()
	if err := writeJSON(p.path, p); err != nil && p.Error == "" {
		p.Error = "Não foi possível salvar o progresso do Desktop"
	}
}

func publishDesktopSession(project *Project, report *Report) error {
	if report.SessionID == "" {
		return nil
	}
	payload, err := json.Marshal(map[string]any{"profile": report.Profile, "session_id": report.SessionID, "workspace": report.Workspace, "repository": report.Repository, "pr": report.PR, "title": report.Title})
	if err != nil {
		return err
	}
	helper := filepath.Join(project.Dir, "scripts", "desktop_bridge.py")
	if _, err := os.Stat(helper); err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "python3", helper, "--publish-session")
	cmd.Env = cleanEnvironment()
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return err
	}
	if err := cmd.Start(); err != nil {
		return err
	}
	_, writeErr := stdin.Write(payload)
	_ = stdin.Close()
	waitErr := cmd.Wait()
	if writeErr != nil {
		return writeErr
	}
	return waitErr
}
