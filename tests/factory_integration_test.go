package tests

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/aditya-soni/forge/factory/sandbox"
	"github.com/aditya-soni/forge/factory/workspace"
)

func TestFactoryIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	if _, err := exec.LookPath("docker"); err != nil {
		t.Skip("docker not available, skipping factory integration test")
	}

	repoDir := t.TempDir()
	initGitRepo(t, repoDir)

	wsMgr := workspace.NewManager()
	ws, err := wsMgr.Create(context.Background(), repoDir, "integration-test")
	if err != nil {
		t.Fatalf("workspace create: %v", err)
	}
	defer wsMgr.Destroy(context.Background(), ws)

	runner := &sandbox.ExecRunner{}
	sbx := sandbox.NewDockerSandbox(runner)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	result, err := sbx.Run(ctx, sandbox.SandboxConfig{
		Image:        "forge:latest",
		WorkspaceDir: ws.Dir,
		Env:          map[string]string{"FORGE_ADAPTER": "echo"},
		NetworkMode:  "none",
	}, []string{"--blueprint", "bug-fix"})
	if err != nil {
		t.Fatalf("sandbox run: %v", err)
	}
	t.Logf("Exit code: %d, Output: %s", result.ExitCode, result.Output)
}

func initGitRepo(t *testing.T, dir string) {
	t.Helper()
	for _, args := range [][]string{
		{"git", "init"},
		{"git", "config", "user.email", "test@forge.dev"},
		{"git", "config", "user.name", "Forge Test"},
	} {
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("%v: %s", err, out)
		}
	}
	os.WriteFile(filepath.Join(dir, "README.md"), []byte("# Test Repo"), 0644)
	cmd := exec.Command("git", "add", "-A")
	cmd.Dir = dir
	cmd.Run()
	cmd = exec.Command("git", "commit", "-m", "init")
	cmd.Dir = dir
	cmd.Run()
}
