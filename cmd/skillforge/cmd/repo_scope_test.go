package cmd

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// commitInRepo creates an initial commit in the git repo at dir so that
// `git rev-parse HEAD` (used by `repo add` via cache.GetCommit) succeeds.
// Hooks, signing, and identity are bypassed/pinned so the test is
// independent of the machine's git config (core.hooksPath,
// commit.gpgsign, user.name/email).
func commitInRepo(t *testing.T, dir string) {
	t.Helper()
	cmd := exec.Command("git", "add", ".")
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git add failed: %v\n%s", err, out)
	}
	cmd = exec.Command("git", "-c", "user.name=test", "-c", "user.email=test@example.com", "-c", "core.hooksPath=/dev/null", "-c", "commit.gpgsign=false", "commit", "-q", "-m", "test: initial")
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git commit failed: %v\n%s", err, out)
	}
}

// newLocalSourceRepo creates a throwaway git repo with one commit and
// returns a file:// URL to clone from plus the repo name derived from
// the URL (matching repoName()). This keeps repo add/list tests hermetic
// (no network access) and free of the user's global git hooks.
func newLocalSourceRepo(t *testing.T, name string) (string, string) {
	t.Helper()
	dir := filepath.Join(t.TempDir(), name)
	if err := os.MkdirAll(filepath.Join(dir, "skills", "demo-skill"), 0755); err != nil {
		t.Fatalf("MkdirAll(skill) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "skills", "demo-skill", "SKILL.md"), []byte("# demo-skill\n"), 0644); err != nil {
		t.Fatalf("WriteFile(SKILL.md) error = %v", err)
	}
	initGitRepo(t, dir)
	commitInRepo(t, dir)
	return "file://" + dir, name
}

// TestRepo_AddLocalValidWhenAlreadyGlobal verifies that adding a
// repository to the local scope is valid when the same repository is
// already registered in the global scope (and therefore already present
// in the shared on-disk cache). The duplicate check is per scope: the
// repo is registered in the local config and the existing cache entry
// is reused instead of failing with "already cached".
func TestRepo_AddLocalValidWhenAlreadyGlobal(t *testing.T) {
	env := newBaselineEnv(t)
	env.chdir()

	env.writeGlobalConfig(t, `
[repos.grimoire]
url = "https://github.com/rwese/agents-grimoire"
branch = "main"
`)

	// Pre-seed the shared cache exactly as a previous global `repo add`
	// would have: the repo directory (named after the URL) exists in the
	// default cache and is a git repo with a commit.
	repoDir := filepath.Join(env.homeDir, ".cache", "skillforge", "repos", "agents-grimoire")
	if err := os.MkdirAll(repoDir, 0755); err != nil {
		t.Fatalf("MkdirAll(cache repo) error = %v", err)
	}
	initGitRepo(t, repoDir)
	if err := os.WriteFile(filepath.Join(repoDir, "SKILL.md"), []byte("# grimoire\n"), 0644); err != nil {
		t.Fatalf("WriteFile(SKILL.md) error = %v", err)
	}
	commitInRepo(t, repoDir)

	// Adding the same repo to the local scope must succeed. No clone is
	// attempted (the shared cache already has it), so no network access
	// is needed.
	_, stderr, code := env.run("repo", "add", "https://github.com/rwese/agents-grimoire", "-s", "local")
	if code != 0 {
		t.Fatalf("repo add -s local should succeed for a globally-registered repo, got code=%d\nstderr=%s", code, stderr)
	}

	// The repo is now registered in the local scope (under its URL-derived
	// name agents-grimoire)...
	stdout, _, _ := env.run("repo", "list", "-s", "local", "-f", "json")
	if !strings.Contains(stdout, "agents-grimoire") {
		t.Fatalf("expected agents-grimoire in local repo list, got: %s", stdout)
	}
	// ...and still registered in the global scope (under its alias grimoire).
	stdout, _, _ = env.run("repo", "list", "-s", "global", "-f", "json")
	if !strings.Contains(stdout, "grimoire") {
		t.Fatalf("expected grimoire in global repo list, got: %s", stdout)
	}

	// A duplicate within the same scope is still rejected.
	_, stderr, code = env.run("repo", "add", "https://github.com/rwese/agents-grimoire", "-s", "local")
	if code == 0 {
		t.Fatalf("expected duplicate local repo add to fail, got code=0\nstderr=%s", stderr)
	}
	if !strings.Contains(stderr, "already exists in the local scope") {
		t.Fatalf("expected per-scope duplicate error, got: %s", stderr)
	}
}

// TestRepo_AddLocalRejectsDifferentURLWithSameName verifies that the
// shared cache entry is only reused when it was cloned from the same
// URL. A different URL that happens to derive the same repo name must
// not silently reuse the other scope's cached content.
func TestRepo_AddLocalRejectsDifferentURLWithSameName(t *testing.T) {
	env := newBaselineEnv(t)
	env.chdir()

	urlA, _ := newLocalSourceRepo(t, "kb")
	_, stderr, code := env.run("repo", "add", urlA, "-s", "global")
	if code != 0 {
		t.Fatalf("global repo add failed: %s", stderr)
	}

	// Different URL, same derived repo name.
	urlB, _ := newLocalSourceRepo(t, "kb")
	_, stderr, code = env.run("repo", "add", urlB, "-s", "local")
	if code == 0 {
		t.Fatalf("expected same-name/different-URL repo add to fail, got code=0\nstderr=%s", stderr)
	}
	if !strings.Contains(stderr, "different URL") {
		t.Fatalf("expected different-URL error, got: %s", stderr)
	}
}
