package sandbox

import (
	"context"
	"errors"
	"testing"
	"time"
)

type mockPoolRunner struct {
	createID  string
	startErr  error
	execOut   string
	execErr   error
	stopErr   error
	removeErr error
}

func (m *mockPoolRunner) Run(_ context.Context, name string, args ...string) (string, int, error) {
	if len(args) > 0 && args[0] == "create" {
		return m.createID, 0, nil
	}
	if len(args) > 0 && args[0] == "start" {
		if m.startErr != nil {
			return "", 1, m.startErr
		}
		return "", 0, nil
	}
	if len(args) > 0 && args[0] == "exec" {
		if m.execErr != nil {
			return m.execOut, 1, m.execErr
		}
		return m.execOut, 0, nil
	}
	if len(args) > 0 && args[0] == "stop" {
		return "", 0, m.stopErr
	}
	if len(args) > 0 && args[0] == "rm" {
		return "", 0, m.removeErr
	}
	return "", 0, nil
}

func TestDockerWarmPoolAcquireAndRelease(t *testing.T) {
	runner := &mockPoolRunner{createID: "container-abc"}
	pool := NewDockerWarmPool(runner, 2)

	ctx := context.Background()
	config := SandboxConfig{Image: "forge:latest", WorkspaceDir: "/tmp/ws"}

	pool.Preheat(ctx, config)
	time.Sleep(50 * time.Millisecond)

	container, err := pool.Acquire(ctx, config)
	if err != nil {
		t.Fatalf("Acquire: %v", err)
	}
	if container == nil {
		t.Fatal("expected non-nil container")
	}
	if container.ContainerID == "" {
		t.Error("expected non-empty ContainerID")
	}

	if err := pool.Release(container); err != nil {
		t.Fatalf("Release: %v", err)
	}
}

func TestDockerWarmPoolFallbackOnEmpty(t *testing.T) {
	runner := &mockPoolRunner{createID: "new-container"}
	pool := NewDockerWarmPool(runner, 0)

	ctx := context.Background()
	config := SandboxConfig{Image: "forge:latest"}

	_, err := pool.Acquire(ctx, config)
	if err != ErrPoolEmpty {
		t.Errorf("expected ErrPoolEmpty, got %v", err)
	}
}

func TestDockerWarmPoolReleaseDestroysOnResetFailure(t *testing.T) {
	runner := &mockPoolRunner{
		createID: "container-xyz",
		execErr:  errors.New("git clean failed"),
	}
	pool := NewDockerWarmPool(runner, 1)

	ctx := context.Background()
	config := SandboxConfig{Image: "forge:latest", WorkspaceDir: "/tmp/ws"}
	pool.Preheat(ctx, config)
	time.Sleep(50 * time.Millisecond)

	container, err := pool.Acquire(ctx, config)
	if err != nil {
		t.Fatalf("Acquire: %v", err)
	}

	if err := pool.Release(container); err != nil {
		t.Fatalf("Release: %v", err)
	}

	// Container should NOT be returned to the pool since reset failed.
	_, err = pool.Acquire(ctx, config)
	if err != ErrPoolEmpty {
		t.Errorf("expected ErrPoolEmpty after failed release, got %v", err)
	}
}

func TestDockerWarmPoolShutdown(t *testing.T) {
	runner := &mockPoolRunner{createID: "container-1"}
	pool := NewDockerWarmPool(runner, 1)

	ctx := context.Background()
	config := SandboxConfig{Image: "forge:latest"}
	pool.Preheat(ctx, config)
	time.Sleep(50 * time.Millisecond)

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := pool.Shutdown(shutdownCtx); err != nil {
		t.Fatalf("Shutdown: %v", err)
	}
}
