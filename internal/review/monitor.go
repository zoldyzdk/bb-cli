package review

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/zoldyzdk/bb-cli/internal/models"
)

type Monitor struct {
	Initialized bool              `json:"initialized"`
	Since       time.Time         `json:"since"`
	PRs         map[string]string `json:"prs"`
	path        string
}

func OpenMonitor(directory, repository string) (*Monitor, error) {
	hash := sha256.Sum256([]byte(repository))
	path := filepath.Join(directory, fmt.Sprintf("monitor-%x.json", hash[:8]))
	m := &Monitor{Since: time.Now().UTC(), PRs: map[string]string{}, path: path}
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return m, nil
	}
	if err != nil {
		return nil, err
	}
	if err = json.Unmarshal(data, m); err != nil {
		return nil, fmt.Errorf("invalid monitor state: %w", err)
	}
	if m.PRs == nil {
		return nil, fmt.Errorf("monitor state lacks PR history")
	}
	// An interrupted runner may already have consumed the provider request.
	// Never silently repeat an execution whose outcome is unknown.
	for id, status := range m.PRs {
		if status == "running" {
			m.PRs[id] = "interrupted"
		}
	}
	return m, nil
}

func (m *Monitor) Poll(ctx context.Context, prs []models.PullRequest, run func(context.Context, models.PullRequest) error) error {
	for _, pr := range prs {
		id := strconv.Itoa(pr.ID)
		if _, exists := m.PRs[id]; exists {
			continue
		}
		switch {
		case !m.Initialized || pr.CreatedOn.Before(m.Since):
			m.PRs[id] = "baseline"
		case pr.Draft:
			m.PRs[id] = "draft"
		default:
			m.PRs[id] = "pending"
		}
	}
	m.Initialized = true
	if err := writeJSON(m.path, m); err != nil {
		return err
	}
	for _, pr := range prs {
		id := strconv.Itoa(pr.ID)
		if m.PRs[id] != "pending" {
			continue
		}
		if pr.Draft {
			m.PRs[id] = "draft"
			continue
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		m.PRs[id] = "running"
		if err := writeJSON(m.path, m); err != nil {
			return err
		}
		m.PRs[id] = "done"
		if err := run(ctx, pr); err != nil {
			m.PRs[id] = "failed"
		}
		if err := writeJSON(m.path, m); err != nil {
			return err
		}
	}
	return writeJSON(m.path, m)
}
