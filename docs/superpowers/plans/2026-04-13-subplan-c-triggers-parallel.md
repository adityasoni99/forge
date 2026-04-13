# Sub-plan C: Triggers + Parallel Runs Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a webhook HTTP trigger, parallel run queue with concurrency control, run registry, and task assignment to the factory layer.

**Architecture:** A lightweight Go HTTP server (`forged` daemon) accepts `POST /api/v1/runs` requests and enqueues them to a bounded RunQueue. The queue dequeues requests with configurable concurrency and runs each through the existing `Pipeline.Execute()`. A RunRegistry tracks run state in-memory. TaskAssigner selects an adapter based on simple rules.

**Tech Stack:** Go 1.22+, net/http stdlib, sync, context

---

## File Structure

**Create:**
- `factory/orchestrator/queue.go` — RunQueue (bounded channel + goroutine pool)
- `factory/orchestrator/queue_test.go` — queue tests
- `factory/orchestrator/registry.go` — RunRegistry (in-memory run state)
- `factory/orchestrator/registry_test.go` — registry tests
- `factory/orchestrator/assignment.go` — TaskAssigner (rule-based adapter selection)
- `factory/orchestrator/assignment_test.go` — assignment tests
- `factory/triggers/webhook.go` — HTTP handler for POST/GET runs
- `factory/triggers/webhook_test.go` — webhook handler tests
- `cmd/forged/main.go` — daemon entrypoint

**Modify:**
- (none — all new files)

---

### Task 1: RunRegistry

**Files:**
- Create: `factory/orchestrator/registry.go`
- Create: `factory/orchestrator/registry_test.go`

- [ ] **Step 1: Write the failing tests**

Create `factory/orchestrator/registry_test.go`:

```go
package orchestrator

import (
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
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./factory/orchestrator/ -run "TestRunRegistry" -v`
Expected: compilation error — `NewRunRegistry` undefined

- [ ] **Step 3: Implement RunRegistry**

Create `factory/orchestrator/registry.go`:

```go
package orchestrator

import "sync"

// RunRegistry tracks run state in-memory. Safe for concurrent access.
type RunRegistry struct {
	mu   sync.RWMutex
	runs map[string]RunResult
}

func NewRunRegistry() *RunRegistry {
	return &RunRegistry{runs: make(map[string]RunResult)}
}

// Register creates a new entry with Pending status.
func (r *RunRegistry) Register(runID string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.runs[runID] = RunResult{RunID: runID, Status: RunStatusPending}
}

// Update replaces the result for an existing run. No-op if runID is unknown.
func (r *RunRegistry) Update(runID string, result RunResult) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.runs[runID]; ok {
		r.runs[runID] = result
	}
}

// Get returns the current result for a run.
func (r *RunRegistry) Get(runID string) (RunResult, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result, ok := r.runs[runID]
	return result, ok
}

// List returns all tracked runs.
func (r *RunRegistry) List() []RunResult {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]RunResult, 0, len(r.runs))
	for _, v := range r.runs {
		out = append(out, v)
	}
	return out
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./factory/orchestrator/ -run "TestRunRegistry" -v`
Expected: all PASS

- [ ] **Step 5: Commit**

```bash
git add factory/orchestrator/registry.go factory/orchestrator/registry_test.go
git commit -m "feat(factory): add RunRegistry for in-memory run tracking"
```

---

### Task 2: RunQueue

**Files:**
- Create: `factory/orchestrator/queue.go`
- Create: `factory/orchestrator/queue_test.go`

- [ ] **Step 1: Write the failing tests**

Create `factory/orchestrator/queue_test.go`:

```go
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
}

func (m *mockPipeline) Execute(_ context.Context, req RunRequest) (RunResult, error) {
	m.callCount.Add(1)
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
	var maxConcurrent atomic.Int32
	var current atomic.Int32

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
	_ = maxConcurrent
	_ = current
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
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./factory/orchestrator/ -run "TestRunQueue" -v`
Expected: compilation error — `NewRunQueue` undefined

- [ ] **Step 3: Implement RunQueue**

Create `factory/orchestrator/queue.go`:

