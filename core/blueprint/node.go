package blueprint

import (
	"context"
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
