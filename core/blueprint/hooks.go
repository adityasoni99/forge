package blueprint

import (
	"context"
	"errors"
	"fmt"
)

// HookEvent identifies when a hook fires during engine execution.
type HookEvent int

const (
	HookRunStart HookEvent = iota
	HookPreNodeExec
	HookPostNodeExec
	HookPreEdgeTraversal
	HookGateEvaluated
	HookRunComplete
	HookRunError
)

func (e HookEvent) String() string {
	switch e {
	case HookRunStart:
		return "HookRunStart"
	case HookPreNodeExec:
		return "HookPreNodeExec"
	case HookPostNodeExec:
		return "HookPostNodeExec"
	case HookPreEdgeTraversal:
		return "HookPreEdgeTraversal"
	case HookGateEvaluated:
		return "HookGateEvaluated"
	case HookRunComplete:
		return "HookRunComplete"
	case HookRunError:
		return "HookRunError"
	default:
		return fmt.Sprintf("HookEvent(%d)", int(e))
	}
}

// HookData carries context about the event.
type HookData struct {
	NodeID   string
	NodeType NodeType
	RunState *RunState
	Result   *NodeResult // nil for pre-exec events
	NextNode string      // for edge traversal
	Error    error       // for error events
}

// HookResult tells the engine how to proceed.
type HookResult struct {
	Continue      bool
	ModifyInput   map[string]interface{}
	InjectContext string
	Error         error
}

// DefaultHookResult returns a result that continues execution.
func DefaultHookResult() HookResult {
	return HookResult{Continue: true}
}

// EngineHook is called at specific points during engine execution.
type EngineHook interface {
	OnEvent(ctx context.Context, event HookEvent, data HookData) HookResult
}

var errHookAborted = errors.New("hook aborted")