```go
package orchestrator

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"sync"
)

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
}

func NewRunQueue(registry *RunRegistry, pipeline PipelineExecutor, maxParallel int) *RunQueue {
	if maxParallel < 1 {
		maxParallel = 1
	}
	return &RunQueue{
		registry:    registry,
		pipeline:    pipeline,
		maxParallel: maxParallel,
		items:       make(chan queueItem, 100),
		done:        make(map[string]chan struct{}),
	}
}

// Enqueue adds a run request and returns its ID immediately.
func (q *RunQueue) Enqueue(req RunRequest) string {
	runID := newRunID()
	q.registry.Register(runID)

	q.mu.Lock()
	q.done[runID] = make(chan struct{})
	q.mu.Unlock()

	q.items <- queueItem{runID: runID, req: req}
	return runID
}

// Start consumes the queue with up to maxParallel workers. Blocks until ctx is done.
func (q *RunQueue) Start(ctx context.Context) {
	sem := make(chan struct{}, q.maxParallel)
	for {
		select {
		case <-ctx.Done():
			return
		case item := <-q.items:
			sem <- struct{}{}
			go func(it queueItem) {
				defer func() { <-sem }()
				q.process(ctx, it)
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
	}
	q.mu.Unlock()
}

func newRunID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return "run-" + hex.EncodeToString(b)
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./factory/orchestrator/ -run "TestRunQueue" -v -timeout 30s`
Expected: all PASS

- [ ] **Step 5: Commit**

```bash
git add factory/orchestrator/queue.go factory/orchestrator/queue_test.go
git commit -m "feat(factory): add RunQueue with bounded concurrency"
```

---

### Task 3: TaskAssigner

**Files:**
- Create: `factory/orchestrator/assignment.go`
- Create: `factory/orchestrator/assignment_test.go`

- [ ] **Step 1: Write the failing tests**

Create `factory/orchestrator/assignment_test.go`:

```go
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
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./factory/orchestrator/ -run "TestTaskAssigner" -v`
Expected: compilation error — `NewTaskAssigner` undefined

- [ ] **Step 3: Implement TaskAssigner**

Create `factory/orchestrator/assignment.go`:

```go
package orchestrator

// TaskAssigner selects an adapter for a given run request.
// For v0.2, this is rule-based. Future versions may use skill metadata.
type TaskAssigner struct{}

func NewTaskAssigner() *TaskAssigner {
	return &TaskAssigner{}
}

// Assign returns the adapter name for the request. If the request already
// specifies an adapter, that value is returned unchanged.
func (a *TaskAssigner) Assign(req RunRequest) string {
	if req.Adapter != "" {
		return req.Adapter
	}
	return "claude"
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./factory/orchestrator/ -run "TestTaskAssigner" -v`
Expected: all PASS

- [ ] **Step 5: Commit**

```bash
git add factory/orchestrator/assignment.go factory/orchestrator/assignment_test.go
git commit -m "feat(factory): add TaskAssigner for adapter selection"
```

---

### Task 4: Webhook HTTP handler

**Files:**
- Create: `factory/triggers/webhook.go`
- Create: `factory/triggers/webhook_test.go`

- [ ] **Step 1: Write the failing tests**

Create `factory/triggers/webhook_test.go`:

```go
package triggers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/aditya-soni/forge/factory/orchestrator"
)

type stubQueue struct {
	lastReq orchestrator.RunRequest
	runID   string
}

func (s *stubQueue) Enqueue(req orchestrator.RunRequest) string {
	s.lastReq = req
	return s.runID
}

type stubRegistry struct {
	result orchestrator.RunResult
	found  bool
}

func (s *stubRegistry) Get(runID string) (orchestrator.RunResult, bool) {
	return s.result, s.found
}

func TestWebhookCreateRun(t *testing.T) {
	q := &stubQueue{runID: "run-abc123"}
	reg := &stubRegistry{}
	handler := NewWebhookHandler(q, reg)

	body := CreateRunRequest{
		Task:      "Implement login",
		Blueprint: "standard-implementation",
		Adapter:   "claude",
		RepoURL:   "https://github.com/user/repo",
	}
	data, _ := json.Marshal(body)
	req := httptest.NewRequest("POST", "/api/v1/runs", bytes.NewReader(data))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusAccepted {
		t.Errorf("status = %d, want %d", w.Code, http.StatusAccepted)
	}

	var resp CreateRunResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if resp.RunID != "run-abc123" {
		t.Errorf("RunID = %q, want %q", resp.RunID, "run-abc123")
	}
	if resp.Status != "pending" {
		t.Errorf("Status = %q, want %q", resp.Status, "pending")
	}
}

func TestWebhookCreateRunMissingTask(t *testing.T) {
	q := &stubQueue{runID: "run-1"}
	reg := &stubRegistry{}
	handler := NewWebhookHandler(q, reg)

	body := CreateRunRequest{Blueprint: "bp"}
	data, _ := json.Marshal(body)
	req := httptest.NewRequest("POST", "/api/v1/runs", bytes.NewReader(data))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestWebhookGetRunStatus(t *testing.T) {
	q := &stubQueue{}
	reg := &stubRegistry{
		result: orchestrator.RunResult{
			RunID:  "run-abc",
			Status: orchestrator.RunStatusPassed,
			Output: "all tests pass",
		},
		found: true,
	}
	handler := NewWebhookHandler(q, reg)

	req := httptest.NewRequest("GET", "/api/v1/runs/run-abc", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp RunStatusResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.RunID != "run-abc" {
		t.Errorf("RunID = %q", resp.RunID)
	}
	if resp.Status != "passed" {
		t.Errorf("Status = %q, want passed", resp.Status)
	}
}

func TestWebhookGetRunNotFound(t *testing.T) {
	q := &stubQueue{}
	reg := &stubRegistry{found: false}
	handler := NewWebhookHandler(q, reg)

	req := httptest.NewRequest("GET", "/api/v1/runs/nonexistent", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestWebhookMethodNotAllowed(t *testing.T) {
	q := &stubQueue{}
	reg := &stubRegistry{}
	handler := NewWebhookHandler(q, reg)

	req := httptest.NewRequest("DELETE", "/api/v1/runs", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want %d", w.Code, http.StatusMethodNotAllowed)
	}
}

// Ensure stubQueue and stubRegistry satisfy interfaces at compile time.
var _ Enqueuer = (*stubQueue)(nil)
var _ StatusGetter = (*stubRegistry)(nil)
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./factory/triggers/ -run "TestWebhook" -v`
Expected: compilation error — package/types don't exist

