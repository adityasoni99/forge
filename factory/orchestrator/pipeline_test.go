package orchestrator

import (
	"context"
	"testing"
	"time"

	"github.com/aditya-soni/forge/factory/delivery"
	"github.com/aditya-soni/forge/factory/sandbox"
	"github.com/aditya-soni/forge/factory/workspace"
)

type mockSandboxManager struct {
	ensureCalled bool
	runResult    sandbox.SandboxResult
	runErr       error
}

func (m *mockSandboxManager) EnsureImage(_ context.Context, _ sandbox.SandboxConfig) error {
	m.ensureCalled = true
	return nil
}

func (m *mockSandboxManager) Run(_ context.Context, _ sandbox.SandboxConfig, _ []string) (sandbox.SandboxResult, error) {
	return m.runResult, m.runErr
}

type mockWorkspaceManager struct {
	ws        *workspace.Workspace
	created   bool
	destroyed bool
}

func (m *mockWorkspaceManager) Create(_ context.Context, _, _ string) (*workspace.Workspace, error) {
	m.created = true
	return m.ws, nil
}

func (m *mockWorkspaceManager) Destroy(_ context.Context, _ *workspace.Workspace) error {
	m.destroyed = true
	return nil
}

type mockDeliveryManager struct {
	result    delivery.DeliveryResult
	delivered bool
}

func (m *mockDeliveryManager) Deliver(_ context.Context, _, _ string, _ delivery.DeliveryConfig) (delivery.DeliveryResult, error) {
	m.delivered = true
	return m.result, nil
}

func TestPipelineSuccess(t *testing.T) {
	sbx := &mockSandboxManager{runResult: sandbox.SandboxResult{ExitCode: 0, Output: "done", Duration: time.Second}}
	ws := &mockWorkspaceManager{ws: &workspace.Workspace{Dir: "/tmp/test", Branch: "forge/run-1", RepoDir: "/repo"}}
	dlv := &mockDeliveryManager{result: delivery.DeliveryResult{Pushed: true, PRCreated: true, PRURL: "https://github.com/x/y/pull/1"}}

	p := NewPipeline(sbx, ws, dlv)
	result, err := p.Execute(context.Background(), RunRequest{
		Task:          "Fix bug",
		BlueprintName: "bug-fix",
		RepoDir:       "/repo",
		Image:         "forge:latest",
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if result.Status != RunStatusPassed {
		t.Errorf("Status = %v, want Passed", result.Status)
	}
	if !ws.created || !ws.destroyed {
		t.Error("workspace should be created and destroyed")
	}
	if !sbx.ensureCalled {
		t.Error("sandbox image should be ensured")
	}
	if !dlv.delivered {
		t.Error("delivery should be called")
	}
	if result.PRURL != "https://github.com/x/y/pull/1" {
		t.Errorf("PRURL = %q", result.PRURL)
	}
	if len(result.Events) < 3 {
		t.Errorf("expected structured lifecycle events, got %d", len(result.Events))
	}
}

func TestPipelineSandboxFails(t *testing.T) {
	sbx := &mockSandboxManager{runResult: sandbox.SandboxResult{ExitCode: 1, Output: "lint failed"}}
	ws := &mockWorkspaceManager{ws: &workspace.Workspace{Dir: "/tmp/test", Branch: "forge/run-1", RepoDir: "/repo"}}
	dlv := &mockDeliveryManager{}

	p := NewPipeline(sbx, ws, dlv)
	result, err := p.Execute(context.Background(), RunRequest{
		Task:          "Fix bug",
		BlueprintName: "bug-fix",
		RepoDir:       "/repo",
		Image:         "forge:latest",
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if result.Status != RunStatusFailed {
		t.Errorf("Status = %v, want Failed", result.Status)
	}
	if dlv.delivered {
		t.Error("delivery should NOT be called on failure")
	}
	if !ws.destroyed {
		t.Error("workspace should still be cleaned up on failure")
	}
}

func TestPipelineNoPR(t *testing.T) {
	sbx := &mockSandboxManager{runResult: sandbox.SandboxResult{ExitCode: 0}}
	ws := &mockWorkspaceManager{ws: &workspace.Workspace{Dir: "/tmp/test", Branch: "forge/run-1", RepoDir: "/repo"}}
	dlv := &mockDeliveryManager{}

	p := NewPipeline(sbx, ws, dlv)
	result, _ := p.Execute(context.Background(), RunRequest{
		Task:    "Fix bug",
		RepoDir: "/repo",
		Image:   "forge:latest",
		NoPR:    true,
	})
	if result.Status != RunStatusPassed {
		t.Errorf("Status = %v, want Passed", result.Status)
	}
	if dlv.delivered {
		t.Error("delivery should not be called with NoPR=true")
	}
}
