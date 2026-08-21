package cmd

import (
	"fmt"
	"strconv"

	"github.com/spf13/cobra"
	"github.com/zoldyzdk/bb-cli/internal/api"
	"github.com/zoldyzdk/bb-cli/internal/config"
	"github.com/zoldyzdk/bb-cli/internal/models"
	"github.com/zoldyzdk/bb-cli/internal/status"
)

var (
	requiredApprovals                int
	requiredDefaultReviewerApprovals int
	requiredSuccessfulBuilds         int
	ignoreComments                   bool
	ignoreTasks                      bool
	ignoreBuilds                     bool
	failOnNotReady                   bool
	statusJSON                       bool
)

var prStatusCmd = &cobra.Command{
	Use:   "status <pr-id>",
	Short: "Show merge readiness of a pull request",
	Args:  cobra.ExactArgs(1),
	RunE:  runPRStatus,
}

func init() {
	prCmd.AddCommand(prStatusCmd)
	prStatusCmd.Flags().IntVar(&requiredApprovals, "required-approvals", -1, "Required approvals (overrides branch rules)")
	prStatusCmd.Flags().IntVar(&requiredDefaultReviewerApprovals, "required-default-reviewer-approvals", -1, "Required default reviewer approvals (overrides branch rules)")
	prStatusCmd.Flags().IntVar(&requiredSuccessfulBuilds, "required-successful-builds", -1, "Required successful builds (overrides branch rules)")
	prStatusCmd.Flags().BoolVar(&ignoreComments, "ignore-comments", false, "Ignore unresolved comments")
	prStatusCmd.Flags().BoolVar(&ignoreTasks, "ignore-tasks", false, "Ignore open tasks")
	prStatusCmd.Flags().BoolVar(&ignoreBuilds, "ignore-builds", false, "Ignore build statuses")
	prStatusCmd.Flags().BoolVar(&failOnNotReady, "fail-on-not-ready", false, "Exit with error if the pull request is not ready to merge")
	prStatusCmd.Flags().BoolVar(&statusJSON, "json", false, "Output as JSON")
}

func runPRStatus(cmd *cobra.Command, args []string) error {
	prID, err := strconv.Atoi(args[0])
	if err != nil {
		return fmt.Errorf("invalid PR ID: %s", args[0])
	}

	workspace, repo, err := resolveWorkspaceAndRepo()
	if err != nil {
		return err
	}

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}
	if !cfg.HasCredentials() {
		return fmt.Errorf("not logged in. Run 'bb auth login' first")
	}

	client := api.NewClient(cfg.Username, cfg.Token)
	pr, err := client.GetPullRequest(workspace, repo, prID)
	if err != nil {
		return fmt.Errorf("failed to get pull request: %w", err)
	}

	var warnings []string
	const pageLimit = 100

	comments, err := client.ListPullRequestComments(workspace, repo, prID, pageLimit)
	commentsOK := err == nil
	if err != nil {
		warnings = append(warnings, fmt.Sprintf("comments unavailable: %v", err))
		comments = nil
	}

	tasks, err := client.ListPullRequestTasks(workspace, repo, prID, pageLimit)
	tasksOK := err == nil
	if err != nil {
		warnings = append(warnings, fmt.Sprintf("tasks unavailable: %v", err))
		tasks = nil
	}

	statuses, err := client.ListPullRequestStatuses(workspace, repo, prID, pageLimit)
	buildsOK := err == nil
	if err != nil {
		warnings = append(warnings, fmt.Sprintf("builds unavailable: %v", err))
		statuses = nil
	}

	conflicts, err := client.ListPullRequestConflicts(workspace, repo, prID)
	conflictsOK := err == nil
	if err != nil {
		warnings = append(warnings, fmt.Sprintf("conflicts unavailable: %v", err))
		conflicts = nil
	}

	rules, err := client.ListBranchRestrictions(workspace, repo, pageLimit)
	rulesOK := err == nil
	if err != nil {
		if api.IsForbidden(err) || api.IsUnauthorized(err) {
			warnings = append(warnings, "branch rules unavailable: insufficient permissions")
		} else {
			warnings = append(warnings, fmt.Sprintf("branch rules unavailable: %v", err))
		}
		rules = nil
	}

	defaultReviewers, err := client.ListDefaultReviewers(workspace, repo, pageLimit)
	defaultReviewersOK := err == nil
	if err != nil {
		defaultReviewers = nil
		if rulesOK {
			warnings = append(warnings, fmt.Sprintf("default reviewers unavailable: %v", err))
		}
	}

	snap := buildSnapshot(pr, comments, commentsOK, tasks, tasksOK, statuses, buildsOK, conflicts, conflictsOK, defaultReviewers, defaultReviewersOK, rulesOK, warnings)
	overrides := buildOverrides(rulesOK)
	req := status.InferRequirements(rules, pr.Destination.Branch.Name, overrides)
	result := status.Evaluate(snap, req)

	if statusJSON {
		b, err := status.FormatJSON(result)
		if err != nil {
			return err
		}
		fmt.Println(string(b))
	} else {
		fmt.Print(status.FormatHuman(result))
	}

	if failOnNotReady && result.Readiness == status.ReadinessNotReady {
		return fmt.Errorf("pull request is not ready to merge")
	}
	return nil
}

