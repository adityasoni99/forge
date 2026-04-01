package blueprint

import "context"

// PermissionDecision represents the outcome of a permission check.
type PermissionDecision int

const (
	PermissionAllow PermissionDecision = iota
	PermissionDeny
	PermissionAsk
)

func (pd PermissionDecision) String() string {
	switch pd {
	case PermissionAllow:
		return "allow"
	case PermissionDeny:
		return "deny"
	case PermissionAsk:
		return "ask"
	default:
		return "unknown"
	}
}

// PermissionChecker evaluates whether a node is permitted to execute.
type PermissionChecker interface {
	Check(ctx context.Context, node Node, state *RunState) (PermissionDecision, error)
}

// TrustedSourceChecker always allows execution (current default behavior).
type TrustedSourceChecker struct{}

func (TrustedSourceChecker) Check(context.Context, Node, *RunState) (PermissionDecision, error) {
	return PermissionAllow, nil
}
