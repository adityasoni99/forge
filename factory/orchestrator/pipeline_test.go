package orchestrator

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/aditya-soni/forge/factory/delivery"
	"github.com/aditya-soni/forge/factory/sandbox"
	"github.com/aditya-soni/forge/factory/workspace"
)

func TestRunStatusString(t *testing.T) {
	tests := []struct {
		status RunStatus
		want   string
	}{
		{RunStatusPending, "pending"},
		{RunStatusRunning, "running"},
		{RunStatusPassed, "passed"},
		{RunStatusFailed, "failed"},
		{RunStatus(99), "unknown"},
	}
	for _, tt := range tests {
		if got := tt.status.String(); got != tt.want {
			t.Errorf("RunStatus(%d).String() = %q, want %q", tt.status, got, tt.want)
		}
	}
}

type mockSandboxManager struct {
	ensureCalled bool
	lastCommand  []string
	runResult    sandbox.SandboxResult
	runErr       error
}

func (m *mockSandboxManager) EnsureImage(_ context.Context, _ sandbox.SandboxConfig) error {
	m.ensureCalled = true
	return nil
}

func (m *mockSandboxManager) Run(_ context.Context, _ sandbox.SandboxConfig, cmd []string) (sandbox.SandboxResult, error) {
	m.lastCommand = cmd
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

// --- Failing mock variants for error path coverage ---

type failingWorkspaceManager struct{}

func (m *failingWorkspaceManager) Create(_ context.Context, _, _ string) (*workspace.Workspace, error) {
	return nil, fmt.Errorf("git worktree add failed")
}

func (m *failingWorkspaceManager) Destroy(_ context.Context, _ *workspace.Workspace) error {
	return nil
}

type failingEnsureImageManager struct{}

func (m *failingEnsureImageManager) EnsureImage(_ context.Context, _ sandbox.SandboxConfig) error {
	return fmt.Errorf("docker pull failed")
}

func (m *failingEnsureImageManager) Run(_ context.Context, _ sandbox.SandboxConfig, _ []string) (sandbox.SandboxResult, error) {
	return sandbox.SandboxResult{}, nil
}

type failingSandboxRunManager struct{}

func (m *failingSandboxRunManager) EnsureImage(_ context.Context, _ sandbox.SandboxConfig) error {
	return nil
}

func (m *failingSandboxRunManager) Run(_ context.Context, _ sandbox.SandboxConfig, _ []string) (sandbox.SandboxResult, error) {
	return sandbox.SandboxResult{}, fmt.Errorf("docker run error")
}

type failingDeliveryManager struct{}

func (m *failingDeliveryManager) Deliver(_ context.Context, _, _ string, _ delivery.DeliveryConfig) (delivery.DeliveryResult, error) {
	return delivery.DeliveryResult{}, fmt.Errorf("git push failed")
}

// --- Error path tests ---

func TestPipelineWorkspaceCreateFails(t *testing.T) {
	p := NewPipeline(&mockSandboxManager{}, &failingWorkspaceManager{}, &mockDeliveryManager{})
	result, err := p.Execute(context.Background(), RunRequest{Task: "test", RepoDir: "/repo", Image: "forge:latest"})
	if err != nil {
		t.Fatalf("Execute should not return error: %v", err)
	}
	if result.Status != RunStatusFailed {
		t.Errorf("Status = %v, want Failed", result.Status)
	}
	if !strings.Contains(result.Error, "workspace create") {
		t.Errorf("Error = %q, want to contain 'workspace create'", result.Error)
	}
}

func TestPipelineEnsureImageFails(t *testing.T) {
	ws := &mockWorkspaceManager{ws: &workspace.Workspace{Dir: "/tmp/test", Branch: "forge/run-1", RepoDir: "/repo"}}
	p := NewPipeline(&failingEnsureImageManager{}, ws, &mockDeliveryManager{})
	result, err := p.Execute(context.Background(), RunRequest{Task: "test", RepoDir: "/repo", Image: "forge:latest"})
	if err != nil {
		t.Fatalf("Execute should not return error: %v", err)
	}
	if result.Status != RunStatusFailed {
		t.Errorf("Status = %v, want Failed", result.Status)
	}
	if !ws.destroyed {
		t.Error("workspace should be cleaned up even on ensure image failure")
	}
}

func TestPipelineSandboxRunError(t *testing.T) {
	ws := &mockWorkspaceManager{ws: &workspace.Workspace{Dir: "/tmp/test", Branch: "forge/run-1", RepoDir: "/repo"}}
	p := NewPipeline(&failingSandboxRunManager{}, ws, &mockDeliveryManager{})
	result, err := p.Execute(context.Background(), RunRequest{Task: "test", RepoDir: "/repo", Image: "forge:latest"})
	if err != nil {
		t.Fatalf("Execute should not return error: %v", err)
	}
	if result.Status != RunStatusFailed {
		t.Errorf("Status = %v, want Failed", result.Status)
	}
	if !strings.Contains(result.Error, "sandbox run") {
		t.Errorf("Error = %q, want to contain 'sandbox run'", result.Error)
	}
}

func TestPipelineDeliveryFails(t *testing.T) {
	sbx := &mockSandboxManager{runResult: sandbox.SandboxResult{ExitCode: 0, Output: "ok"}}
	ws := &mockWorkspaceManager{ws: &workspace.Workspace{Dir: "/tmp/test", Branch: "forge/run-1", RepoDir: "/repo"}}
	p := NewPipeline(sbx, ws, &failingDeliveryManager{})
	result, err := p.Execute(context.Background(), RunRequest{Task: "test", RepoDir: "/repo", Image: "forge:latest"})
	if err != nil {
		t.Fatalf("Execute should not return error: %v", err)
	}
	if result.Status != RunStatusFailed {
		t.Errorf("Status = %v, want Failed", result.Status)
	}
	if !strings.Contains(result.Error, "delivery") {
		t.Errorf("Error = %q, want to contain 'delivery'", result.Error)
	}
}

func TestPipelineDefaultImage(t *testing.T) {
	sbx := &mockSandboxManager{runResult: sandbox.SandboxResult{ExitCode: 0}}
	ws := &mockWorkspaceManager{ws: &workspace.Workspace{Dir: "/tmp/test", Branch: "forge/run-1", RepoDir: "/repo"}}
	dlv := &mockDeliveryManager{result: delivery.DeliveryResult{Pushed: true}}

	p := NewPipeline(sbx, ws, dlv)
	result, err := p.Execute(context.Background(), RunRequest{
		Task:    "test",
		RepoDir: "/repo",
		NoPR:    true,
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if result.Status != RunStatusPassed {
		t.Errorf("Status = %v, want Passed (default image should work)", result.Status)
	}
}

func TestBuildSandboxCommandAllFields(t *testing.T) {
	req := RunRequest{
		BlueprintName: "bug-fix",
		BlueprintFile: "/path/to/bp.yaml",
		Task:          "Fix bug",
		Adapter:       "claude",
	}
	args := buildSandboxCommand(req)
	if len(args) != 8 {
		t.Fatalf("expected 8 args, got %d: %v", len(args), args)
	}
	expected := []string{
		"--blueprint", "bug-fix",
		"--blueprint-file", "/path/to/bp.yaml",
		"--task", "Fix bug",
		"--adapter", "claude",
	}
	for i, want := range expected {
		if args[i] != want {
			t.Errorf("args[%d] = %q, want %q", i, args[i], want)
		}
	}
}

func TestBuildSandboxCommandMinimal(t *testing.T) {
	args := buildSandboxCommand(RunRequest{})
	if len(args) != 0 {
		t.Errorf("expected 0 args for empty request, got %d: %v", len(args), args)
	}
}

func TestPipelineUsesTaskAssigner(t *testing.T) {
	sbx := &mockSandboxManager{runResult: sandbox.SandboxResult{ExitCode: 0, Output: "ok"}}
	ws := &mockWorkspaceManager{ws: &workspace.Workspace{Dir: "/tmp/test", Branch: "forge/run-1", RepoDir: "/repo"}}
	dlv := &mockDeliveryManager{}
	assigner := NewTaskAssigner()
	p := NewPipeline(sbx, ws, dlv, WithTaskAssigner(assigner))

	req := RunRequest{
		Task:    "implement feature",
		RepoDir: t.TempDir(),
		NoPR:    true,
	}
	result, err := p.Execute(context.Background(), req)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if result.Status != RunStatusPassed {
		t.Errorf("Status = %v, want Passed", result.Status)
	}
	found := false
	for i, arg := range sbx.lastCommand {
		if arg == "--adapter" && i+1 < len(sbx.lastCommand) && sbx.lastCommand[i+1] == "claude" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected --adapter claude in sandbox command, got %v", sbx.lastCommand)
	}
}

func TestPipelineEmitsSessionEvents(t *testing.T) {
	sbx := &mockSandboxManager{runResult: sandbox.SandboxResult{ExitCode: 0, Output: "ok"}}
	ws := &mockWorkspaceManager{ws: &workspace.Workspace{Dir: "/tmp/test", Branch: "forge/run-1", RepoDir: "/repo"}}
	dlv := &mockDeliveryManager{}
	sessionLog := NewFileSessionLog(t.TempDir())
	p := NewPipeline(sbx, ws, dlv, WithSessionLog(sessionLog))

	result, err := p.Execute(context.Background(), RunRequest{
		Task:    "test session",
		RepoDir: "/repo",
		NoPR:    true,
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}

	events, err := sessionLog.GetEvents(context.Background(), result.RunID)
	if err != nil {
		t.Fatalf("GetEvents: %v", err)
	}
	if len(events) != 3 {
		t.Fatalf("expected 3 session events, got %d", len(events))
	}
	wantTypes := []SessionEventType{EventWorkspaceCreated, EventNodeCompleted, EventRunComplete}
	for i, want := range wantTypes {
		if events[i].Type != want {
			t.Errorf("events[%d].Type = %q, want %q", i, events[i].Type, want)
		}
	}
}

func TestSubplanAIntegration(t *testing.T) {
	sessDir := t.TempDir()
	sessionLog := NewFileSessionLog(sessDir)
	registry := NewRunRegistry()

	pipeline := &mockPipeline{fn: func(ctx context.Context, req RunRequest) (RunResult, error) {
		return RunResult{Status: RunStatusPassed, Output: "done"}, nil
	}}

	queue := NewRunQueue(registry, pipeline, 2)
	ctx, cancel := context.WithCancel(context.Background())
	go queue.Start(ctx)

	id := queue.Enqueue(RunRequest{Task: "test task"})
	queueCtx, queueCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer queueCancel()
	if err := queue.Wait(queueCtx, id); err != nil {
		t.Fatalf("Wait: %v", err)
	}

	result, ok := registry.Get(id)
	if !ok {
		t.Fatal("run not found in registry")
	}
	if result.Status != RunStatusPassed {
		t.Errorf("status = %v, want passed", result.Status)
	}

	if err := sessionLog.Emit(context.Background(), SessionEvent{
		RunID:     id,
		Type:      EventRunComplete,
		Timestamp: time.Now(),
		Data:      map[string]interface{}{"status": "passed"},
	}); err != nil {
		t.Fatalf("Emit: %v", err)
	}
	events, err := sessionLog.GetEvents(context.Background(), id)
	if err != nil {
		t.Fatalf("GetEvents: %v", err)
	}
	if len(events) != 1 {
		t.Errorf("events = %d, want 1", len(events))
	}

	cancel()
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer shutdownCancel()
	if err := queue.Shutdown(shutdownCtx); err != nil {
		t.Fatalf("Shutdown: %v", err)
	}
}

func TestBuildSandboxCommandBlueprintFileOnly(t *testing.T) {
	req := RunRequest{
		BlueprintFile: "tests/testdata/integration-smoke.yaml",
		Task:          "smoke task",
		Adapter:       "echo",
	}
	args := buildSandboxCommand(req)
	want := []string{
		"--blueprint-file", "tests/testdata/integration-smoke.yaml",
		"--task", "smoke task",
		"--adapter", "echo",
	}
	if len(args) != len(want) {
		t.Fatalf("expected %d args, got %d: %v", len(want), len(args), args)
	}
	for i, w := range want {
		if args[i] != w {
			t.Errorf("args[%d] = %q, want %q", i, args[i], w)
		}
	}
}
