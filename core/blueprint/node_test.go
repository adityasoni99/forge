package blueprint

import (
	"context"
	"testing"
	"time"
)

func TestDeterministicNodeExecuteSuccess(t *testing.T) {
	node := NewDeterministicNode("lint", "echo hello")
	state := NewRunState("test", "run-1")
	result, err := node.Execute(context.Background(), state)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if result.Status != NodeStatusPassed {
		t.Errorf("Status = %v, want Passed", result.Status)
	}
	if result.Output == "" {
		t.Error("expected non-empty output")
	}
}

func TestDeterministicNodeExecuteFailure(t *testing.T) {
	node := NewDeterministicNode("bad", "exit 1")
	state := NewRunState("test", "run-1")
	result, err := node.Execute(context.Background(), state)
	if err != nil {
		t.Fatalf("Execute() should not return Go error, got %v", err)
	}
	if result.Status != NodeStatusFailed {
		t.Errorf("Status = %v, want Failed", result.Status)
	}
}

func TestDeterministicNodeTimeout(t *testing.T) {
	node := NewDeterministicNode("slow", "sleep 10")
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	result, err := node.Execute(ctx, NewRunState("test", "run-1"))
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if result.Status != NodeStatusFailed {
		t.Errorf("Status = %v, want Failed (timeout)", result.Status)
	}
}

func TestDeterministicNodeIDAndType(t *testing.T) {
	node := NewDeterministicNode("lint", "echo ok")
	if node.ID() != "lint" {
		t.Errorf("ID() = %q, want %q", node.ID(), "lint")
	}
	if node.Type() != NodeTypeDeterministic {
		t.Errorf("Type() = %v, want Deterministic", node.Type())
	}
}

func TestGateNodePass(t *testing.T) {
	gate := NewGateNode("lint-gate", "lint")
	state := NewRunState("test", "run-1")
	state.NodeResults["lint"] = NodeResult{Status: NodeStatusPassed}
	result, err := gate.Execute(context.Background(), state)
	if err != nil {
		t.Fatalf("error = %v", err)
	}
	if result.Status != NodeStatusPassed {
		t.Errorf("Status = %v, want Passed", result.Status)
	}
}

func TestGateNodeFail(t *testing.T) {
	gate := NewGateNode("lint-gate", "lint")
	state := NewRunState("test", "run-1")
	state.NodeResults["lint"] = NodeResult{Status: NodeStatusFailed}
	result, err := gate.Execute(context.Background(), state)
	if err != nil {
		t.Fatalf("error = %v", err)
	}
	if result.Status != NodeStatusFailed {
		t.Errorf("Status = %v, want Failed", result.Status)
	}
}

func TestGateNodeMissingCheckNode(t *testing.T) {
	gate := NewGateNode("gate", "nonexistent")
	state := NewRunState("test", "run-1")
	result, err := gate.Execute(context.Background(), state)
	if err != nil {
		t.Fatalf("error = %v", err)
	}
	if result.Status != NodeStatusFailed {
		t.Errorf("Status = %v, want Failed", result.Status)
	}
	if result.Error == "" {
		t.Error("expected error message for missing check node")
	}
}

func TestGateNodeIDAndType(t *testing.T) {
	gate := NewGateNode("g", "x")
	if gate.ID() != "g" {
		t.Errorf("ID() = %q, want %q", gate.ID(), "g")
	}
	if gate.Type() != NodeTypeGate {
		t.Errorf("Type() = %v, want Gate", gate.Type())
	}
}
