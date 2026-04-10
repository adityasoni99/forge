package workspace

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func initTestRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	cmds := [][]string{
		{"git", "init"},
		{"git", "config", "user.email", "test@test.com"},
		{"git", "config", "user.name", "Test"},
	}
	for _, c := range cmds {
		cmd := exec.Command(c[0], c[1:]...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("%v: %s", err, out)
		}
	}
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("# Test"), 0644); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("git", "add", "-A")
	cmd.Dir = dir
	cmd.Run()
	cmd = exec.Command("git", "commit", "-m", "init")
	cmd.Dir = dir
	cmd.Run()
	return dir
}

func TestCreateWorktree(t *testing.T) {
	repoDir := initTestRepo(t)
	mgr := NewManager()
	ws, err := mgr.Create(context.Background(), repoDir, "test-run-1")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	defer mgr.Destroy(context.Background(), ws)

	if ws.Dir == "" {
		t.Fatal("Dir is empty")
	}
	if ws.Branch == "" {
		t.Fatal("Branch is empty")
	}
	readme := filepath.Join(ws.Dir, "README.md")
	if _, err := os.Stat(readme); err != nil {
		t.Fatalf("README.md not found in worktree: %v", err)
	}
}

func TestDestroyWorktree(t *testing.T) {
	repoDir := initTestRepo(t)
	mgr := NewManager()
	ws, err := mgr.Create(context.Background(), repoDir, "test-run-2")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	dir := ws.Dir
	if err := mgr.Destroy(context.Background(), ws); err != nil {
		t.Fatalf("Destroy: %v", err)
	}
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Errorf("worktree dir should be removed, but stat returned: %v", err)
	}
}

func TestCreateWorktreeNotARepo(t *testing.T) {
	mgr := NewManager()
	_, err := mgr.Create(context.Background(), t.TempDir(), "run-1")
	if err == nil {
		t.Fatal("expected error for non-git directory")
	}
}
