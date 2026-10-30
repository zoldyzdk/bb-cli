package review

import (
	"context"
	"testing"
	"time"

	"github.com/zoldyzdk/bb-cli/internal/models"
)

func TestMonitorBaselinesRecoversNewPRsAndNeverRepeatsDrafts(t *testing.T) {
	ctx := context.Background()
	state := t.TempDir()
	monitor, err := OpenMonitor(state, "team/app")
	if err != nil {
		t.Fatal(err)
	}
	old := models.PullRequest{ID: 1, CreatedOn: time.Now().Add(-time.Hour)}
	newPR := models.PullRequest{ID: 2, CreatedOn: time.Now()}
	draft := models.PullRequest{ID: 3, CreatedOn: time.Now(), Draft: true}
	var reviewed []int
	run := func(_ context.Context, pr models.PullRequest) error { reviewed = append(reviewed, pr.ID); return nil }
	if err := monitor.Poll(ctx, []models.PullRequest{old}, run); err != nil {
		t.Fatal(err)
	}
	monitor, err = OpenMonitor(state, "team/app")
	if err != nil {
		t.Fatal(err)
	}
	if err := monitor.Poll(ctx, []models.PullRequest{old, newPR, draft}, run); err != nil {
		t.Fatal(err)
	}
	draft.Draft = false
	if err := monitor.Poll(ctx, []models.PullRequest{old, newPR, draft}, run); err != nil {
		t.Fatal(err)
	}
	if len(reviewed) != 1 || reviewed[0] != 2 {
		t.Fatalf("reviewed %v, want only PR 2", reviewed)
	}
}
