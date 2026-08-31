package status

const (
	ReadinessReady    = "READY"
	ReadinessNotReady = "NOT_READY"
	ReadinessUnknown  = "UNKNOWN"

	CheckOK      = "OK"
	CheckFail    = "FAIL"
	CheckUnknown = "UNKNOWN"
	CheckInfo    = "INFO"
)

type ThresholdState int

const (
	ThresholdStateNone    ThresholdState = iota // no requirement for this category
	ThresholdStateValue                         // numeric requirement
	ThresholdStateUnknown                       // could not evaluate requirement
)

type Threshold struct {
	State ThresholdState
	Value int
}

func ThresholdNone() Threshold { return Threshold{State: ThresholdStateNone} }
func ThresholdValue(v int) Threshold {
	return Threshold{State: ThresholdStateValue, Value: v}
}
func ThresholdUnknown() Threshold { return Threshold{State: ThresholdStateUnknown} }

type Snapshot struct {
	ID                        int
	Title                     string
	State                     string
	Draft                     bool
	DestinationBranch         string
	ApprovalCount             int
	DefaultReviewerApprovals  int
	UnresolvedComments        int
	OpenTasks                 int
	SuccessfulBuilds          int
	ConflictCount             int
	CommentsAvailable         bool
	TasksAvailable            bool
	BuildsAvailable           bool
	ConflictsAvailable        bool
	BranchRulesAvailable      bool
	DefaultReviewersAvailable bool
	Warnings                  []string
}

type Requirements struct {
	Approvals                Threshold
	DefaultReviewerApprovals Threshold
	SuccessfulBuilds         Threshold
	IgnoreComments           bool
	IgnoreTasks              bool
	IgnoreBuilds             bool
}

type Check struct {
	Name     string `json:"name"`
	Status   string `json:"status"`
	Observed *int   `json:"observed,omitempty"`
	Required *int   `json:"required,omitempty"`
	Message  string `json:"message"`
}

type Result struct {
	PullRequest struct {
		ID    int    `json:"id"`
		Title string `json:"title"`
		State string `json:"state"`
	} `json:"pull_request"`
	Readiness string   `json:"readiness"`
	Checks    []Check  `json:"checks"`
	Missing   []string `json:"missing"`
	Warnings  []string `json:"warnings"`
}
