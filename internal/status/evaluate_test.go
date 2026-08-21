package status

import "testing"

func baseSnapshot() Snapshot {
	return Snapshot{
		ID: 123, Title: "Add search feature", State: "OPEN", Draft: false,
		DestinationBranch: "main",
		ApprovalCount:     2, DefaultReviewerApprovals: 1,
		UnresolvedComments: 0, OpenTasks: 0, SuccessfulBuilds: 1, ConflictCount: 0,
		CommentsAvailable: true, TasksAvailable: true, BuildsAvailable: true,
		ConflictsAvailable: true, BranchRulesAvailable: true,
	}
}

func TestEvaluateReady(t *testing.T) {
	snap := baseSnapshot()
	req := Requirements{
		Approvals:                ThresholdValue(2),
		DefaultReviewerApprovals: ThresholdValue(1),
		SuccessfulBuilds:         ThresholdValue(1),
	}
	res := Evaluate(snap, req)
	if res.Readiness != ReadinessReady {
		t.Fatalf("got %s missing=%v", res.Readiness, res.Missing)
	}
	if res.Missing == nil {
		t.Fatal("Missing is nil, want empty slice")
	}
	if len(res.Missing) != 0 {
		t.Fatalf("Missing: %v", res.Missing)
	}
	if res.Warnings == nil {
		t.Fatal("Warnings is nil, want empty slice")
	}
	if len(res.Warnings) != 0 {
		t.Fatalf("Warnings: %v", res.Warnings)
	}
}

func TestEvaluateNotReady(t *testing.T) {
	snap := baseSnapshot()
	snap.ApprovalCount = 1
	snap.UnresolvedComments = 3
	req := Requirements{Approvals: ThresholdValue(2)}
	res := Evaluate(snap, req)
	if res.Readiness != ReadinessNotReady {
		t.Fatalf("got %s", res.Readiness)
	}
	if len(res.Missing) < 2 {
		t.Fatalf("missing: %v", res.Missing)
	}
}

func TestEvaluateUnknownWhenApprovalsUnavailable(t *testing.T) {
	snap := baseSnapshot()
	req := Requirements{Approvals: ThresholdUnknown(), SuccessfulBuilds: ThresholdNone()}
	res := Evaluate(snap, req)
	if res.Readiness != ReadinessUnknown {
		t.Fatalf("got %s", res.Readiness)
	}
}

func TestEvaluateIgnoreComments(t *testing.T) {
	snap := baseSnapshot()
	snap.UnresolvedComments = 5
	req := Requirements{Approvals: ThresholdValue(2), IgnoreComments: true}
	res := Evaluate(snap, req)
	if res.Readiness != ReadinessReady {
		t.Fatalf("got %s", res.Readiness)
	}
	found := false
	for _, c := range res.Checks {
		if c.Name == "comments" && c.Status == CheckInfo {
			found = true
		}
	}
	if !found {
		t.Fatal("expected INFO comments check")
	}
}

func TestEvaluateDraft(t *testing.T) {
	snap := baseSnapshot()
	snap.Draft = true
	req := Requirements{Approvals: ThresholdValue(2)}
	res := Evaluate(snap, req)
	if res.Readiness != ReadinessNotReady {
		t.Fatalf("got %s", res.Readiness)
	}
}

func TestEvaluateClosed(t *testing.T) {
	for _, state := range []string{"MERGED", "DECLINED"} {
		t.Run(state, func(t *testing.T) {
			snap := baseSnapshot()
			snap.State = state
			req := Requirements{Approvals: ThresholdValue(2)}
			res := Evaluate(snap, req)
			if res.Readiness != ReadinessNotReady {
				t.Fatalf("got %s", res.Readiness)
			}
		})
	}
}

func TestEvaluateConflicts(t *testing.T) {
	snap := baseSnapshot()
	snap.ConflictCount = 1
	req := Requirements{Approvals: ThresholdValue(2)}
	res := Evaluate(snap, req)
	if res.Readiness != ReadinessNotReady {
		t.Fatalf("got %s", res.Readiness)
	}
}

func TestEvaluateUnknownWhenBuildsUnavailable(t *testing.T) {
	snap := baseSnapshot()
	snap.BuildsAvailable = false
	req := Requirements{
		Approvals:        ThresholdValue(2),
		SuccessfulBuilds: ThresholdUnknown(),
	}
	res := Evaluate(snap, req)
	if res.Readiness != ReadinessUnknown {
		t.Fatalf("got %s", res.Readiness)
	}
}

func TestEvaluateBuildsNoneDoesNotForceUnknown(t *testing.T) {
	snap := baseSnapshot()
	snap.BuildsAvailable = false
	req := Requirements{
		Approvals:        ThresholdValue(2),
		SuccessfulBuilds: ThresholdNone(),
	}
	res := Evaluate(snap, req)
	if res.Readiness != ReadinessReady {
		t.Fatalf("got %s missing=%v", res.Readiness, res.Missing)
	}
}
