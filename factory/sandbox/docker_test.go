package sandbox

import (
	"context"
	"fmt"
	"strings"
	"testing"
)

func TestExecRunnerSuccess(t *testing.T) {
	r := &ExecRunner{}
	output, code, err := r.Run(context.Background(), "echo", "hello")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if code != 0 {
		t.Errorf("exit code = %d, want 0", code)
	}
	if !strings.Contains(output, "hello") {
		t.Errorf("output = %q, want to contain 'hello'", output)
	}
}

func TestExecRunnerNonZeroExit(t *testing.T) {
	r := &ExecRunner{}
	_, code, err := r.Run(context.Background(), "sh", "-c", "exit 42")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if code != 42 {
		t.Errorf("exit code = %d, want 42", code)
	}
}

func TestExecRunnerCommandNotFound(t *testing.T) {
	r := &ExecRunner{}
	_, code, err := r.Run(context.Background(), "forge-nonexistent-cmd-xyz")
	if err == nil {
		t.Fatal("expected error for nonexistent command")
	}
	if code != -1 {
		t.Errorf("exit code = %d, want -1", code)
	}
}

func TestNewDockerSandboxNilRunner(t *testing.T) {
	ds := NewDockerSandbox(nil)
	if ds == nil {
		t.Fatal("expected non-nil DockerSandbox")
	}
}

type mockRunner struct {
	calls []mockCall
	idx   int
}

type mockCall struct {
	wantName string
	wantArgs []string
	output   string
	exitCode int
	err      error
}

func (m *mockRunner) Run(_ context.Context, name string, args ...string) (string, int, error) {
	if m.idx >= len(m.calls) {
		return "", -1, fmt.Errorf("unexpected call #%d: %s %v", m.idx, name, args)
	}
	call := m.calls[m.idx]
	m.idx++
	if call.wantName != "" && call.wantName != name {
		return "", -1, fmt.Errorf("call #%d: want name %q, got %q", m.idx-1, call.wantName, name)
	}
	return call.output, call.exitCode, call.err
}

func TestDockerSandboxEnsureImageExists(t *testing.T) {
	runner := &mockRunner{calls: []mockCall{
		{wantName: "docker", output: "[{\"Id\":\"sha256:abc\"}]", exitCode: 0},
	}}
	ds := NewDockerSandbox(runner)
	err := ds.EnsureImage(context.Background(), SandboxConfig{Image: "forge:latest"})
	if err != nil {
		t.Fatalf("EnsureImage: %v", err)
	}
}

func TestDockerSandboxEnsureImagePulls(t *testing.T) {
	runner := &mockRunner{calls: []mockCall{
		{wantName: "docker", exitCode: 1},
		{wantName: "docker", output: "Pulled", exitCode: 0},
	}}
	ds := NewDockerSandbox(runner)
	err := ds.EnsureImage(context.Background(), SandboxConfig{Image: "forge:latest"})
	if err != nil {
		t.Fatalf("EnsureImage: %v", err)
	}
	if runner.idx != 2 {
		t.Errorf("expected 2 calls (inspect + pull), got %d", runner.idx)
	}
}

func TestDockerSandboxRunSuccess(t *testing.T) {
	runner := &mockRunner{calls: []mockCall{
		{wantName: "docker", output: "blueprint completed", exitCode: 0},
	}}
	ds := NewDockerSandbox(runner)
	result, err := ds.Run(context.Background(), SandboxConfig{
		Image:        "forge:latest",
		WorkspaceDir: "/tmp/workspace",
		Env:          map[string]string{"FORGE_ADAPTER": "echo"},
		NetworkMode:  "none",
	}, []string{"--blueprint", "bug-fix"})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if result.ExitCode != 0 {
		t.Errorf("ExitCode = %d, want 0", result.ExitCode)
	}
	if !strings.Contains(result.Output, "blueprint completed") {
		t.Errorf("Output = %q, want to contain 'blueprint completed'", result.Output)
	}
}

func TestDockerSandboxRunWithLimits(t *testing.T) {
	runner := &mockRunner{calls: []mockCall{
		{wantName: "docker", exitCode: 0},
	}}
	ds := NewDockerSandbox(runner)
	_, _ = ds.Run(context.Background(), SandboxConfig{
		Image:       "forge:latest",
		CPULimit:    "2.0",
		MemoryLimit: "4g",
	}, []string{})
	if runner.idx != 1 {
		t.Errorf("expected 1 call, got %d", runner.idx)
	}
}

func TestDockerSandboxRunNonZeroExit(t *testing.T) {
	runner := &mockRunner{calls: []mockCall{
		{wantName: "docker", output: "lint failed", exitCode: 1},
	}}
	ds := NewDockerSandbox(runner)
	result, err := ds.Run(context.Background(), SandboxConfig{
		Image: "forge:latest",
	}, []string{})
	if err != nil {
		t.Fatalf("Run should not error on non-zero exit: %v", err)
	}
	if result.ExitCode != 1 {
		t.Errorf("ExitCode = %d, want 1", result.ExitCode)
	}
}

type blockingRunner struct{}

func (blockingRunner) Run(ctx context.Context, _ string, _ ...string) (string, int, error) {
	<-ctx.Done()
	return "", -1, ctx.Err()
}

func TestDockerSandboxRunRespectsContextCancel(t *testing.T) {
	ds := NewDockerSandbox(blockingRunner{})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := ds.Run(ctx, SandboxConfig{Image: "forge:latest"}, []string{})
	if err == nil {
		t.Fatal("expected error when context is already cancelled")
	}
}

