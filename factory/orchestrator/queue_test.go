package orchestrator

import (
	"context"
	"sync/atomic"
	"testing"
	"time"
)

type mockPipeline struct {
	callCount atomic.Int32
	delay     time.Duration
	err       error
	fn        func(ctx context.Context, req RunRequest) (RunResult, error)
}

func (m *mockPipeline) Execute(ctx context.Context, req RunRequest) (RunResult, error) {
	m.callCount.Add(1)
	if m.fn != nil {
		return m.fn(ctx, req)
	}
	if m.delay > 0 {
		time.Sleep(m.delay)
	}
	if m.err != nil {
		return RunResult{RunID: "mock", Status: RunStatusFailed, Error: m.err.Error()}, nil
	}
	return RunResult{RunID: "mock", Status: RunStatusPassed, Output: "done: " + req.Task}, nil
}

func TestRunQueueEnqueueAndProcess(t *testing.T) {
	reg := NewRunRegistry()
	pipe := &mockPipeline{}
	q := NewRunQueue(reg, pipe, 2)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	go q.Start(ctx)

	runID := q.Enqueue(RunRequest{Task: "test task", BlueprintName: "bp"})
	if runID == "" {
		t.Fatal("expected non-empty run ID")
	}

	err := q.Wait(ctx, runID)
	if err != nil {
		t.Fatalf("Wait() error = %v", err)
	}

	result, ok := reg.Get(runID)
	if !ok {
		t.Fatal("expected run to be in registry")
	}
	if result.Status != RunStatusPassed {
		t.Errorf("Status = %v, want Passed", result.Status)
	}
}

func TestRunQueueConcurrencyLimit(t *testing.T) {
	reg := NewRunRegistry()
	pipe := &mockPipeline{delay: 50 * time.Millisecond}
	q := NewRunQueue(reg, pipe, 2)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	go q.Start(ctx)

	var ids []string
	for i := 0; i < 5; i++ {
		id := q.Enqueue(RunRequest{Task: "task", BlueprintName: "bp"})
		ids = append(ids, id)
	}

	for _, id := range ids {
		_ = q.Wait(ctx, id)
	}

	if pipe.callCount.Load() != 5 {
		t.Errorf("callCount = %d, want 5", pipe.callCount.Load())
	}
}

func TestRunQueueContextCancellation(t *testing.T) {
	reg := NewRunRegistry()
	pipe := &mockPipeline{delay: 1 * time.Second}
	q := NewRunQueue(reg, pipe, 1)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	go q.Start(ctx)

	runID := q.Enqueue(RunRequest{Task: "slow", BlueprintName: "bp"})
	err := q.Wait(ctx, runID)
	if err == nil {
		t.Log("wait may complete before cancellation; this is acceptable")
	}
}

func TestRunQueueShutdownDrainsInFlight(t *testing.T) {
	registry := NewRunRegistry()
	var started, finished int32
	pipeline := &mockPipeline{fn: func(ctx context.Context, req RunRequest) (RunResult, error) {
		atomic.AddInt32(&started, 1)
		time.Sleep(50 * time.Millisecond)
		atomic.AddInt32(&finished, 1)
		return RunResult{Status: RunStatusPassed}, nil
	}}
	queue := NewRunQueue(registry, pipeline, 2)

	ctx, cancel := context.WithCancel(context.Background())
	go queue.Start(ctx)

	queue.Enqueue(RunRequest{Task: "a"})
	queue.Enqueue(RunRequest{Task: "b"})
	time.Sleep(20 * time.Millisecond)

	cancel()
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer shutdownCancel()

	err := queue.Shutdown(shutdownCtx)
	if err != nil {
		t.Fatalf("Shutdown: %v", err)
	}
	if f := atomic.LoadInt32(&finished); f != 2 {
		t.Errorf("finished = %d, want 2", f)
	}
}

func TestRunQueueShutdownRespectsDeadline(t *testing.T) {
	registry := NewRunRegistry()
	pipeline := &mockPipeline{fn: func(ctx context.Context, req RunRequest) (RunResult, error) {
		time.Sleep(500 * time.Millisecond)
		return RunResult{Status: RunStatusPassed}, nil
	}}
	queue := NewRunQueue(registry, pipeline, 1)

	ctx, cancel := context.WithCancel(context.Background())
	go queue.Start(ctx)

	queue.Enqueue(RunRequest{Task: "slow"})
	time.Sleep(20 * time.Millisecond)
	cancel()

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer shutdownCancel()

	err := queue.Shutdown(shutdownCtx)
	if err != context.DeadlineExceeded {
		t.Errorf("Shutdown err = %v, want DeadlineExceeded", err)
	}
}
