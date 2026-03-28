package blueprint

import (
	"context"
	"time"
)

type NodeType int

const (
	NodeTypeAgentic NodeType = iota
	NodeTypeDeterministic
	NodeTypeGate
)

func (nt NodeType) String() string {
	switch nt {
	case NodeTypeAgentic:
		return "agentic"
	case NodeTypeDeterministic:
		return "deterministic"
	case NodeTypeGate:
		return "gate"
	default:
		return "unknown"
	}
}

type NodeStatus int

const (
	NodeStatusPending NodeStatus = iota
	NodeStatusRunning
	NodeStatusPassed
	NodeStatusFailed
	NodeStatusSkipped
)

func (ns NodeStatus) String() string {
	switch ns {
	case NodeStatusPending:
		return "pending"
	case NodeStatusRunning:
		return "running"
	case NodeStatusPassed:
		return "passed"
	case NodeStatusFailed:
		return "failed"
	case NodeStatusSkipped:
		return "skipped"
	default:
		return "unknown"
	}
}

type NodeResult struct {
	Status NodeStatus
	Output string
	Error  string
}

type Node interface {
	ID() string
	Type() NodeType
	Execute(ctx context.Context, state *RunState) (NodeResult, error)
}

type Edge struct {
	From      string
	To        string
	Condition string // "pass", "fail", or "" (unconditional)
}

type RunState struct {
	BlueprintName string
	RunID         string
	Status        NodeStatus
	CurrentNode   string
	NodeResults   map[string]NodeResult
	Context       map[string]interface{}
	StartTime     time.Time
	EndTime       time.Time
}

func NewRunState(blueprintName, runID string) *RunState {
	return &RunState{
		BlueprintName: blueprintName,
		RunID:         runID,
		Status:        NodeStatusPending,
		NodeResults:   make(map[string]NodeResult),
		Context:       make(map[string]interface{}),
	}
}

// AgentExecutor is the interface Layer 2 will implement.
// For testing, use mockExecutor. In production, gRPC adapter calls the harness.
type AgentExecutor interface {
	Execute(ctx context.Context, prompt string, config map[string]interface{}) (string, error)
}
