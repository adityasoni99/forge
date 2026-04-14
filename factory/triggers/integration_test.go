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

	client := &http.Client{Timeout: 5 * time.Second}

	// POST create run
	body, _ := json.Marshal(triggers.CreateRunRequest{
		Task:      "Integration test task",
		Blueprint: "test-bp",
	})
	resp, err := client.Post(srv.URL+"/api/v1/runs", "application/json", bytes.NewReader(body))
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
	const maxPolls = 20
	polls := 0
	for i := 0; i < maxPolls; i++ {
		time.Sleep(100 * time.Millisecond)
		polls++
		getResp, err := client.Get(fmt.Sprintf("%s/api/v1/runs/%s", srv.URL, createResp.RunID))
		if err != nil {
			continue
		}
		if err := json.NewDecoder(getResp.Body).Decode(&statusResp); err != nil {
			getResp.Body.Close()
			continue
		}
		getResp.Body.Close()
		if statusResp.Status == "passed" || statusResp.Status == "failed" {
			break
		}
	}

	if statusResp.Status != "passed" {
		t.Errorf("final status = %q, want passed (polled %d times)", statusResp.Status, polls)
	}
	if statusResp.Output == "" {
		t.Error("expected non-empty output")
	}
}
