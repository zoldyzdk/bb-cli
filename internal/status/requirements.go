package status

import (
	"path"

	"github.com/zoldyzdk/bb-cli/internal/models"
)

type OverrideFlags struct {
	BranchRulesUnavailable           bool
	RequiredApprovals                *int
	RequiredDefaultReviewerApprovals *int
	RequiredSuccessfulBuilds         *int
	IgnoreComments                   bool
	IgnoreTasks                      bool
	IgnoreBuilds                     bool
}

// matchBranch reports whether dest matches r. For glob (or empty kind), Pattern
// is matched with path.Match. branching_model is never a match in v1; we do
// not invent branch types.
func matchBranch(dest string, r models.BranchRestriction) bool {
	switch r.BranchMatchKind {
	case "glob", "":
		ok, err := path.Match(r.Pattern, dest)
		return err == nil && ok
	case "branching_model":
		return false
	default:
		return false
	}
}

func InferRequirements(rules []models.BranchRestriction, dest string, flags OverrideFlags) Requirements {
	req := Requirements{}
	if flags.BranchRulesUnavailable {
		req.Approvals = ThresholdUnknown()
		req.DefaultReviewerApprovals = ThresholdUnknown()
		req.SuccessfulBuilds = ThresholdUnknown()
	} else {
		req.Approvals = ThresholdNone()
		req.DefaultReviewerApprovals = ThresholdNone()
		req.SuccessfulBuilds = ThresholdNone()
	}

	for _, r := range rules {
		if !matchBranch(dest, r) || r.Value == nil {
			continue
		}
		switch r.Kind {
		case "require_approvals_to_merge":
			req.Approvals = maxThreshold(req.Approvals, *r.Value)
		case "require_default_reviewer_approvals_to_merge":
			req.DefaultReviewerApprovals = maxThreshold(req.DefaultReviewerApprovals, *r.Value)
		case "require_passing_builds_to_merge":
			req.SuccessfulBuilds = maxThreshold(req.SuccessfulBuilds, *r.Value)
		}
	}

	if flags.RequiredApprovals != nil {
		req.Approvals = ThresholdValue(*flags.RequiredApprovals)
	}
	if flags.RequiredDefaultReviewerApprovals != nil {
		req.DefaultReviewerApprovals = ThresholdValue(*flags.RequiredDefaultReviewerApprovals)
	}
	if flags.RequiredSuccessfulBuilds != nil {
		req.SuccessfulBuilds = ThresholdValue(*flags.RequiredSuccessfulBuilds)
	}

	req.IgnoreComments = flags.IgnoreComments
	req.IgnoreTasks = flags.IgnoreTasks
	req.IgnoreBuilds = flags.IgnoreBuilds
	if flags.IgnoreBuilds {
		req.SuccessfulBuilds = ThresholdNone()
	}

	return req
}

func maxThreshold(cur Threshold, v int) Threshold {
	if cur.State != ThresholdStateValue || v > cur.Value {
		return ThresholdValue(v)
	}
	return cur
}
