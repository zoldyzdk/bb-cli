package models

import (
	"encoding/json"
	"testing"
)

func TestCreatePullRequestBodyBranchEndpointsOmitCommit(t *testing.T) {
	body := CreatePullRequestBody{
		Title: "title",
		Source: CreatePullRequestEndpoint{
			Branch: Branch{Name: "chore/concata-571-mulesoft-al2023"},
		},
		Destination: &CreatePullRequestEndpoint{
			Branch: Branch{Name: "master"},
		},
	}

	data, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}

	var payload map[string]any
	if err := json.Unmarshal(data, &payload); err != nil {
		t.Fatal(err)
	}

	source, ok := payload["source"].(map[string]any)
	if !ok {
		t.Fatalf("source missing: %s", data)
	}
	destination, ok := payload["destination"].(map[string]any)
	if !ok {
		t.Fatalf("destination missing: %s", data)
	}
	if _, ok := source["commit"]; ok {
		t.Fatalf("source commit sent: %s", data)
	}
	if _, ok := destination["commit"]; ok {
		t.Fatalf("destination commit sent: %s", data)
	}

	sourceBranch, _ := source["branch"].(map[string]any)
	destinationBranch, _ := destination["branch"].(map[string]any)
	if sourceBranch["name"] != "chore/concata-571-mulesoft-al2023" || destinationBranch["name"] != "master" {
		t.Fatalf("branch names: %s", data)
	}
}

func TestCreatePullRequestBodyOmitsDestination(t *testing.T) {
	body := CreatePullRequestBody{
		Title: "title",
		Source: CreatePullRequestEndpoint{
			Branch: Branch{Name: "feature"},
		},
	}

	data, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}

	var payload map[string]any
	if err := json.Unmarshal(data, &payload); err != nil {
		t.Fatal(err)
	}
	if _, ok := payload["destination"]; ok {
		t.Fatalf("destination sent: %s", data)
	}
}
