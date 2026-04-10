package orchestrator

import "time"

type RunStatus int

const (
	RunStatusPending RunStatus = iota
	RunStatusRunning
	RunStatusPassed
	RunStatusFailed
)

func (s RunStatus) String() string {
	switch s {
	case RunStatusPending:
		return "pending"
	case RunStatusRunning:
		return "running"
	case RunStatusPassed:
		return "passed"
	case RunStatusFailed:
		return "failed"
	default:
		return "unknown"
	}
}

type RunRequest struct {
	Task          string
	BlueprintName string
	BlueprintFile string
	RepoDir       string
	Adapter       string
	Image         string
	Env           map[string]string
	Timeout       time.Duration
	NoPR          bool
	BaseBranch    string
}

type RunEvent struct {
	Phase     string
	Timestamp time.Time
	Message   string
	Duration  time.Duration
}

type RunResult struct {
	RunID    string
	Status   RunStatus
	Branch   string
	PRURL    string
	Output   string
	Duration time.Duration
	Error    string
	Events   []RunEvent
}
