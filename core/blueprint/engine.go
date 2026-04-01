package blueprint

import (
	"context"
	"fmt"
	"time"
)

const DefaultMaxIterations = 100

// DefaultMaxConcurrency caps concurrent node goroutines when multiple next nodes are safe.
const DefaultMaxConcurrency = 10

type Engine struct {
	graph             *Graph
	blueprintName     string
	maxIterations     int
	maxConcurrency    int
	hooks             []EngineHook
	permissionChecker PermissionChecker // nil means allow-all
	headless          bool              // when true, PermissionAsk is treated as deny
}

func NewEngine(g *Graph, blueprintName string) *Engine {
	return &Engine{
		graph:          g,
		blueprintName:  blueprintName,
		maxIterations:  DefaultMaxIterations,
		maxConcurrency: DefaultMaxConcurrency,
	}
}

func (e *Engine) SetMaxIterations(n int) {
	e.maxIterations = n
}

// SetMaxConcurrency sets the errgroup limit for parallel node execution (minimum 1).
func (e *Engine) SetMaxConcurrency(n int) {
	if n < 1 {
		n = 1
	}
	e.maxConcurrency = n
}

func (e *Engine) RegisterHook(hook EngineHook) {
	e.hooks = append(e.hooks, hook)
}

// SetPermissionChecker sets the checker run before each node executes. Nil allows all nodes.
func (e *Engine) SetPermissionChecker(pc PermissionChecker) {
	e.permissionChecker = pc
}

// SetHeadless configures whether PermissionAsk is denied (true) or allowed pending future UI (false).
func (e *Engine) SetHeadless(v bool) {
	e.headless = v
}

func (e *Engine) checkPermission(ctx context.Context, node Node, state *RunState) error {
	if e.permissionChecker == nil {
		return nil
	}
	decision, err := e.permissionChecker.Check(ctx, node, state)
	if err != nil {
		return err
	}
	nodeID := node.ID()
	switch decision {
	case PermissionAllow:
		return nil
	case PermissionDeny:
		return fmt.Errorf("permission denied for node %q", nodeID)
	case PermissionAsk:
		if e.headless {
			return fmt.Errorf("permission denied (headless mode) for node %q", nodeID)
		}
		return nil
	default:
		return fmt.Errorf("unknown permission decision for node %q", nodeID)
	}
}

func (e *Engine) fireHooks(ctx context.Context, event HookEvent, data HookData) error {
	for _, h := range e.hooks {
		res := h.OnEvent(ctx, event, data)
		if !res.Continue {
			if res.Error != nil {
				return res.Error
			}
			return errHookAborted
		}
	}
	return nil
}

func (e *Engine) failRun(state *RunState, err error) (*RunState, error) {
	state.Status = NodeStatusFailed
	state.EndTime = time.Now()
	return state, err
}

