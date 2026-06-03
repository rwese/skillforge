package repo

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// Cache handles git repository caching.
type Cache struct {
	Path string
}

// NewCache creates a new repository cache.
func NewCache(path string) *Cache {
	return &Cache{Path: path}
}

// Ensure initializes the cache directory.
func (c *Cache) Ensure() error {
	return os.MkdirAll(c.Path, 0755)
}

// Clone clones a repository to the cache.
func (c *Cache) Clone(url, branch string) error {
	if err := c.Ensure(); err != nil {
		return err
	}

	name := repoName(url)
	targetDir := filepath.Join(c.Path, name)

	// Check if already exists
	if _, err := os.Stat(targetDir); err == nil {
		return fmt.Errorf("repository %q already cached", name)
	}

	args := []string{"clone", "--depth", "1"}
	if branch != "main" {
		args = append(args, "-b", branch)
	}
	args = append(args, url, targetDir)

	cmd := exec.Command("git", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// Fetch fetches updates for a cached repository.
func (c *Cache) Fetch(name string) error {
	targetDir := filepath.Join(c.Path, name)
	if _, err := os.Stat(targetDir); os.IsNotExist(err) {
		return fmt.Errorf("repository %q not found in cache", name)
	}

	cmd := exec.Command("git", "-C", targetDir, "fetch", "--depth", "1", "origin")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// Pull pulls updates for a cached repository.
func (c *Cache) Pull(name string) error {
	targetDir := filepath.Join(c.Path, name)
	if _, err := os.Stat(targetDir); os.IsNotExist(err) {
		return fmt.Errorf("repository %q not found in cache", name)
	}

	cmd := exec.Command("git", "-C", targetDir, "pull")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// Remove removes a cached repository.
func (c *Cache) Remove(name string) error {
	targetDir := filepath.Join(c.Path, name)
	return os.RemoveAll(targetDir)
}

// GetCommit gets the current commit hash of a cached repository.
func (c *Cache) GetCommit(name string) (string, error) {
	targetDir := filepath.Join(c.Path, name)
	cmd := exec.Command("git", "-C", targetDir, "rev-parse", "HEAD")
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// GetRemoteCommit gets the current fetched commit hash for origin/<branch>.
func (c *Cache) GetRemoteCommit(name, branch string) (string, error) {
	if branch == "" {
		branch = "main"
	}
	targetDir := filepath.Join(c.Path, name)
	cmd := exec.Command("git", "-C", targetDir, "rev-parse", "origin/"+branch)
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// GetIncomingLog returns one-line commits present in origin/<branch> but not HEAD.
func (c *Cache) GetIncomingLog(name, branch string) (string, error) {
	if branch == "" {
		branch = "main"
	}
	targetDir := filepath.Join(c.Path, name)
	cmd := exec.Command("git", "-C", targetDir, "log", "--oneline", "HEAD..origin/"+branch)
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// GetIncomingNameStatus returns changed files present in origin/<branch> but not HEAD.
func (c *Cache) GetIncomingNameStatus(name, branch string) (string, error) {
	if branch == "" {
		branch = "main"
	}
	targetDir := filepath.Join(c.Path, name)
	cmd := exec.Command("git", "-C", targetDir, "diff", "--name-status", "HEAD..origin/"+branch)
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// GetUpdated gets the last update time of a cached repository.
func (c *Cache) GetUpdated(name string) (time.Time, error) {
	targetDir := filepath.Join(c.Path, name)
	info, err := os.Stat(targetDir)
	if err != nil {
		return time.Time{}, err
	}
	return info.ModTime(), nil
}

// Exists checks if a repository is cached.
func (c *Cache) Exists(name string) bool {
	targetDir := filepath.Join(c.Path, name)
	_, err := os.Stat(targetDir)
	return err == nil
}

// Path returns the full path to a cached repository.
func (c *Cache) PathFor(name string) string {
	return filepath.Join(c.Path, name)
}

// repoName extracts the repository name from a URL.
func repoName(url string) string {
	url = strings.TrimSuffix(url, ".git")
	parts := strings.Split(url, "/")
	return parts[len(parts)-1]
}
