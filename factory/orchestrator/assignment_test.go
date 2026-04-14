package orchestrator

import "testing"

func TestTaskAssignerReviewTask(t *testing.T) {
	a := NewTaskAssigner()
	adapter := a.Assign(RunRequest{Task: "Review the auth module"})
	if adapter != "claude" {
		t.Errorf("Assign() = %q, want %q", adapter, "claude")
	}
}

func TestTaskAssignerImplementTask(t *testing.T) {
	a := NewTaskAssigner()
	adapter := a.Assign(RunRequest{Task: "Implement user registration"})
	if adapter != "claude" {
		t.Errorf("Assign() = %q, want %q", adapter, "claude")
	}
}

func TestTaskAssignerExplicitAdapter(t *testing.T) {
	a := NewTaskAssigner()
	adapter := a.Assign(RunRequest{Task: "do stuff", Adapter: "echo"})
	if adapter != "echo" {
		t.Errorf("Assign() = %q, want %q for explicit adapter", adapter, "echo")
	}
}

func TestTaskAssignerDefaultAdapter(t *testing.T) {
	a := NewTaskAssigner()
	adapter := a.Assign(RunRequest{Task: "something generic"})
	if adapter != "claude" {
		t.Errorf("Assign() = %q, want %q as default", adapter, "claude")
	}
}

func TestTaskAssignerEmptyTask(t *testing.T) {
	a := NewTaskAssigner()
	adapter := a.Assign(RunRequest{})
	if adapter != "claude" {
		t.Errorf("Assign() = %q, want %q for empty task", adapter, "claude")
	}
}
