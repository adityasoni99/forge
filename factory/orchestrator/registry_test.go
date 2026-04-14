package orchestrator

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

func TestRunRegistryRegisterAndGet(t *testing.T) {
	reg := NewRunRegistry()
	reg.Register("run-1")

	result, ok := reg.Get("run-1")
	if !ok {
		t.Fatal("expected run-1 to exist")
	}
	if result.RunID != "run-1" {
		t.Errorf("RunID = %q, want %q", result.RunID, "run-1")
	}
	if result.Status != RunStatusPending {
		t.Errorf("Status = %v, want Pending", result.Status)
	}
}

func TestRunRegistryGetUnknown(t *testing.T) {
	reg := NewRunRegistry()
	_, ok := reg.Get("nonexistent")
	if ok {
		t.Error("expected false for unknown run ID")
	}
}

func TestRunRegistryUpdate(t *testing.T) {
	reg := NewRunRegistry()
	reg.Register("run-1")

	updated := RunResult{
		RunID:    "run-1",
		Status:   RunStatusPassed,
		Output:   "done",
		Duration: 5 * time.Second,
	}
	reg.Update("run-1", updated)

	result, ok := reg.Get("run-1")
	if !ok {
		t.Fatal("expected run-1 to exist")
	}
	if result.Status != RunStatusPassed {
		t.Errorf("Status = %v, want Passed", result.Status)
	}
	if result.Output != "done" {
		t.Errorf("Output = %q, want %q", result.Output, "done")
	}
}

func TestRunRegistryUpdateUnknown(t *testing.T) {
	reg := NewRunRegistry()
	reg.Update("ghost", RunResult{RunID: "ghost", Status: RunStatusFailed})
	_, ok := reg.Get("ghost")
	if ok {
		t.Error("update on unknown ID should not create entry")
	}
}

func TestRunRegistryList(t *testing.T) {
	reg := NewRunRegistry()
	reg.Register("run-1")
	reg.Register("run-2")

	all := reg.List()
	if len(all) != 2 {
		t.Errorf("List() len = %d, want 2", len(all))
	}
}

func TestRunRegistryConcurrent(t *testing.T) {
	reg := NewRunRegistry()
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			id := fmt.Sprintf("run-%d", n)
			reg.Register(id)
			reg.Get(id)
			reg.Update(id, RunResult{RunID: id, Status: RunStatusPassed})
			reg.List()
		}(i)
	}
	wg.Wait()
	if len(reg.List()) != 50 {
		t.Errorf("List() len = %d, want 50", len(reg.List()))
	}
}
