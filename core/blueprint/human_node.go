package blueprint

import (
	"context"
	"fmt"
	"time"
)

// ApprovalHandler is the interface for requesting human approval during
// blueprint execution. Layer 2 or CLI implementations provide the concrete
// handler; HeadlessApprovalHandler is the default for non-interactive runs.
type ApprovalHandler interface {
	RequestApproval(ctx context.Context, nodeID string, prompt string, state *RunState) (ApprovalResult, error)
}

type ApprovalResult struct {
	Approved bool
	Response string
}

type HumanNode struct {
	id       string
	prompt   string
	timeout  time.Duration
	handler  ApprovalHandler
	headless bool
}

func NewHumanNode(id, prompt string, timeout time.Duration, handler ApprovalHandler, headless bool) *HumanNode {
	return &HumanNode{
		id:       id,
		prompt:   prompt,
		timeout:  timeout,
		handler:  handler,
		headless: headless,
	}
}

func (n *HumanNode) ID() string              { return n.id }
func (n *HumanNode) Type() NodeType           { return NodeTypeHuman }
func (n *HumanNode) IsConcurrencySafe() bool  { return false }

func (n *HumanNode) Execute(ctx context.Context, state *RunState) (NodeResult, error) {
	if n.headless {
		return NodeResult{
			Status: NodeStatusFailed,
			Error:  "human approval required but running in headless mode",
		}, nil
	}

	if n.handler == nil {
		return NodeResult{
			Status: NodeStatusFailed,
			Error:  "no approval handler configured",
		}, nil
	}

	execCtx := ctx
	if n.timeout > 0 {
		var cancel context.CancelFunc
		execCtx, cancel = context.WithTimeout(ctx, n.timeout)
		defer cancel()
	}

	result, err := n.handler.RequestApproval(execCtx, n.id, n.prompt, state)
	if err != nil {
		return NodeResult{
			Status: NodeStatusFailed,
			Error:  fmt.Sprintf("approval error: %v", err),
		}, nil
	}

	if result.Approved {
		return NodeResult{Status: NodeStatusPassed, Output: result.Response}, nil
	}
	return NodeResult{Status: NodeStatusFailed, Output: result.Response, Error: "approval denied"}, nil
}

// HeadlessApprovalHandler auto-denies every approval request. Used as the
// default handler when no interactive handler is configured.
type HeadlessApprovalHandler struct{}

func (HeadlessApprovalHandler) RequestApproval(_ context.Context, _ string, _ string, _ *RunState) (ApprovalResult, error) {
	return ApprovalResult{Approved: false, Response: "auto-denied (headless)"}, nil
}
