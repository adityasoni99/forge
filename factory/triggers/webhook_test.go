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

// Compile-time interface checks.
var _ Enqueuer = (*stubQueue)(nil)
var _ StatusGetter = (*stubRegistry)(nil)