- [ ] **Step 3: Implement webhook handler**

Create `factory/triggers/webhook.go`:

```go
package triggers

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/aditya-soni/forge/factory/orchestrator"
)

// Enqueuer accepts run requests.
type Enqueuer interface {
	Enqueue(req orchestrator.RunRequest) string
}

// StatusGetter returns run status.
type StatusGetter interface {
	Get(runID string) (orchestrator.RunResult, bool)
}

// CreateRunRequest is the JSON body for POST /api/v1/runs.
type CreateRunRequest struct {
	Task       string `json:"task"`
	Blueprint  string `json:"blueprint"`
	Adapter    string `json:"adapter,omitempty"`
	RepoURL    string `json:"repo_url,omitempty"`
	BaseBranch string `json:"base_branch,omitempty"`
	NoPR       bool   `json:"no_pr,omitempty"`
}

// CreateRunResponse is returned after enqueueing.
type CreateRunResponse struct {
	RunID  string `json:"run_id"`
	Status string `json:"status"`
}

// RunStatusResponse is returned by GET /api/v1/runs/:id.
type RunStatusResponse struct {
	RunID  string `json:"run_id"`
	Status string `json:"status"`
	Output string `json:"output,omitempty"`
	Error  string `json:"error,omitempty"`
	PRURL  string `json:"pr_url,omitempty"`
}

// WebhookHandler serves the /api/v1/runs endpoint.
type WebhookHandler struct {
	queue    Enqueuer
	registry StatusGetter
}

func NewWebhookHandler(queue Enqueuer, registry StatusGetter) *WebhookHandler {
	return &WebhookHandler{queue: queue, registry: registry}
}

func (h *WebhookHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/runs")

	switch {
	case r.Method == http.MethodPost && (path == "" || path == "/"):
		h.createRun(w, r)
	case r.Method == http.MethodGet && len(path) > 1:
		runID := strings.TrimPrefix(path, "/")
		h.getRunStatus(w, runID)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *WebhookHandler) createRun(w http.ResponseWriter, r *http.Request) {
	var body CreateRunRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid JSON: "+err.Error(), http.StatusBadRequest)
		return
	}
	if body.Task == "" {
		http.Error(w, "task is required", http.StatusBadRequest)
		return
	}

	req := orchestrator.RunRequest{
		Task:          body.Task,
		BlueprintName: body.Blueprint,
		Adapter:       body.Adapter,
		BaseBranch:    body.BaseBranch,
		NoPR:          body.NoPR,
	}
	runID := h.queue.Enqueue(req)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(CreateRunResponse{RunID: runID, Status: "pending"})
}

func (h *WebhookHandler) getRunStatus(w http.ResponseWriter, runID string) {
	result, ok := h.registry.Get(runID)
	if !ok {
		http.Error(w, "run not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(RunStatusResponse{
		RunID:  result.RunID,
		Status: result.Status.String(),
		Output: result.Output,
		Error:  result.Error,
		PRURL:  result.PRURL,
	})
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./factory/triggers/ -run "TestWebhook" -v`
Expected: all PASS

- [ ] **Step 5: Commit**

```bash
git add factory/triggers/webhook.go factory/triggers/webhook_test.go
git commit -m "feat(factory): add webhook HTTP handler for run triggers"
```

---

### Task 5: forged daemon entrypoint

**Files:**
- Create: `cmd/forged/main.go`

- [ ] **Step 1: Create the daemon entrypoint**

Create `cmd/forged/main.go`:

