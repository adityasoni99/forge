package blueprint

import (
	"context"
	"fmt"
	"time"
)

const DefaultMaxIterations = 100

type Engine struct {
	graph         *Graph
	blueprintName string
	maxIterations int
	hooks         []EngineHook
}

func NewEngine(g *Graph, blueprintName string) *Engine {
	return &Engine{
		graph:         g,
		blueprintName: blueprintName,
		maxIterations: DefaultMaxIterations,
	}
}

func (e *Engine) SetMaxIterations(n int) {
	e.maxIterations = n
}

func (e *Engine) RegisterHook(hook EngineHook) {
	e.hooks = append(e.hooks, hook)
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

		node, ok := e.graph.GetNode(state.CurrentNode)
		if !ok {
			nfErr := fmt.Errorf("node %q not found", state.CurrentNode)
			_ = e.fireHooks(ctx, HookRunError, HookData{
				NodeID:   state.CurrentNode,
				RunState: state,
				Error:    nfErr,
			})
			return e.failRun(state, nfErr)
		}

		if err := e.fireHooks(ctx, HookPreNodeExec, HookData{
			NodeID:   state.CurrentNode,
			NodeType: node.Type(),
			RunState: state,
		}); err != nil {
			return e.failRun(state, err)
		}

		result, err := node.Execute(ctx, state)
		if err != nil {
			_ = e.fireHooks(ctx, HookRunError, HookData{
				NodeID:   state.CurrentNode,
				NodeType: node.Type(),
				RunState: state,
				Error:    err,
			})
			return e.failRun(state, fmt.Errorf("node %q error: %w", state.CurrentNode, err))
		}

		resCopy := result
		if err := e.fireHooks(ctx, HookPostNodeExec, HookData{
			NodeID:   state.CurrentNode,
			NodeType: node.Type(),
			RunState: state,
			Result:   &resCopy,
		}); err != nil {
			return e.failRun(state, err)
		}

		if node.Type() == NodeTypeGate {
			if err := e.fireHooks(ctx, HookGateEvaluated, HookData{
				NodeID:   state.CurrentNode,
				NodeType: node.Type(),
				RunState: state,
				Result:   &resCopy,
			}); err != nil {
				return e.failRun(state, err)
			}
		}

		if err := e.fireHooks(ctx, HookPreEdgeTraversal, HookData{
			NodeID:   state.CurrentNode,
			NodeType: node.Type(),
			RunState: state,
			Result:   &resCopy,
		}); err != nil {
			return e.failRun(state, err)
		}

		state.NodeResults[state.CurrentNode] = result
		state.CurrentNode = e.resolveNextNode(state.CurrentNode, node, result)
	}

	state.Status = NodeStatusPassed
	state.EndTime = time.Now()
	if err := e.fireHooks(ctx, HookRunComplete, HookData{RunState: state}); err != nil {
		return e.failRun(state, err)
	}
	return state, nil
}

func (e *Engine) resolveNextNode(currentID string, node Node, result NodeResult) string {
	if node.Type() == NodeTypeGate {
		condition := "pass"
		if result.Status == NodeStatusFailed {
			condition = "fail"
		}
		next := e.graph.NextNodes(currentID, condition)
		if len(next) > 0 {
			return next[0]
		}
		return ""
	}
	next := e.graph.NextNodes(currentID, "")
	if len(next) > 0 {
		return next[0]
	}
	return ""
}
