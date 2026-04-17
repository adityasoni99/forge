package blueprint

import (
	"context"
	"testing"
	"time"
)

type mockApprovalHandler struct {
	approved bool
	response string
	err      error
}

func (m *mockApprovalHandler) RequestApproval(_ context.Context, _ string, _ string, _ *RunState) (ApprovalResult, error) {
	if m.err != nil {
		return ApprovalResult{}, m.err
	}
	return ApprovalResult{Approved: m.approved, Response: m.response}, nil
}

func TestHumanNodeApproved(t *testing.T) {
	node := NewHumanNode("approve", "Please approve this PR", 0, &mockApprovalHandler{approved: true, response: "LGTM"}, false)

	if node.ID() != "approve" {
		t.Errorf("ID() = %q, want %q", node.ID(), "approve")
	}
	if node.Type() != NodeTypeHuman {
		t.Errorf("Type() = %v, want NodeTypeHuman", node.Type())
	}

	state := NewRunState("test", "run-1")
	result, err := node.Execute(context.Background(), state)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if result.Status != NodeStatusPassed {
		t.Errorf("Status = %v, want Passed", result.Status)
	}
	if result.Output != "LGTM" {
		t.Errorf("Output = %q, want %q", result.Output, "LGTM")
	}
}

func TestHumanNodeDenied(t *testing.T) {
	node := NewHumanNode("approve", "Review this", 0, &mockApprovalHandler{approved: false, response: "Needs fixes"}, false)
	state := NewRunState("test", "run-1")
	result, err := node.Execute(context.Background(), state)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if result.Status != NodeStatusFailed {
		t.Errorf("Status = %v, want Failed", result.Status)
	}
}

func TestHumanNodeHeadlessAutoDenies(t *testing.T) {
	node := NewHumanNode("approve", "Review this", 0, &mockApprovalHandler{approved: true}, true)
	state := NewRunState("test", "run-1")
	result, err := node.Execute(context.Background(), state)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if result.Status != NodeStatusFailed {
		t.Errorf("Status = %v, want Failed (headless auto-deny)", result.Status)
	}
	if result.Error == "" {
		t.Error("expected error message about headless mode")
	}
}

func TestHumanNodeTimeout(t *testing.T) {
	slow := &mockApprovalHandler{}
	slow.err = context.DeadlineExceeded

	node := NewHumanNode("approve", "Review this", 100*time.Millisecond, slow, false)
	state := NewRunState("test", "run-1")
	result, err := node.Execute(context.Background(), state)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if result.Status != NodeStatusFailed {
		t.Errorf("Status = %v, want Failed (timeout)", result.Status)
	}
}

func TestHumanNodeNotConcurrencySafe(t *testing.T) {
	node := NewHumanNode("approve", "Review", 0, &mockApprovalHandler{}, false)
	if node.IsConcurrencySafe() {
		t.Error("HumanNode should not be concurrency-safe")
	}
}
