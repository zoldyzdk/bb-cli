package status

import "fmt"

func intPtr(v int) *int { return &v }

func Evaluate(snapshot Snapshot, requirements Requirements) Result {
	var res Result
	res.PullRequest.ID = snapshot.ID
	res.PullRequest.Title = snapshot.Title
	res.PullRequest.State = snapshot.State
	res.Missing = []string{}
	res.Warnings = []string{}
	if snapshot.Warnings != nil {
		res.Warnings = snapshot.Warnings
	}

	res.Checks = append(res.Checks, evaluateState(snapshot))
	res.Checks = append(res.Checks, evaluateDraft(snapshot))
	res.Checks = append(res.Checks, evaluateThreshold(
		"approvals",
		snapshot.ApprovalCount,
		requirements.Approvals,
		func(need int) string { return fmt.Sprintf("%d more approval(s) required", need) },
		"approval requirement unknown",
	))
	if c, ok := evaluateDefaultReviewerApprovals(snapshot, requirements); ok {
		res.Checks = append(res.Checks, c)
	}
	res.Checks = append(res.Checks, evaluateComments(snapshot, requirements))
	res.Checks = append(res.Checks, evaluateTasks(snapshot, requirements))
	res.Checks = append(res.Checks, evaluateBuilds(snapshot, requirements))
	res.Checks = append(res.Checks, evaluateConflicts(snapshot))

	hasFail := false
	hasUnknown := false
	for _, c := range res.Checks {
		switch c.Status {
		case CheckFail:
			hasFail = true
			res.Missing = append(res.Missing, c.Message)
		case CheckUnknown:
			hasUnknown = true
		}
	}
	switch {
	case hasFail:
		res.Readiness = ReadinessNotReady
	case hasUnknown:
		res.Readiness = ReadinessUnknown
	default:
		res.Readiness = ReadinessReady
	}
	return res
}

func evaluateState(snap Snapshot) Check {
	if snap.State == "OPEN" {
		return Check{Name: "state", Status: CheckOK, Message: snap.State}
	}
	return Check{Name: "state", Status: CheckFail, Message: "PR is not open"}
}

func evaluateDraft(snap Snapshot) Check {
	if !snap.Draft {
		return Check{Name: "draft", Status: CheckOK, Message: "not a draft"}
	}
	return Check{Name: "draft", Status: CheckFail, Message: "PR is a draft"}
}

func evaluateDefaultReviewerApprovals(snap Snapshot, req Requirements) (Check, bool) {
	th := req.DefaultReviewerApprovals
	if th.State == ThresholdStateNone && snap.DefaultReviewerApprovals == 0 {
		return Check{}, false
	}
	return evaluateThreshold(
		"default_reviewer_approvals",
		snap.DefaultReviewerApprovals,
		th,
		func(need int) string {
			return fmt.Sprintf("%d more default reviewer approval(s) required", need)
		},
		"default reviewer approval requirement unknown",
	), true
}

func evaluateComments(snap Snapshot, req Requirements) Check {
	return evaluateGate(
		"comments",
		snap.UnresolvedComments,
		snap.CommentsAvailable,
		req.IgnoreComments,
		func(n int) string { return fmt.Sprintf("Resolve %d open comments", n) },
		"comment data unavailable",
	)
}

func evaluateTasks(snap Snapshot, req Requirements) Check {
	return evaluateGate(
		"tasks",
		snap.OpenTasks,
		snap.TasksAvailable,
		req.IgnoreTasks,
		func(n int) string { return fmt.Sprintf("Resolve %d open tasks", n) },
		"task data unavailable",
	)
}

func evaluateBuilds(snap Snapshot, req Requirements) Check {
	th := req.SuccessfulBuilds
	if th.State == ThresholdStateNone {
		return Check{
			Name:     "builds",
			Status:   CheckInfo,
			Observed: intPtr(snap.SuccessfulBuilds),
			Message:  fmt.Sprintf("%d successful", snap.SuccessfulBuilds),
		}
	}
	if !snap.BuildsAvailable {
		return Check{
			Name:    "builds",
			Status:  CheckUnknown,
			Message: "build statuses unavailable",
		}
	}
	return evaluateThreshold(
		"builds",
		snap.SuccessfulBuilds,
		th,
		func(need int) string { return fmt.Sprintf("%d more successful build(s) required", need) },
		"build requirement unknown",
	)
}

func evaluateConflicts(snap Snapshot) Check {
	if !snap.ConflictsAvailable {
		return Check{Name: "conflicts", Status: CheckUnknown, Message: "conflict data unavailable"}
	}
	if snap.ConflictCount > 0 {
		return Check{
			Name:     "conflicts",
			Status:   CheckFail,
			Observed: intPtr(snap.ConflictCount),
			Message:  "conflicts present",
		}
	}
	return Check{
		Name:     "conflicts",
		Status:   CheckOK,
		Observed: intPtr(0),
		Message:  "none",
	}
}

func evaluateThreshold(name string, observed int, th Threshold, failMsg func(need int) string, unknownMsg string) Check {
	c := Check{Name: name, Observed: intPtr(observed)}
	switch th.State {
	case ThresholdStateNone:
		c.Status = CheckInfo
		c.Message = fmt.Sprintf("%d observed", observed)
	case ThresholdStateUnknown:
		c.Status = CheckUnknown
		c.Message = unknownMsg
	default:
		c.Required = intPtr(th.Value)
		if observed >= th.Value {
			c.Status = CheckOK
			c.Message = fmt.Sprintf("%d/%d", observed, th.Value)
		} else {
			c.Status = CheckFail
			c.Message = failMsg(th.Value - observed)
		}
	}
	return c
}

func evaluateGate(name string, count int, available, ignore bool, failMsg func(int) string, unavailableMsg string) Check {
	c := Check{Name: name, Observed: intPtr(count)}
	if ignore {
		c.Status = CheckInfo
		c.Message = fmt.Sprintf("%d (ignored)", count)
		return c
	}
	if !available {
		c.Status = CheckUnknown
		c.Observed = nil
		c.Message = unavailableMsg
		return c
	}
	if count > 0 {
		c.Status = CheckFail
		c.Message = failMsg(count)
		return c
	}
	c.Status = CheckOK
	c.Message = "none"
	return c
}
