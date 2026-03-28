package blueprint

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
)

type DeterministicNode struct {
	id      string
	command string
}

func NewDeterministicNode(id, command string) *DeterministicNode {
	return &DeterministicNode{id: id, command: command}
}

func (n *DeterministicNode) ID() string     { return n.id }
func (n *DeterministicNode) Type() NodeType { return NodeTypeDeterministic }

func (n *DeterministicNode) Execute(ctx context.Context, _ *RunState) (NodeResult, error) {
	cmd := exec.CommandContext(ctx, "sh", "-c", n.command)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return NodeResult{
			Status: NodeStatusFailed,
			Output: strings.TrimSpace(string(output)),
			Error:  err.Error(),
		}, nil
	}
	return NodeResult{
		Status: NodeStatusPassed,
		Output: strings.TrimSpace(string(output)),
	}, nil
}

type GateNode struct {
	id          string
	checkNodeID string
}

func NewGateNode(id, checkNodeID string) *GateNode {
	return &GateNode{id: id, checkNodeID: checkNodeID}
}

func (n *GateNode) ID() string     { return n.id }
func (n *GateNode) Type() NodeType { return NodeTypeGate }

func (n *GateNode) Execute(_ context.Context, state *RunState) (NodeResult, error) {
	prev, ok := state.NodeResults[n.checkNodeID]
	if !ok {
		return NodeResult{
			Status: NodeStatusFailed,
			Error:  fmt.Sprintf("gate %q: checked node %q has no result", n.id, n.checkNodeID),
		}, nil
	}
	if prev.Status == NodeStatusPassed {
		return NodeResult{Status: NodeStatusPassed}, nil
	}
	return NodeResult{Status: NodeStatusFailed}, nil
}
