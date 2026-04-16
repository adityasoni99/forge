package triggers

import (
	"context"
	"os"
	"os/exec"
	"testing"
)

func TestGitRepoResolverClonesAndReturns(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not found")
	}

	origin := t.TempDir()
	cmd := exec.Command("git", "init", "--bare", origin)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git init --bare: %s: %v", out, err)
	}

	cacheDir := t.TempDir()
	resolver := NewGitRepoResolver(cacheDir)

	path, err := resolver.Resolve(context.Background(), origin)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if path == "" {
		t.Fatal("expected non-empty path")
	}
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Errorf("resolved path does not exist: %s", path)
	}
}

func TestGitRepoResolverReusesCache(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not found")
	}

	origin := t.TempDir()
	_ = exec.Command("git", "init", "--bare", origin).Run()

	cacheDir := t.TempDir()
	resolver := NewGitRepoResolver(cacheDir)

	path1, err := resolver.Resolve(context.Background(), origin)
	if err != nil {
		t.Fatalf("first Resolve: %v", err)
	}
	path2, err := resolver.Resolve(context.Background(), origin)
	if err != nil {
		t.Fatalf("second Resolve: %v", err)
	}

	if path1 != path2 {
		t.Errorf("expected same cached path, got %q and %q", path1, path2)
	}
}

func TestGitRepoResolverRejectsExtProtocol(t *testing.T) {
	cacheDir := t.TempDir()
	resolver := NewGitRepoResolver(cacheDir)

	_, err := resolver.Resolve(context.Background(), "ext::sh -c 'echo pwned'")
	if err == nil {
		t.Error("expected error for ext:: protocol URL")
	}
}

func TestGitRepoResolverInvalidURL(t *testing.T) {
	cacheDir := t.TempDir()
	resolver := NewGitRepoResolver(cacheDir)

	_, err := resolver.Resolve(context.Background(), "https://nonexistent.invalid/repo.git")
	if err == nil {
		t.Error("expected error for invalid repo URL")
	}
}