func (e *Engine) Execute(ctx context.Context) (*RunState, error) {
	if err := e.graph.Validate(); err != nil {
		return nil, fmt.Errorf("invalid graph: %w", err)
	}

	state := NewRunState(e.blueprintName, fmt.Sprintf("run-%d", time.Now().UnixNano()))
	state.Status = NodeStatusRunning
	state.StartTime = time.Now()
	state.CurrentNode = e.graph.StartNode()

	if err := e.fireHooks(ctx, HookRunStart, HookData{RunState: state}); err != nil {
		return e.failRun(state, err)
	}

	iterations := 0
	for state.CurrentNode != "" {
		if err := ctx.Err(); err != nil {
			_ = e.fireHooks(ctx, HookRunError, HookData{
				NodeID:   state.CurrentNode,
				RunState: state,
				Error:    err,
			})
			return e.failRun(state, fmt.Errorf("context cancelled: %w", err))
		}
		if iterations >= e.maxIterations {
			loopErr := fmt.Errorf("exceeded max iterations (%d)", e.maxIterations)
			_ = e.fireHooks(ctx, HookRunError, HookData{
				NodeID:   state.CurrentNode,
				RunState: state,
				Error:    loopErr,
			})
			return e.failRun(state, loopErr)
		}
		iterations++

		nodeID := state.CurrentNode
		node, ok := e.graph.GetNode(nodeID)
		if !ok {
			nfErr := fmt.Errorf("node %q not found", nodeID)
			_ = e.fireHooks(ctx, HookRunError, HookData{
				NodeID:   nodeID,
				RunState: state,
				Error:    nfErr,
			})
			return e.failRun(state, nfErr)
		}

		result, err := e.runNodeWithHooks(ctx, state, nodeID, node)
		if err != nil {
			return e.failRun(state, err)
		}

		state.NodeResults[nodeID] = result

		next := e.resolveNextNodes(nodeID, node, result)
		if len(next) == 0 {
			state.CurrentNode = ""
			continue
		}
		if len(next) == 1 || !e.allConcurrencySafe(next) {
			state.CurrentNode = next[0]
			continue
		}

		if iterations+len(next) > e.maxIterations {
			loopErr := fmt.Errorf("exceeded max iterations (%d)", e.maxIterations)
			_ = e.fireHooks(ctx, HookRunError, HookData{
				NodeID:   nodeID,
				RunState: state,
				Error:    loopErr,
			})
			return e.failRun(state, loopErr)
		}
		iterations += len(next)

		merged, err := e.runParallelFanOut(ctx, state, next)
		if err != nil {
			return e.failRun(state, err)
		}
		state.CurrentNode = merged
	}

	state.Status = NodeStatusPassed
	state.EndTime = time.Now()
	if err := e.fireHooks(ctx, HookRunComplete, HookData{RunState: state}); err != nil {
		return e.failRun(state, err)
	}
	return state, nil
}

func (e *Engine) resolveNextNodes(currentID string, node Node, result NodeResult) []string {
	if node.Type() == NodeTypeGate {
		condition := "pass"
		if result.Status == NodeStatusFailed {
			condition = "fail"
		}
		return e.graph.NextNodes(currentID, condition)
	}
	return e.graph.NextNodes(currentID, "")
}

func (e *Engine) allConcurrencySafe(ids []string) bool {
	for _, id := range ids {
		n, ok := e.graph.GetNode(id)
		if !ok || !n.IsConcurrencySafe() {
			return false
		}
	}
	return true
}

func (e *Engine) runNodeWithHooks(ctx context.Context, state *RunState, nodeID string, node Node) (NodeResult, error) {
	if err := e.fireHooks(ctx, HookPreNodeExec, HookData{
		NodeID:   nodeID,
		NodeType: node.Type(),
		RunState: state,
	}); err != nil {
		return NodeResult{}, err
	}

	if err := e.checkPermission(ctx, node, state); err != nil {
		return NodeResult{}, err
	}

	result, err := node.Execute(ctx, state)
	if err != nil {
		_ = e.fireHooks(ctx, HookRunError, HookData{
			NodeID:   nodeID,
			NodeType: node.Type(),
			RunState: state,
			Error:    err,
		})
		return NodeResult{}, fmt.Errorf("node %q error: %w", nodeID, err)
	}

	resCopy := result
	if err := e.fireHooks(ctx, HookPostNodeExec, HookData{
		NodeID:   nodeID,
		NodeType: node.Type(),
		RunState: state,
		Result:   &resCopy,
	}); err != nil {
		return NodeResult{}, err
	}

	if node.Type() == NodeTypeGate {
		if err := e.fireHooks(ctx, HookGateEvaluated, HookData{
			NodeID:   nodeID,
			NodeType: node.Type(),
			RunState: state,
			Result:   &resCopy,
		}); err != nil {
			return NodeResult{}, err
		}
	}

	if err := e.fireHooks(ctx, HookPreEdgeTraversal, HookData{
		NodeID:   nodeID,
		NodeType: node.Type(),
		RunState: state,
		Result:   &resCopy,
	}); err != nil {
		return NodeResult{}, err
	}

	return result, nil
}