func TestDockerSandboxEnsureImagePullFails(t *testing.T) {
	runner := &mockRunner{calls: []mockCall{
		{wantName: "docker", exitCode: 1},
		{wantName: "docker", output: "error", exitCode: 1},
	}}
	ds := NewDockerSandbox(runner)
	err := ds.EnsureImage(context.Background(), SandboxConfig{Image: "bad:image"})
	if err == nil {
		t.Fatal("expected error when pull fails")
	}
}

func TestDockerSandboxEnsureImageInspectError(t *testing.T) {
	runner := &mockRunner{calls: []mockCall{
		{wantName: "docker", err: fmt.Errorf("connection refused")},
	}}
	ds := NewDockerSandbox(runner)
	err := ds.EnsureImage(context.Background(), SandboxConfig{Image: "forge:latest"})
	if err == nil {
		t.Fatal("expected error when docker inspect fails with error")
	}
}

func TestBuildRunArgsFiltersSecrets(t *testing.T) {
	ds := NewDockerSandbox(nil)
	config := SandboxConfig{
		Image:        "forge:latest",
		WorkspaceDir: "/tmp/ws",
		Env: map[string]string{
			"SAFE_VAR":       "ok",
			"API_KEY":        "secret123",
			"DB_PASSWORD":    "pass",
			"AWS_SECRET_KEY": "aws-secret",
			"FORGE_ADAPTER":  "claude",
		},
	}
	args := ds.buildRunArgs(config, []string{"run"})

	secretEntries := []string{"API_KEY=secret123", "DB_PASSWORD=pass", "AWS_SECRET_KEY=aws-secret"}
	for _, arg := range args {
		for _, secret := range secretEntries {
			if arg == secret {
				t.Errorf("secret env var %q leaked into args", arg)
			}
		}
	}

	found := false
	for i, arg := range args {
		if arg == "-e" && i+1 < len(args) && args[i+1] == "SAFE_VAR=ok" {
			found = true
		}
		if arg == "-e" && i+1 < len(args) && args[i+1] == "FORGE_ADAPTER=claude" {
			found = true
		}
	}
	if !found {
		t.Error("expected safe env vars in args")
	}
}

func TestDockerSandboxEnsureImagePullError(t *testing.T) {
	runner := &mockRunner{calls: []mockCall{
		{wantName: "docker", exitCode: 1},
		{wantName: "docker", err: fmt.Errorf("network error")},
	}}
	ds := NewDockerSandbox(runner)
	err := ds.EnsureImage(context.Background(), SandboxConfig{Image: "forge:latest"})
	if err == nil {
		t.Fatal("expected error when pull returns error")
	}
}

// mockWarmPool implements WarmPool for testing the warm-container path
// in DockerSandbox.Run.
type mockWarmPool struct {
	acquireContainer *WarmContainer
	acquireErr       error
	released         bool
}

func (m *mockWarmPool) Acquire(_ context.Context, _ SandboxConfig) (*WarmContainer, error) {
	return m.acquireContainer, m.acquireErr
}

func (m *mockWarmPool) Release(_ *WarmContainer) error {
	m.released = true
	return nil
}

func (m *mockWarmPool) Shutdown(_ context.Context) error { return nil }

func TestDockerSandboxRunUsesWarmPool(t *testing.T) {
	runner := &mockRunner{calls: []mockCall{
		{wantName: "docker", output: "warm result", exitCode: 0},
	}}
	pool := &mockWarmPool{
		acquireContainer: &WarmContainer{ContainerID: "warm-abc", Image: "forge:latest"},
	}
	ds := NewDockerSandbox(runner)
	ds.SetWarmPool(pool)

	result, err := ds.Run(context.Background(), SandboxConfig{Image: "forge:latest"}, []string{"--blueprint", "test"})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if result.ExitCode != 0 {
		t.Errorf("ExitCode = %d, want 0", result.ExitCode)
	}
	if !strings.Contains(result.Output, "warm result") {
		t.Errorf("Output = %q, want to contain 'warm result'", result.Output)
	}
	if !pool.released {
		t.Error("expected pool.Release to be called")
	}
}

func TestDockerSandboxRunFallsBackOnPoolEmpty(t *testing.T) {
	runner := &mockRunner{calls: []mockCall{
		{wantName: "docker", output: "cold result", exitCode: 0},
	}}
	pool := &mockWarmPool{acquireErr: ErrPoolEmpty}
	ds := NewDockerSandbox(runner)
	ds.SetWarmPool(pool)

	result, err := ds.Run(context.Background(), SandboxConfig{Image: "forge:latest"}, []string{})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if !strings.Contains(result.Output, "cold result") {
		t.Errorf("Output = %q, want to contain 'cold result' (cold path)", result.Output)
	}
	if pool.released {
		t.Error("expected pool.Release NOT to be called on fallback path")
	}
}

func TestDockerSandboxRunWarmContainerExecError(t *testing.T) {
	runner := &mockRunner{calls: []mockCall{
		{wantName: "docker", err: fmt.Errorf("exec failed")},
	}}
	pool := &mockWarmPool{
		acquireContainer: &WarmContainer{ContainerID: "warm-xyz", Image: "forge:latest"},
	}
	ds := NewDockerSandbox(runner)
	ds.SetWarmPool(pool)

	_, err := ds.Run(context.Background(), SandboxConfig{Image: "forge:latest"}, []string{})
	if err == nil {
		t.Fatal("expected error when docker exec fails in warm container")
	}
	if !strings.Contains(err.Error(), "warm container") {
		t.Errorf("error = %q, want to mention 'warm container'", err.Error())
	}
	if !pool.released {
		t.Error("expected pool.Release to be called even on exec error")
	}
}
