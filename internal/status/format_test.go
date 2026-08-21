package status

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestFormatHumanNotReady(t *testing.T) {
	res := Evaluate(func() Snapshot {
		s := baseSnapshot()
		s.ApprovalCount = 1
		s.UnresolvedComments = 3
		return s
	}(), Requirements{Approvals: ThresholdValue(2)})
	out := FormatHuman(res)
	if !strings.Contains(out, "Merge readiness: NOT READY") {
		t.Fatal(out)
	}
	if !strings.Contains(out, "[FAIL]") || !strings.Contains(out, "Missing:") {
		t.Fatal(out)
	}
}

func TestFormatJSONStable(t *testing.T) {
	res := Evaluate(baseSnapshot(), Requirements{Approvals: ThresholdValue(2), SuccessfulBuilds: ThresholdValue(1), DefaultReviewerApprovals: ThresholdValue(1)})
	b, err := FormatJSON(res)
	if err != nil {
		t.Fatal(err)
	}
	var decoded Result
	if err := json.Unmarshal(b, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.Readiness != ReadinessReady {
		t.Fatal(decoded.Readiness)
	}
}

func TestFormatHumanReady(t *testing.T) {
	res := Evaluate(baseSnapshot(), Requirements{
		Approvals:                ThresholdValue(2),
		SuccessfulBuilds:         ThresholdValue(1),
		DefaultReviewerApprovals: ThresholdValue(1),
	})
	out := FormatHuman(res)
	if !strings.Contains(out, "PR #123 Add search feature") {
		t.Fatal(out)
	}
	if !strings.Contains(out, "Merge readiness: READY") {
		t.Fatal(out)
	}
	if strings.Contains(out, "Missing:") {
		t.Fatal(out)
	}
}

func TestFormatHumanUnknown(t *testing.T) {
	res := Evaluate(baseSnapshot(), Requirements{Approvals: ThresholdUnknown(), SuccessfulBuilds: ThresholdNone()})
	out := FormatHuman(res)
	if !strings.Contains(out, "Merge readiness: UNKNOWN") {
		t.Fatal(out)
	}
	if !strings.Contains(out, "[UNKNOWN]") {
		t.Fatal(out)
	}
}

func TestFormatHumanWarnings(t *testing.T) {
	snap := baseSnapshot()
	snap.Warnings = []string{"branch restriction lookup failed"}
	res := Evaluate(snap, Requirements{Approvals: ThresholdValue(2)})
	out := FormatHuman(res)
	if !strings.Contains(out, "Warnings:") {
		t.Fatal(out)
	}
	if !strings.Contains(out, "branch restriction lookup failed") {
		t.Fatal(out)
	}
}

func TestFormatJSONEmptySlices(t *testing.T) {
	res := Evaluate(baseSnapshot(), Requirements{Approvals: ThresholdValue(2)})
	b, err := FormatJSON(res)
	if err != nil {
		t.Fatal(err)
	}
	s := string(b)
	if !strings.Contains(s, `"missing": []`) {
		t.Fatal(s)
	}
	if !strings.Contains(s, `"warnings": []`) {
		t.Fatal(s)
	}
	if strings.Contains(s, `"missing": null`) || strings.Contains(s, `"warnings": null`) {
		t.Fatal(s)
	}
}