func countApprovals(participants []models.Participant) int {
	n := 0
	for _, p := range participants {
		if p.Approved {
			n++
		}
	}
	return n
}

func countDefaultReviewerApprovals(participants []models.Participant, defaultReviewers []models.User) int {
	uuids := make(map[string]struct{}, len(defaultReviewers))
	for _, u := range defaultReviewers {
		uuids[u.UUID] = struct{}{}
	}
	n := 0
	for _, p := range participants {
		if p.Approved {
			if _, ok := uuids[p.User.UUID]; ok {
				n++
			}
		}
	}
	return n
}

func countUnresolved(comments []models.Comment) int {
	n := 0
	for _, c := range comments {
		if !c.Deleted && c.Resolution == nil && c.Parent == nil {
			n++
		}
	}
	return n
}

func countOpenTasks(tasks []models.PullRequestTask) int {
	n := 0
	for _, t := range tasks {
		if t.State != "RESOLVED" {
			n++
		}
	}
	return n
}

func countSuccessfulBuilds(statuses []models.CommitStatus) int {
	n := 0
	for _, s := range statuses {
		if s.State == "SUCCESSFUL" {
			n++
		}
	}
	return n
}

func buildOverrides(rulesOK bool) status.OverrideFlags {
	o := status.OverrideFlags{
		BranchRulesUnavailable: !rulesOK,
		IgnoreComments:         ignoreComments,
		IgnoreTasks:            ignoreTasks,
		IgnoreBuilds:           ignoreBuilds,
	}
	if requiredApprovals >= 0 {
		v := requiredApprovals
		o.RequiredApprovals = &v
	}
	if requiredDefaultReviewerApprovals >= 0 {
		v := requiredDefaultReviewerApprovals
		o.RequiredDefaultReviewerApprovals = &v
	}
	if requiredSuccessfulBuilds >= 0 {
		v := requiredSuccessfulBuilds
		o.RequiredSuccessfulBuilds = &v
	}
	return o
}

func buildSnapshot(
	pr *models.PullRequest,
	comments []models.Comment,
	commentsOK bool,
	tasks []models.PullRequestTask,
	tasksOK bool,
	statuses []models.CommitStatus,
	buildsOK bool,
	conflicts []models.FileConflict,
	conflictsOK bool,
	defaultReviewers []models.User,
	defaultReviewersOK bool,
	rulesOK bool,
	warnings []string,
) status.Snapshot {
	return status.Snapshot{
		ID:                        pr.ID,
		Title:                     pr.Title,
		State:                     pr.State,
		Draft:                     pr.Draft,
		DestinationBranch:         pr.Destination.Branch.Name,
		ApprovalCount:             countApprovals(pr.Participants),
		DefaultReviewerApprovals:  countDefaultReviewerApprovals(pr.Participants, defaultReviewers),
		UnresolvedComments:        countUnresolved(comments),
		OpenTasks:                 countOpenTasks(tasks),
		SuccessfulBuilds:          countSuccessfulBuilds(statuses),
		ConflictCount:             len(conflicts),
		CommentsAvailable:         commentsOK,
		TasksAvailable:            tasksOK,
		BuildsAvailable:           buildsOK,
		ConflictsAvailable:        conflictsOK,
		BranchRulesAvailable:      rulesOK,
		DefaultReviewersAvailable: defaultReviewersOK,
		Warnings:                  warnings,
	}
}
