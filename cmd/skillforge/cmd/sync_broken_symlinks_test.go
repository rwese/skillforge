package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestSyncFixBrokenSymlinksReLinksLocalTarget is the regression test for
// the user-reported case: a project is moved (e.g. Devops was moved into
// git.void.cold.at/nope-at/) and the relative symlinks under
// .pi/skills/ become dangling. `sync --fix-broken-symlinks` must find
// the broken symlinks in the local target, look up the matching skill
// in the catalog (the cache is shared, so the repo may live in either
// scope's config), and re-link them with an absolute path.
func TestSyncFixBrokenSymlinksReLinksLocalTarget(t *testing.T) {
	env := newBaselineEnv(t)
	t.Chdir(env.tmpDir) // sync's local-target resolution requires cwd in a git repo
	initGitRepo(t, env.tmpDir)

	// Custom cache.path so the test does not depend on the developer's
	// real ~/.cache/skillforge/repos content.
	customCache := filepath.Join(env.tmpDir, "cache")
	env.writeGlobalConfig(t, `
cache.path = "`+customCache+`"

[targets.pi]
globalPath = "/tmp/agents-global"
localPath = ".pi/skills"
enabled = true
`)

	// Pre-populate the on-disk cache with one repo + one skill, exactly
	// the way `repo add` would have left it. The cache path here is
	// customCache (the effective cache), not the default.
	repoCache := filepath.Join(customCache, "grimoire")
	skillSrc := filepath.Join(repoCache, "skills", "cloudflare-dns")
	if err := os.MkdirAll(skillSrc, 0755); err != nil {
		t.Fatalf("MkdirAll(skill) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(skillSrc, "SKILL.md"), []byte("# cloudflare-dns\n"), 0644); err != nil {
		t.Fatalf("WriteFile(SKILL.md) error = %v", err)
	}

	// Register the repo in the local config so the catalog includes it
	// (mirrors `repo add -s local` flow).
	env.writeLocalConfig(t, `
[repos.grimoire]
url = "https://github.com/rwese/agents-grimoire"
branch = "main"
`)

	// Hand-craft a broken relative symlink in the local target, the
	// way the old LinkSkill would have left it: a relative path that
	// does not resolve from the new project location.
	localTarget := filepath.Join(env.tmpDir, ".pi", "skills")
	if err := os.MkdirAll(localTarget, 0755); err != nil {
		t.Fatalf("MkdirAll(localTarget) error = %v", err)
	}
	brokenLink := filepath.Join(localTarget, "cloudflare-dns")
	// 7 levels up to reach the test tmpDir, then into cache/grimoire/...
	// (the symlink target only needs to NOT resolve; the fix is a
	// re-link to the absolute cached path.)
	if err := os.Symlink(strings.Repeat("../", 7)+"nowhere/grimoire/skills/cloudflare-dns", brokenLink); err != nil {
		t.Fatalf("Symlink() error = %v", err)
	}

	// Sanity: the symlink is currently broken.
	if _, err := os.Stat(brokenLink); err == nil {
		t.Fatal("pre-condition violated: symlink unexpectedly resolves")
	}

	// Run the fix.
	stdout, stderr, code := env.run("sync", "--fix-broken-symlinks")
	if code != 0 {
		t.Fatalf("sync --fix-broken-symlinks failed: code=%d\nstdout=%s\nstderr=%s", code, stdout, stderr)
	}

	// The symlink must now resolve to the cached skill.
	resolved, err := filepath.EvalSymlinks(brokenLink)
	if err != nil {
		t.Fatalf("EvalSymlinks() error = %v (symlink is still broken):\nstdout=%s\nstderr=%s", err, stdout, stderr)
	}
	wantResolved, err := filepath.EvalSymlinks(skillSrc)
	if err != nil {
		t.Fatalf("EvalSymlinks(skillSrc) error = %v", err)
	}
	if resolved != wantResolved {
		t.Errorf("resolved symlink target = %q, want %q", resolved, wantResolved)
	}

	// And the symlink string itself must be an absolute path.
	linkTarget, err := os.Readlink(brokenLink)
	if err != nil {
		t.Fatalf("Readlink() error = %v", err)
	}
	if !filepath.IsAbs(linkTarget) {
		t.Errorf("re-linked symlink target = %q, want absolute", linkTarget)
	}
}

// TestSyncFixBrokenSymlinksDryRunDoesNotMutate verifies that without
// --fix the symlink is only reported, not replaced.
func TestSyncFixBrokenSymlinksDryRunDoesNotMutate(t *testing.T) {
	env := newBaselineEnv(t)
	t.Chdir(env.tmpDir)
	initGitRepo(t, env.tmpDir)

	customCache := filepath.Join(env.tmpDir, "cache")
	env.writeGlobalConfig(t, `
cache.path = "`+customCache+`"

[targets.pi]
localPath = ".pi/skills"
enabled = true
`)

	repoCache := filepath.Join(customCache, "grimoire")
	skillSrc := filepath.Join(repoCache, "skills", "cloudflare-dns")
	if err := os.MkdirAll(skillSrc, 0755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(skillSrc, "SKILL.md"), []byte("# cloudflare-dns\n"), 0644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	env.writeLocalConfig(t, `
[repos.grimoire]
url = "https://github.com/rwese/agents-grimoire"
branch = "main"
`)

	localTarget := filepath.Join(env.tmpDir, ".pi", "skills")
	if err := os.MkdirAll(localTarget, 0755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	brokenLink := filepath.Join(localTarget, "cloudflare-dns")
	originalTarget := strings.Repeat("../", 7) + "nowhere/grimoire/skills/cloudflare-dns"
	if err := os.Symlink(originalTarget, brokenLink); err != nil {
		t.Fatalf("Symlink() error = %v", err)
	}

	stdout, _, code := env.run("sync")
	if code != 0 {
		t.Fatalf("sync (check) failed: code=%d\nstdout=%s", code, stdout)
	}

	// The symlink must still be the original (broken) one.
	_ = stdout // already dumped above on failure
	linkTarget, err := os.Readlink(brokenLink)
	if err != nil {
		t.Fatalf("Readlink() error = %v", err)
	}
	if linkTarget != originalTarget {
		t.Errorf("sync (no fix) mutated symlink: target = %q, want %q", linkTarget, originalTarget)
	}
	if _, err := os.Stat(brokenLink); err == nil {
		t.Error("sync (no fix) resolved a symlink that should remain broken")
	}
}

// TestSyncFixBrokenSymlinksReportsOrphaned covers the documented safe
// default: a broken symlink whose name matches no skill in the cache
// is reported but left in place.
func TestSyncFixBrokenSymlinksReportsOrphaned(t *testing.T) {
	env := newBaselineEnv(t)
	t.Chdir(env.tmpDir)
	initGitRepo(t, env.tmpDir)

	env.writeGlobalConfig(t, `
[targets.pi]
localPath = ".pi/skills"
enabled = true
`)

	// No local config repos: the catalog is empty, so every broken
	// symlink is an orphan.
	localTarget := filepath.Join(env.tmpDir, ".pi", "skills")
	if err := os.MkdirAll(localTarget, 0755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	orphanLink := filepath.Join(localTarget, "ghost-skill")
	if err := os.Symlink("/nope/ghost-skill", orphanLink); err != nil {
		t.Fatalf("Symlink() error = %v", err)
	}

	stdout, _, code := env.run("sync", "--fix-broken-symlinks")
	if code != 0 {
		t.Fatalf("sync --fix-broken-symlinks failed: code=%d", code)
	}

	if !strings.Contains(stdout, "ghost-skill") {
		t.Errorf("orphan not reported in output:\n%s", stdout)
	}
	if !strings.Contains(stdout, "orphaned") {
		t.Errorf("orphan label missing from output:\n%s", stdout)
	}

	// The orphan must still exist (left in place).
	if _, err := os.Lstat(orphanLink); err != nil {
		t.Errorf("orphan symlink was removed: %v", err)
	}
}
