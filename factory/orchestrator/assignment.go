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
