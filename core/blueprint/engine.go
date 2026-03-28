package blueprint

import (
	"context"
	"fmt"
	"time"
)

const DefaultMaxIterations = 100

type Engine struct {
	graph         *Graph
	maxIterations int
}

func NewEngine(g *Graph) *Engine {
	return &Engine{graph: g, maxIterations: DefaultMaxIterations}
}

func (e *Engine) SetMaxIterations(n int) {
	e.maxIterations = n
}

func (e *Engine) Execute(ctx context.Context) (*RunState, error) {
	if err := e.graph.Validate(); err != nil {
		return nil, fmt.Errorf("invalid graph: %w", err)
	}

	state := NewRunState("", fmt.Sprintf("run-%d", time.Now().UnixNano()))
	state.Status = NodeStatusRunning
	state.StartTime = time.Now()
	state.CurrentNode = e.graph.StartNode()

	iterations := 0
	for state.CurrentNode != "" {
		if err := ctx.Err(); err != nil {
			state.Status = NodeStatusFailed
			state.EndTime = time.Now()
			return state, fmt.Errorf("context cancelled: %w", err)
		}
		if iterations >= e.maxIterations {
			state.Status = NodeStatusFailed
			state.EndTime = time.Now()
			return state, fmt.Errorf("exceeded max iterations (%d)", e.maxIterations)
		}
		iterations++

		node, ok := e.graph.GetNode(state.CurrentNode)
		if !ok {
			state.Status = NodeStatusFailed
			state.EndTime = time.Now()
			return state, fmt.Errorf("node %q not found", state.CurrentNode)
		}

		result, err := node.Execute(ctx, state)
		if err != nil {
			state.Status = NodeStatusFailed
			state.EndTime = time.Now()
			return state, fmt.Errorf("node %q error: %w", state.CurrentNode, err)
		}

		state.NodeResults[state.CurrentNode] = result
		state.CurrentNode = e.resolveNextNode(state.CurrentNode, node, result)
	}

	state.Status = NodeStatusPassed
	state.EndTime = time.Now()
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
