package status

import (
	"encoding/json"
	"fmt"
	"strings"
)

const checkTagWidth = len("[UNKNOWN]")

var checkLabels = map[string]string{
	"state":                      "State",
	"draft":                      "Draft",
	"approvals":                  "Approvals",
	"default_reviewer_approvals": "Default reviewer approvals",
	"comments":                   "Comments",
	"tasks":                      "Tasks",
	"builds":                     "Builds",
	"conflicts":                  "Conflicts",
}

func FormatHuman(result Result) string {
	var b strings.Builder
	fmt.Fprintf(&b, "PR #%d %s\n\n", result.PullRequest.ID, result.PullRequest.Title)
	fmt.Fprintf(&b, "Merge readiness: %s\n\n", strings.ReplaceAll(result.Readiness, "_", " "))
	for _, c := range result.Checks {
		fmt.Fprintf(&b, "%s %s: %s\n", formatCheckTag(c.Status), checkLabel(c.Name), checkDetail(c))
	}
	if len(result.Missing) > 0 {
		b.WriteString("\nMissing:\n")
		for _, item := range result.Missing {
			fmt.Fprintf(&b, "- %s\n", item)
		}
	}
	if len(result.Warnings) > 0 {
		b.WriteString("\nWarnings:\n")
		for _, w := range result.Warnings {
			fmt.Fprintf(&b, "- %s\n", w)
		}
	}
	return b.String()
}

func FormatJSON(result Result) ([]byte, error) {
	return json.MarshalIndent(result, "", "  ")
}

func formatCheckTag(status string) string {
	return fmt.Sprintf("%-*s", checkTagWidth, "["+status+"]")
}

func checkLabel(name string) string {
	if label, ok := checkLabels[name]; ok {
		return label
	}
	return name
}

func checkDetail(c Check) string {
	if c.Observed != nil && c.Required != nil {
		return fmt.Sprintf("%d/%d", *c.Observed, *c.Required)
	}
	if c.Observed != nil && c.Status != CheckInfo {
		switch c.Name {
		case "comments":
			return fmt.Sprintf("%d unresolved", *c.Observed)
		case "tasks":
			return fmt.Sprintf("%d open", *c.Observed)
		}
	}
	return c.Message
}
