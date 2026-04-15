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

func dockerImageExists(image string) bool {
	cmd := exec.Command("docker", "inspect", "--type=image", image)
	return cmd.Run() == nil
}

func TestFactoryIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	if _, err := exec.LookPath("docker"); err != nil {
		t.Skip("docker not available, skipping factory integration test")
	}
	if !dockerImageExists("forge:latest") {
		t.Skip("forge:latest image not built, skipping (run 'make docker-build' first)")
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

func TestFactoryIntegrationSmokeBlueprint(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	if _, err := exec.LookPath("docker"); err != nil {
		t.Skip("docker not available, skipping factory integration test")
	}
	if !dockerImageExists("forge:latest") {
		t.Skip("forge:latest image not built, skipping (run 'make docker-build' first)")
	}

	repoDir := t.TempDir()
	initGitRepo(t, repoDir)
	copyFile(t,
		filepath.Join("testdata", "integration-smoke.yaml"),
		filepath.Join(repoDir, "integration-smoke.yaml"),
	)
	commitSmokeBlueprint(t, repoDir)

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
	}, []string{"--blueprint-file", "integration-smoke.yaml", "--task", "smoke task"})
	if err != nil {
		t.Fatalf("sandbox run: %v", err)
	}
	if result.ExitCode != 0 {
		t.Fatalf("exit code = %d, output = %s", result.ExitCode, result.Output)
	}
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

func copyFile(t *testing.T, src, dst string) {
	t.Helper()
	data, err := os.ReadFile(src)
	if err != nil {
		t.Fatalf("read %s: %v", src, err)
	}
	if err := os.WriteFile(dst, data, 0644); err != nil {
		t.Fatalf("write %s: %v", dst, err)
	}
}

// commitSmokeBlueprint records the blueprint in git so git worktree checkouts
// include the file (untracked files are not present in new worktrees).
func commitSmokeBlueprint(t *testing.T, repoDir string) {
	t.Helper()
	cmd := exec.Command("git", "add", "integration-smoke.yaml")
	cmd.Dir = repoDir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git add integration-smoke.yaml: %v: %s", err, out)
	}
	cmd = exec.Command("git", "commit", "-m", "add integration smoke blueprint")
	cmd.Dir = repoDir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git commit: %v: %s", err, out)
	}
}
