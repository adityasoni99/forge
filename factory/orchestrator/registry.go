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
		result.RunID = runID
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
