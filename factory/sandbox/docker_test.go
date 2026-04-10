package sandbox

import (
	"context"
	"fmt"
	"strings"
	"testing"
)

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
