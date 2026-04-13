package blueprint

import (
	"context"
	"errors"
	"testing"
)

func TestEvalNodePassesAboveThreshold(t *testing.T) {
	executor := &mockExecutor{output: "0.85"}
	node := NewEvalNode("quality-check", "Evaluate code quality", []string{"correctness", "style"}, 0.8, executor)

	if node.ID() != "quality-check" {
		t.Errorf("ID() = %q, want %q", node.ID(), "quality-check")
	}
	if node.Type() != NodeTypeEval {
		t.Errorf("Type() = %v, want %v", node.Type(), NodeTypeEval)
	}
	if node.IsConcurrencySafe() {
		t.Error("IsConcurrencySafe() = true, want false")
	}

	state := NewRunState("test", "run-1")
	result, err := node.Execute(context.Background(), state)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if result.Status != NodeStatusPassed {
		t.Errorf("Status = %v, want %v", result.Status, NodeStatusPassed)
	}
	if result.Output == "" {
		t.Error("Output should contain score info")
	}
}

func TestEvalNodeFailsBelowThreshold(t *testing.T) {
	executor := &mockExecutor{output: "0.65"}
	node := NewEvalNode("quality-check", "Evaluate code quality", []string{"correctness"}, 0.8, executor)

	state := NewRunState("test", "run-1")
	result, err := node.Execute(context.Background(), state)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if result.Status != NodeStatusFailed {
		t.Errorf("Status = %v, want %v", result.Status, NodeStatusFailed)
	}
}

func TestEvalNodeExactThreshold(t *testing.T) {
	executor := &mockExecutor{output: "0.80"}
	node := NewEvalNode("check", "Eval", nil, 0.8, executor)

	state := NewRunState("test", "run-1")
	result, err := node.Execute(context.Background(), state)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if result.Status != NodeStatusPassed {
		t.Errorf("Status = %v, want Passed (score == threshold)", result.Status)
	}
}

func TestEvalNodeUnparseableScore(t *testing.T) {
	executor := &mockExecutor{output: "The code looks good overall"}
	node := NewEvalNode("check", "Eval", nil, 0.5, executor)

	state := NewRunState("test", "run-1")
	result, err := node.Execute(context.Background(), state)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if result.Status != NodeStatusFailed {
		t.Errorf("Status = %v, want Failed for unparseable score", result.Status)
	}
	if result.Error == "" {
		t.Error("Error should explain parse failure")
	}
}

func TestEvalNodeExecutorError(t *testing.T) {
	executor := &mockExecutor{err: errors.New("timeout")}
	node := NewEvalNode("check", "Eval", nil, 0.5, executor)

	state := NewRunState("test", "run-1")
	result, err := node.Execute(context.Background(), state)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if result.Status != NodeStatusFailed {
		t.Errorf("Status = %v, want Failed", result.Status)
	}
	if result.Error == "" {
		t.Error("Error should contain executor error message")
	}
}

func TestEvalNodeScoreInLongerText(t *testing.T) {
	executor := &mockExecutor{output: "Based on analysis, score: 0.92 out of 1.0"}
	node := NewEvalNode("check", "Eval", []string{"quality"}, 0.9, executor)

	state := NewRunState("test", "run-1")
	result, err := node.Execute(context.Background(), state)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if result.Status != NodeStatusPassed {
		t.Errorf("Status = %v, want Passed (0.92 >= 0.9)", result.Status)
	}
}

func TestEvalNodeBuildEvalPrompt(t *testing.T) {
	node := NewEvalNode("check", "Rate the code", []string{"correctness", "style"}, 0.8, nil)
	prompt := node.BuildEvalPrompt(NewRunState("bp", "r1"))
	if prompt == "" {
		t.Fatal("prompt should not be empty")
	}
	if !containsSubstring(prompt, "Rate the code") {
		t.Error("prompt should contain base prompt")
	}
	if !containsSubstring(prompt, "correctness") {
		t.Error("prompt should contain criteria")
	}
	if !containsSubstring(prompt, "0.0 and 1.0") {
		t.Error("prompt should instruct score format")
	}
}

func TestParseEvalScore(t *testing.T) {
	tests := []struct {
		input   string
		want    float64
		wantErr bool
	}{
		{"0.85", 0.85, false},
		{"  0.90  ", 0.90, false},
		{"Score: 0.75", 0.75, false},
		{"The answer is 0.60 based on criteria", 0.60, false},
		{"no numbers here", 0, true},
		{"1.5", 0, true},
		{"-0.3", 0, true},
	}
	for _, tt := range tests {
		got, err := ParseEvalScore(tt.input)
		if (err != nil) != tt.wantErr {
			t.Errorf("ParseEvalScore(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			continue
		}
		if !tt.wantErr && got != tt.want {
			t.Errorf("ParseEvalScore(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

func containsSubstring(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(s) > 0 && containsHelper(s, sub))
}

func containsHelper(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
