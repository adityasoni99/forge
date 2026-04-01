package blueprint

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
)

// DeterministicNode runs blueprint-supplied commands via sh -c. Only use blueprint
// YAML from trusted sources; untrusted input can execute arbitrary shell commands.
type DeterministicNode struct {
	id          string
	command     string
	description string // YAML metadata; does not affect execution
	maxRetries  int    // YAML metadata; does not affect execution yet
}

func NewDeterministicNode(id, command string) *DeterministicNode {
	return &DeterministicNode{id: id, command: command}
}

func (n *DeterministicNode) ID() string     { return n.id }
func (n *DeterministicNode) Type() NodeType { return NodeTypeDeterministic }

func (n *DeterministicNode) IsConcurrencySafe() bool { return true }

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
	description string // YAML metadata; does not affect execution
	maxRetries  int    // YAML metadata; does not affect execution yet
}

func NewGateNode(id, checkNodeID string) *GateNode {
	return &GateNode{id: id, checkNodeID: checkNodeID}
}

func (n *GateNode) ID() string     { return n.id }
func (n *GateNode) Type() NodeType { return NodeTypeGate }

func (n *GateNode) IsConcurrencySafe() bool { return false }

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

type AgenticNode struct {
	id              string
	prompt          string
	config          map[string]interface{}
	executor        AgentExecutor
	concurrencySafe bool
}

func NewAgenticNode(id, prompt string, config map[string]interface{}, executor AgentExecutor) *AgenticNode {
	return &AgenticNode{id: id, prompt: prompt, config: config, executor: executor}
}

// SetConcurrencySafe marks whether this node may run in parallel with other nodes.
// Default is false (order-dependent or shared state).
func (n *AgenticNode) SetConcurrencySafe(safe bool) *AgenticNode {
	n.concurrencySafe = safe
	return n
}

func (n *AgenticNode) ID() string     { return n.id }
func (n *AgenticNode) Type() NodeType { return NodeTypeAgentic }

func (n *AgenticNode) IsConcurrencySafe() bool { return n.concurrencySafe }

func (n *AgenticNode) Execute(ctx context.Context, _ *RunState) (NodeResult, error) {
	output, err := n.executor.Execute(ctx, n.prompt, n.config)
	if err != nil {
		return NodeResult{
			Status: NodeStatusFailed,
			Error:  err.Error(),
		}, nil
	}
	return NodeResult{
		Status: NodeStatusPassed,
		Output: output,
	}, nil
}
