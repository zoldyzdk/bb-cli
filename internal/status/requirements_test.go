package status

import (
	"testing"

	"github.com/zoldyzdk/bb-cli/internal/models"
)

func TestMatchGlob(t *testing.T) {
	if !matchBranch("main", models.BranchRestriction{BranchMatchKind: "glob", Pattern: "main"}) {
		t.Fatal("exact")
	}
	if !matchBranch("feature/x", models.BranchRestriction{BranchMatchKind: "glob", Pattern: "feature/*"}) {
		t.Fatal("glob")
	}
	if matchBranch("develop", models.BranchRestriction{BranchMatchKind: "glob", Pattern: "main"}) {
		t.Fatal("no match")
	}
}

func TestInferRequirements(t *testing.T) {
	two := 2
	one := 1
	rules := []models.BranchRestriction{
		{Kind: "require_approvals_to_merge", Value: &two, BranchMatchKind: "glob", Pattern: "main"},
		{Kind: "require_passing_builds_to_merge", Value: &one, BranchMatchKind: "glob", Pattern: "main"},
		{Kind: "require_approvals_to_merge", Value: &one, BranchMatchKind: "glob", Pattern: "other"},
	}
	req := InferRequirements(rules, "main", OverrideFlags{})
	if req.Approvals.State != ThresholdStateValue || req.Approvals.Value != 2 {
		t.Fatalf("approvals %+v", req.Approvals)
	}
	if req.SuccessfulBuilds.State != ThresholdStateValue || req.SuccessfulBuilds.Value != 1 {
		t.Fatalf("builds %+v", req.SuccessfulBuilds)
	}
}

func TestInferRequirementsUnavailable(t *testing.T) {
	req := InferRequirements(nil, "main", OverrideFlags{BranchRulesUnavailable: true})
	if req.Approvals.State != ThresholdStateUnknown || req.SuccessfulBuilds.State != ThresholdStateUnknown {
		t.Fatalf("%+v", req)
	}
}

func TestCLIOverrides(t *testing.T) {
	req := InferRequirements(nil, "main", OverrideFlags{
		BranchRulesUnavailable: true,
		RequiredApprovals:      intPtr(3),
		IgnoreBuilds:           true,
	})
	if req.Approvals.State != ThresholdStateValue || req.Approvals.Value != 3 {
		t.Fatalf("%+v", req.Approvals)
	}
	if req.SuccessfulBuilds.State != ThresholdStateNone {
		t.Fatalf("ignore builds should clear requirement, got %+v", req.SuccessfulBuilds)
	}
}

func intPtr(v int) *int { return &v }
