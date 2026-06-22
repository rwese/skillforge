package repo

import (
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// runGit runs git in dir with the given args and fails the test on error.
func runGit(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s in %s: %v\n%s", strings.Join(args, " "), dir, err, out)
	}
	return strings.TrimSpace(string(out))
}

// initBareRemote creates a bare remote at <root>/remote.git that the cache
// can fetch from. The seed repo writes `marker` and commits it as the
// initial commit so the repo is non-empty.
func initBareRemote(t *testing.T, root, marker string) string {
	t.Helper()
	remoteDir := filepath.Join(root, "remote.git")
	seedDir := filepath.Join(root, "seed")

	runGit(t, root, "init", "--bare", "--initial-branch=main", remoteDir)
	runGit(t, root, "init", "--initial-branch=main", seedDir)
	runGit(t, seedDir, "config", "user.email", "test@example.com")
	runGit(t, seedDir, "config", "user.name", "test")
	runGit(t, seedDir, "config", "commit.gpgsign", "false")

	if err := writeFile(filepath.Join(seedDir, marker), "v1\n"); err != nil {
		t.Fatal(err)
	}
	runGit(t, seedDir, "add", marker)
	runGit(t, seedDir, "commit", "-m", "test: initial commit")
	runGit(t, seedDir, "remote", "add", "origin", remoteDir)
	runGit(t, seedDir, "push", "origin", "main")

	return remoteDir
}

// pushCommit adds a new commit to the bare remote.
func pushCommit(t *testing.T, root, marker, body string) string {
	t.Helper()
	seedDir := filepath.Join(root, "seed")
	runGit(t, seedDir, "checkout", "main")
	runGit(t, seedDir, "pull", "--ff-only", "origin", "main")
	if err := writeFile(filepath.Join(seedDir, marker), body); err != nil {
		t.Fatal(err)
	}
	runGit(t, seedDir, "add", marker)
	runGit(t, seedDir, "commit", "-m", "test: update marker")
	runGit(t, seedDir, "push", "origin", "main")
	return runGit(t, seedDir, "rev-parse", "HEAD")
}

// TestPullAdvancesShallowCachePastStaleSnapshot is a regression test for the
// "fatal: Need to specify how to reconcile divergent branches" failure.
//
// Root cause: the cache is cloned with `--depth 1`, and Fetch uses
// `--depth 1`. After the local branch is left behind by a remote commit,
// `git pull` cannot find a merge base between the local shallow branch and
// the freshly-fetched depth-1 commit, so it refuses to do anything.
//
// The fix is that Pull must fast-forward the local branch to the just-fetched
// origin/<branch> tip rather than relying on git's automatic merge logic.
func TestPullAdvancesShallowCachePastStaleSnapshot(t *testing.T) {
	root := t.TempDir()
	remoteURL := initBareRemote(t, root, "marker")
	initialHead := runGit(t, filepath.Join(root, "seed"), "rev-parse", "HEAD")

	cacheDir := t.TempDir()
	cache := NewCache(cacheDir)

	if err := cache.Clone(remoteURL, "main"); err != nil {
		t.Fatalf("Clone: %v", err)
	}

	gotHead, err := cache.GetCommit("remote")
	if err != nil {
		t.Fatalf("GetCommit after clone: %v", err)
	}
	if gotHead != initialHead {
		t.Fatalf("after Clone, HEAD = %s, want %s", gotHead, initialHead)
	}

	// Remote moves forward while the cache stays behind.
	newHead := pushCommit(t, root, "marker", "v2\n")

	if err := cache.Fetch("remote"); err != nil {
		t.Fatalf("Fetch: %v", err)
	}
	// Sanity: Fetch should have made origin/main point at the new commit.
	remoteRef, err := cache.GetRemoteCommit("remote", "main")
	if err != nil {
		t.Fatalf("GetRemoteCommit: %v", err)
	}
	if remoteRef != newHead {
		t.Fatalf("after Fetch, origin/main = %s, want %s", remoteRef, newHead)
	}

	// The bug: this call fails with "Need to specify how to reconcile
	// divergent branches" because the cache is shallow and the freshly
	// fetched depth-1 commit is detached from the local shallow history.
	if err := cache.Pull("remote"); err != nil {
		t.Fatalf("Pull should fast-forward shallow cache to origin/main tip, got: %v", err)
	}

	gotHead, err = cache.GetCommit("remote")
	if err != nil {
		t.Fatalf("GetCommit after pull: %v", err)
	}
	if gotHead != newHead {
		t.Fatalf("after Pull, HEAD = %s, want %s", gotHead, newHead)
	}
}

// TestPullIsIdempotentWhenAlreadyUpToDate ensures that calling Pull on a
// cache that is already at origin/main's tip succeeds and is a no-op.
func TestPullIsIdempotentWhenAlreadyUpToDate(t *testing.T) {
	root := t.TempDir()
	remoteURL := initBareRemote(t, root, "marker")

	cacheDir := t.TempDir()
	cache := NewCache(cacheDir)
	if err := cache.Clone(remoteURL, "main"); err != nil {
		t.Fatalf("Clone: %v", err)
	}

	if err := cache.Fetch("remote"); err != nil {
		t.Fatalf("Fetch: %v", err)
	}
	if err := cache.Pull("remote"); err != nil {
		t.Fatalf("Pull: %v", err)
	}

	head, err := cache.GetCommit("remote")
	if err != nil {
		t.Fatalf("GetCommit: %v", err)
	}
	want := runGit(t, filepath.Join(root, "seed"), "rev-parse", "HEAD")
	if head != want {
		t.Fatalf("HEAD = %s, want %s", head, want)
	}
}