package blueprint

import "testing"

func TestNodeTypeString(t *testing.T) {
	tests := []struct {
		nt   NodeType
		want string
	}{
		{NodeTypeAgentic, "agentic"},
		{NodeTypeDeterministic, "deterministic"},
		{NodeTypeGate, "gate"},
	}
	for _, tt := range tests {
		if got := tt.nt.String(); got != tt.want {
			t.Errorf("NodeType(%d).String() = %q, want %q", tt.nt, got, tt.want)
		}
	}
	if got := NodeType(99).String(); got != "unknown" {
		t.Errorf("NodeType(99).String() = %q, want %q", got, "unknown")
	}
}

func TestNodeStatusString(t *testing.T) {
	tests := []struct {
		ns   NodeStatus
		want string
	}{
		{NodeStatusPending, "pending"},
		{NodeStatusRunning, "running"},
		{NodeStatusPassed, "passed"},
		{NodeStatusFailed, "failed"},
	}
	for _, tt := range tests {
		if got := tt.ns.String(); got != tt.want {
			t.Errorf("NodeStatus(%d).String() = %q, want %q", tt.ns, got, tt.want)
		}
	}
	if got := NodeStatus(99).String(); got != "unknown" {
		t.Errorf("NodeStatus(99).String() = %q, want %q", got, "unknown")
	}
}

func TestNewRunState(t *testing.T) {
	rs := NewRunState("test-bp", "run-123")
	if rs.BlueprintName != "test-bp" {
		t.Errorf("BlueprintName = %q, want %q", rs.BlueprintName, "test-bp")
	}
	if rs.RunID != "run-123" {
		t.Errorf("RunID = %q, want %q", rs.RunID, "run-123")
	}
	if rs.Status != NodeStatusPending {
		t.Errorf("Status = %v, want %v", rs.Status, NodeStatusPending)
	}
	if rs.NodeResults == nil {
		t.Error("NodeResults should be initialized")
	}
	if rs.Context == nil {
		t.Error("Context should be initialized")
	}
}
