package models

import (
	"encoding/json"
	"testing"
)

func TestBranchRestrictionDecode(t *testing.T) {
	raw := `{
		"id": 1,
		"kind": "require_approvals_to_merge",
		"value": 2,
		"branch_match_kind": "glob",
		"pattern": "main",
		"type": "branchrestriction"
	}`
	var br BranchRestriction
	if err := json.Unmarshal([]byte(raw), &br); err != nil {
		t.Fatal(err)
	}
	if br.Kind != "require_approvals_to_merge" || br.Value == nil || *br.Value != 2 || br.Pattern != "main" {
		t.Fatalf("unexpected decode: %+v", br)
	}
}

func TestTaskAndStatusDecode(t *testing.T) {
	var task PullRequestTask
	if err := json.Unmarshal([]byte(`{"state":"UNRESOLVED","content":{"raw":"fix"}}`), &task); err != nil {
		t.Fatal(err)
	}
	if task.State != "UNRESOLVED" {
		t.Fatalf("got %q", task.State)
	}
	var st CommitStatus
	if err := json.Unmarshal([]byte(`{"state":"SUCCESSFUL","key":"ci","name":"build"}`), &st); err != nil {
		t.Fatal(err)
	}
	if st.State != "SUCCESSFUL" {
		t.Fatalf("got %q", st.State)
	}
}
