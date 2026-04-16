package triggers

import (
	"encoding/json"
	"fmt"
	"log"
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
	resolver RepoResolver
}

// WebhookOption configures optional WebhookHandler behaviour.
type WebhookOption func(*WebhookHandler)

// WithRepoResolver attaches a RepoResolver that converts incoming repo_url
// values into local paths before enqueueing a run.
func WithRepoResolver(r RepoResolver) WebhookOption {
	return func(h *WebhookHandler) { h.resolver = r }
}

// NewWebhookHandler creates a handler wired to the given queue and registry.
func NewWebhookHandler(queue Enqueuer, registry StatusGetter, opts ...WebhookOption) *WebhookHandler {
	h := &WebhookHandler{queue: queue, registry: registry}
	for _, opt := range opts {
		opt(h)
	}
	return h
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
		http.Error(w, "invalid request body", http.StatusBadRequest)
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

	if body.RepoURL != "" && h.resolver != nil {
		localPath, err := h.resolver.Resolve(r.Context(), body.RepoURL)
		if err != nil {
			http.Error(w, fmt.Sprintf("failed to resolve repo_url: %v", err), http.StatusBadRequest)
			return
		}
		req.RepoDir = localPath
	}

	runID := h.queue.Enqueue(req)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	if err := json.NewEncoder(w).Encode(CreateRunResponse{RunID: runID, Status: "pending"}); err != nil {
		log.Printf("webhook: encode response: %v", err)
	}
}

func (h *WebhookHandler) getRunStatus(w http.ResponseWriter, runID string) {
	result, ok := h.registry.Get(runID)
	if !ok {
		http.Error(w, "run not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(RunStatusResponse{
		RunID:  result.RunID,
		Status: result.Status.String(),
		Output: result.Output,
		Error:  result.Error,
		PRURL:  result.PRURL,
	}); err != nil {
		log.Printf("webhook: encode response: %v", err)
	}
}
