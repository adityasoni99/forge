package blueprint

import (
	"context"
	"path/filepath"
)

// PermissionRule maps a node-ID glob pattern to a decision.
// The first matching rule wins during the deterministic phase.
type PermissionRule struct {
	Pattern  string
	Decision PermissionDecision
}

// PermissionPipeline implements PermissionChecker with a two-phase design:
//  1. Deterministic: iterate rules, first glob match wins.
//  2. Async (only when decision is PermissionAsk): delegate to ApprovalHandler.
//     In headless mode or when no handler is configured, Ask becomes Deny.
type PermissionPipeline struct {
	rules    []PermissionRule
	handler  ApprovalHandler
	headless bool
}

func NewPermissionPipeline(rules []PermissionRule, handler ApprovalHandler, headless bool) *PermissionPipeline {
	return &PermissionPipeline{rules: rules, handler: handler, headless: headless}
}

func (pp *PermissionPipeline) Check(ctx context.Context, node Node, state *RunState) (PermissionDecision, error) {
	decision := pp.deterministicCheck(node.ID())

	switch decision {
	case PermissionAllow, PermissionDeny:
		return decision, nil
	case PermissionAsk:
		return pp.asyncCheck(ctx, node, state)
	default:
		return PermissionAllow, nil
	}
}

func (pp *PermissionPipeline) deterministicCheck(nodeID string) PermissionDecision {
	for _, rule := range pp.rules {
		matched, _ := filepath.Match(rule.Pattern, nodeID)
		if matched {
			return rule.Decision
		}
	}
	return PermissionAllow
}

func (pp *PermissionPipeline) asyncCheck(ctx context.Context, node Node, state *RunState) (PermissionDecision, error) {
	if pp.headless {
		return PermissionDeny, nil
	}
	if pp.handler == nil {
		return PermissionDeny, nil
	}

	result, err := pp.handler.RequestApproval(ctx, node.ID(), "Permission check: approve execution?", state)
	if err != nil {
		return PermissionDeny, err
	}
	if result.Approved {
		return PermissionAllow, nil
	}
	return PermissionDeny, nil
}