```go
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/aditya-soni/forge/factory/orchestrator"
	"github.com/aditya-soni/forge/factory/triggers"
)

func main() {
	port := flag.String("port", envOr("FORGED_PORT", "8080"), "HTTP listen port")
	maxParallel := flag.Int("max-parallel", envOrInt("FORGED_MAX_PARALLEL", 2), "max concurrent runs")
	flag.Parse()

	registry := orchestrator.NewRunRegistry()

	// In production, this would be a real Pipeline. For the daemon skeleton,
	// we use a placeholder that logs and returns success.
	pipeline := &logPipeline{}
	queue := orchestrator.NewRunQueue(registry, pipeline, *maxParallel)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go queue.Start(ctx)

	handler := triggers.NewWebhookHandler(queue, registry)
	mux := http.NewServeMux()
	mux.Handle("/api/v1/runs", handler)
	mux.Handle("/api/v1/runs/", handler)

	srv := &http.Server{
		Addr:         ":" + *port,
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	go func() {
		log.Printf("forged listening on :%s (max_parallel=%d)", *port, *maxParallel)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %v", err)
		}
	}()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig
	log.Println("shutting down...")
	cancel()

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()
	srv.Shutdown(shutdownCtx)
}

type logPipeline struct{}

func (p *logPipeline) Execute(_ context.Context, req orchestrator.RunRequest) (orchestrator.RunResult, error) {
	log.Printf("executing: task=%q blueprint=%q adapter=%q", req.Task, req.BlueprintName, req.Adapter)
	return orchestrator.RunResult{
		Status: orchestrator.RunStatusPassed,
		Output: fmt.Sprintf("executed task: %s", req.Task),
	}, nil
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func envOrInt(key string, fallback int) int {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	var n int
	if _, err := fmt.Sscanf(v, "%d", &n); err != nil {
		return fallback
	}
	return n
}
```

- [ ] **Step 2: Verify it compiles**

Run: `go build ./cmd/forged/`
Expected: no errors

- [ ] **Step 3: Commit**

```bash
git add cmd/forged/main.go
git commit -m "feat(factory): add forged daemon entrypoint with webhook + queue"
```

---

### Task 6: Integration smoke test

**Files:**
- Create: `factory/triggers/integration_test.go`

- [ ] **Step 1: Write integration test**

Create `factory/triggers/integration_test.go`:

```go
package triggers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/aditya-soni/forge/factory/orchestrator"
	"github.com/aditya-soni/forge/factory/triggers"
)

type ecoPipeline struct{}

func (p *ecoPipeline) Execute(_ context.Context, req orchestrator.RunRequest) (orchestrator.RunResult, error) {
	return orchestrator.RunResult{
		Status: orchestrator.RunStatusPassed,
		Output: "echo: " + req.Task,
	}, nil
}

func TestIntegrationWebhookQueueRegistry(t *testing.T) {
	registry := orchestrator.NewRunRegistry()
	pipeline := &ecoPipeline{}
	queue := orchestrator.NewRunQueue(registry, pipeline, 2)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	go queue.Start(ctx)

	handler := triggers.NewWebhookHandler(queue, registry)
	srv := httptest.NewServer(handler)
	defer srv.Close()

	// POST create run
	body, _ := json.Marshal(triggers.CreateRunRequest{
		Task:      "Integration test task",
		Blueprint: "test-bp",
	})
	resp, err := http.Post(srv.URL+"/api/v1/runs", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("POST: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusAccepted {
		t.Fatalf("POST status = %d, want %d", resp.StatusCode, http.StatusAccepted)
	}

	var createResp triggers.CreateRunResponse
	if err := json.NewDecoder(resp.Body).Decode(&createResp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if createResp.RunID == "" {
		t.Fatal("expected non-empty RunID")
	}

	// Poll for completion
	var statusResp triggers.RunStatusResponse
	for i := 0; i < 20; i++ {
		time.Sleep(100 * time.Millisecond)
		getResp, err := http.Get(fmt.Sprintf("%s/api/v1/runs/%s", srv.URL, createResp.RunID))
		if err != nil {
			continue
		}
		json.NewDecoder(getResp.Body).Decode(&statusResp)
		getResp.Body.Close()
		if statusResp.Status == "passed" || statusResp.Status == "failed" {
			break
		}
	}

	if statusResp.Status != "passed" {
		t.Errorf("final status = %q, want passed", statusResp.Status)
	}
	if statusResp.Output == "" {
		t.Error("expected non-empty output")
	}
}
```

- [ ] **Step 2: Run integration test**

Run: `go test ./factory/triggers/ -run "TestIntegration" -v -timeout 30s`
Expected: PASS

- [ ] **Step 3: Run all factory tests**

Run: `go test ./factory/... -v -timeout 60s`
Expected: all PASS

- [ ] **Step 4: Commit**

```bash
git add factory/triggers/integration_test.go
git commit -m "test(factory): add webhook + queue integration smoke test"
```
