package blueprint

import (
	"context"
	"fmt"
	"strconv"
	"strings"
)

// EvalNode sends an evaluation prompt through AgentExecutor and parses a
// numeric score. If score >= threshold the node passes; otherwise it fails.
type EvalNode struct {
	id        string
	prompt    string
	criteria  []string
	threshold float64
	executor  AgentExecutor
}

func NewEvalNode(id, prompt string, criteria []string, threshold float64, executor AgentExecutor) *EvalNode {
	return &EvalNode{
		id:        id,
		prompt:    prompt,
		criteria:  criteria,
		threshold: threshold,
		executor:  executor,
	}
}

func (n *EvalNode) ID() string              { return n.id }
func (n *EvalNode) Type() NodeType          { return NodeTypeEval }
func (n *EvalNode) IsConcurrencySafe() bool { return false }

func (n *EvalNode) Execute(ctx context.Context, state *RunState) (NodeResult, error) {
	evalPrompt := n.BuildEvalPrompt(state)
	output, err := n.executor.Execute(ctx, evalPrompt, nil)
	if err != nil {
		return NodeResult{
			Status: NodeStatusFailed,
			Error:  err.Error(),
		}, nil
	}

	score, err := ParseEvalScore(output)
	if err != nil {
		return NodeResult{
			Status: NodeStatusFailed,
			Output: output,
			Error:  fmt.Sprintf("failed to parse eval score: %s", err),
		}, nil
	}

	summary := fmt.Sprintf("score=%.2f threshold=%.2f", score, n.threshold)
	if score >= n.threshold {
		return NodeResult{Status: NodeStatusPassed, Output: summary}, nil
	}
	return NodeResult{Status: NodeStatusFailed, Output: summary}, nil
}

// BuildEvalPrompt constructs the full evaluation prompt including criteria and
// scoring instructions.
func (n *EvalNode) BuildEvalPrompt(state *RunState) string {
	var b strings.Builder
	b.WriteString(n.prompt)
	if len(n.criteria) > 0 {
		b.WriteString("\n\nEvaluation criteria:\n")
		for i, c := range n.criteria {
			fmt.Fprintf(&b, "%d. %s\n", i+1, c)
		}
	}
	b.WriteString("\nRespond with ONLY a numeric score between 0.0 and 1.0.")
	return b.String()
}

// ParseEvalScore extracts a float64 score in [0.0, 1.0] from executor output.
func ParseEvalScore(output string) (float64, error) {
	for _, word := range strings.Fields(strings.TrimSpace(output)) {
		clean := strings.TrimRight(word, ".,;:!?)")
		score, err := strconv.ParseFloat(clean, 64)
		if err == nil && score >= 0 && score <= 1 {
			return score, nil
		}
	}
	return 0, fmt.Errorf("no valid score (0.0-1.0) found in output: %q", strings.TrimSpace(output))
}
