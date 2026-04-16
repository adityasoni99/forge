package triggers

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
)

// RepoResolver resolves a repository URL to a local path.
type RepoResolver interface {
	Resolve(ctx context.Context, repoURL string) (localPath string, err error)
}

// GitRepoResolver clones bare repositories into a local cache directory
// and reuses them for subsequent requests to the same URL.
type GitRepoResolver struct {
	cacheDir string
}

// NewGitRepoResolver creates a resolver that caches bare clones under cacheDir.
func NewGitRepoResolver(cacheDir string) *GitRepoResolver {
	return &GitRepoResolver{cacheDir: cacheDir}
}

func validateRepoURL(raw string) error {
	u, err := url.Parse(raw)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}
	switch u.Scheme {
	case "https", "http", "ssh", "git":
		return nil
	case "":
		if filepath.IsAbs(raw) {
			return nil
		}
		return fmt.Errorf("unsupported repo URL scheme (empty)")
	default:
		return fmt.Errorf("unsupported repo URL scheme: %q", u.Scheme)
	}
}

// Resolve returns a local path to a bare clone of repoURL. If the clone
// already exists, it fetches updates instead of re-cloning.
func (r *GitRepoResolver) Resolve(ctx context.Context, repoURL string) (string, error) {
	if err := validateRepoURL(repoURL); err != nil {
		return "", fmt.Errorf("repo resolver: %w", err)
	}

	hash := sha256.Sum256([]byte(repoURL))
	dirName := hex.EncodeToString(hash[:8])
	clonePath := filepath.Join(r.cacheDir, dirName)

	if _, err := os.Stat(filepath.Join(clonePath, "HEAD")); err == nil {
		cmd := exec.CommandContext(ctx, "git", "-C", clonePath, "fetch", "--all")
		if err := cmd.Run(); err != nil {
			log.Printf("repo resolver: fetch failed for %s (using stale cache): %v", dirName, err)
		}
		return clonePath, nil
	}

	if err := os.MkdirAll(r.cacheDir, 0o755); err != nil {
		return "", fmt.Errorf("repo resolver: mkdir: %w", err)
	}

	cmd := exec.CommandContext(ctx, "git", "clone", "--bare", "--", repoURL, clonePath)
	if out, err := cmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("repo resolver: git clone: %s: %w", out, err)
	}
	return clonePath, nil
}
