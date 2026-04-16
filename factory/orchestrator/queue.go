package orchestrator

import (
	"context"
	"sync"
)

const defaultQueueBuffer = 100

// PipelineExecutor abstracts Pipeline.Execute for testability.
type PipelineExecutor interface {
	Execute(ctx context.Context, req RunRequest) (RunResult, error)
}

type queueItem struct {
	runID string
	req   RunRequest
}

// RunQueue provides bounded-concurrency pipeline execution.
type RunQueue struct {
	registry    *RunRegistry
	pipeline    PipelineExecutor
	maxParallel int
	items       chan queueItem
	done        map[string]chan struct{}
	mu          sync.Mutex
	wg          sync.WaitGroup
	stopped     chan struct{}
}

// NewRunQueue constructs a queue with at least one worker slot.
// maxParallel is the maximum number of pipeline executions that may run
// concurrently; values below 1 are treated as 1.
func NewRunQueue(registry *RunRegistry, pipeline PipelineExecutor, maxParallel int) *RunQueue {
	if maxParallel < 1 {
		maxParallel = 1
	}
	return &RunQueue{
		registry:    registry,
		pipeline:    pipeline,
		maxParallel: maxParallel,
		items:       make(chan queueItem, defaultQueueBuffer),
		done:        make(map[string]chan struct{}),
		stopped:     make(chan struct{}),
	}
}

// Enqueue adds a run request and returns its ID immediately.
// It blocks if the internal buffer is full until space is available.
func (q *RunQueue) Enqueue(req RunRequest) string {
	runID := "run-" + generateRunID()
	q.registry.Register(runID)

	q.mu.Lock()
	q.done[runID] = make(chan struct{})
	q.mu.Unlock()

	q.items <- queueItem{runID: runID, req: req}
	return runID
}

// Start consumes the queue with up to maxParallel workers. It must run for the
// lifetime of the queue. Enqueue blocks if the internal buffer is full.
// Cancelling ctx stops accepting new work; in-flight items complete using a
// background context so they are not interrupted by the cancellation.
func (q *RunQueue) Start(ctx context.Context) {
	defer close(q.stopped)
	sem := make(chan struct{}, q.maxParallel)
	for {
		select {
		case <-ctx.Done():
			return
		case item := <-q.items:
			sem <- struct{}{}
			q.wg.Add(1)
			go func(it queueItem) {
				defer func() {
					<-sem
					q.wg.Done()
				}()
				q.process(context.Background(), it)
			}(item)
		}
	}
}

// Wait blocks until the given run completes or ctx expires.
func (q *RunQueue) Wait(ctx context.Context, runID string) error {
	q.mu.Lock()
	ch, ok := q.done[runID]
	q.mu.Unlock()
	if !ok {
		return nil
	}
	select {
	case <-ch:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Shutdown waits for all in-flight pipeline executions to complete.
// It respects the provided context's deadline; if the deadline expires
// before all workers finish, it returns the context error. In that case,
// in-flight workers continue running in the background; the caller should
// consider exiting the process if a clean stop is required.
func (q *RunQueue) Shutdown(ctx context.Context) error {
	select {
	case <-q.stopped:
	case <-ctx.Done():
		return ctx.Err()
	}
	done := make(chan struct{})
	go func() {
		q.wg.Wait()
		close(done)
	}()
	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (q *RunQueue) process(ctx context.Context, item queueItem) {
	q.registry.Update(item.runID, RunResult{RunID: item.runID, Status: RunStatusRunning})

	result, err := q.pipeline.Execute(ctx, item.req)
	if err != nil {
		result = RunResult{RunID: item.runID, Status: RunStatusFailed, Error: err.Error()}
	}
	result.RunID = item.runID
	q.registry.Update(item.runID, result)

	q.mu.Lock()
	if ch, ok := q.done[item.runID]; ok {
		close(ch)
		delete(q.done, item.runID)
	}
	q.mu.Unlock()
}
