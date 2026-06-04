package cmd

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// initGitRepo runs `git init` in the given directory. Local-scope installs
// only resolve local target paths when the cwd is inside a git repository.
func initGitRepo(t *testing.T, dir string) {
	t.Helper()
	cmd := exec.Command("git", "init", "-q")
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git init failed: %v\n%s", err, out)
	}
}

// TestSkillInstallLocalFindsRepoWhenGlobalHasCustomCachePath is a regression test for
// the bug where `skillforge skill install -s local` failed with
// "skill X not found in any cached repository" when the global config had a
// custom `cache.path` but the local config had no cache override.
//
// Root cause: `runSkillInstall` initialized the cache from
// `globalCfg.Cache.Path` (the global's cache path) and `repo add -s local`
// used `localCfg.Cache.Path` (the default, since the local had no override).
// The two scopes disagreed about where the cache lived, so the skill that
// `repo add` had cloned was invisible to `skill install`.
//
// Fix: cache path resolution is now driven by `config.EffectiveCachePath`,
// which returns: local override > global override > default. All commands
// that read or write the on-disk cache use the same effective path, so
// adding a repo with `-s local` and then installing it from a `-s local`
// scope always agree on the cache location.
func TestSkillInstallLocalFindsRepoWhenGlobalHasCustomCachePath(t *testing.T) {
	env := newBaselineEnv(t)
	t.Chdir(env.tmpDir) // auto-restores cwd so the next test can find the project root
	initGitRepo(t, env.tmpDir)

	// 1. Global config: custom cache path (NOT the default) + target definition
	globalCache := filepath.Join(env.tmpDir, "global-cache")
	env.writeGlobalConfig(t, `
cache.path = "`+globalCache+`"

[targets.agents]
globalPath = "/tmp/agents-global"
localPath = ".agents/skills"
enabled = true
`)

	// 2. Local config: target + repo entry, NO cache.path override
	env.writeLocalConfig(t, `
[targets.agents]
globalPath = "/tmp/agents-global"
localPath = ".agents/skills"
enabled = true

[repos.grimoire]
url = "https://github.com/rwese/agents-grimoire"
branch = "main"
`)

	// 3. Pre-create the cache at the EFFECTIVE cache path (the global
	//    override, since the local has none). This mirrors what
	//    `repo add -s local` would now do, and what `skill install -s local`
	//    must look in.
	repoCache := filepath.Join(globalCache, "grimoire")
	skillDir := filepath.Join(repoCache, "skills", "pi-extension-development")
	if err := os.MkdirAll(skillDir, 0755); err != nil {
		t.Fatalf("MkdirAll(skill) error = %v", err)
	}
	if err := os.WriteFile(
		filepath.Join(skillDir, "SKILL.md"),
		[]byte("# pi-extension-development\n"),
		0644,
	); err != nil {
		t.Fatalf("WriteFile(SKILL.md) error = %v", err)
	}

	// Sanity: the default cache should NOT contain the repo.
	defaultCache := filepath.Join(env.homeDir, ".cache", "skillforge", "repos", "grimoire")
	if _, err := os.Stat(defaultCache); err == nil {
		t.Fatalf("pre-condition violated: repo should not be at default cache %q", defaultCache)
	}

	// 4. Run `skill install -s local -t agents pi-extension-development`
	stdout, stderr, code := env.run(
		"skill", "install",
		"-s", "local",
		"-t", "agents",
		"pi-extension-development",
	)

	if code != 0 {
		t.Fatalf("expected install to succeed, got code=%d\nstdout=%s\nstderr=%s", code, stdout, stderr)
	}

	if strings.Contains(stderr, "not found in any cached repository") {
		t.Fatalf("regression: skill reported as not found in any cached repository\nstderr=%s", stderr)
	}

	// Verify the skill was actually linked into the local target path
	linked := filepath.Join(env.tmpDir, ".agents", "skills", "pi-extension-development")
	if _, err := os.Stat(linked); err != nil {
		t.Fatalf("expected skill to be linked at %q: %v", linked, err)
	}
}

// TestSkillInstallLocalFindsRepoWhenLocalHasCustomCachePath is the symmetric
// regression: the local config sets `cache.path` to a custom location. Adding
// the repo with `-s local` clones it to the local cache; install must look
// in the local cache, not the default or global cache path.
func TestSkillInstallLocalFindsRepoWhenLocalHasCustomCachePath(t *testing.T) {
	env := newBaselineEnv(t)
	t.Chdir(env.tmpDir) // auto-restores cwd so the next test can find the project root
	initGitRepo(t, env.tmpDir)

	// Global: target only, no cache override
	env.writeGlobalConfig(t, `
[targets.agents]
globalPath = "/tmp/agents-global"
localPath = ".agents/skills"
enabled = true
`)

	// Local: cache.path = custom location, plus target and repo
	localCache := filepath.Join(env.tmpDir, "local-cache")
	env.writeLocalConfig(t, `
cache.path = "`+localCache+`"

[targets.agents]
globalPath = "/tmp/agents-global"
localPath = ".agents/skills"
enabled = true

[repos.grimoire]
url = "https://github.com/rwese/agents-grimoire"
branch = "main"
`)

	// Pre-create the cache at the LOCAL cache path (not default)
	repoCache := filepath.Join(localCache, "grimoire")
	skillDir := filepath.Join(repoCache, "skills", "pi-extension-development")
	if err := os.MkdirAll(skillDir, 0755); err != nil {
		t.Fatalf("MkdirAll(skill) error = %v", err)
	}
	if err := os.WriteFile(
		filepath.Join(skillDir, "SKILL.md"),
		[]byte("# pi-extension-development\n"),
		0644,
	); err != nil {
		t.Fatalf("WriteFile(SKILL.md) error = %v", err)
	}

	stdout, stderr, code := env.run(
		"skill", "install",
		"-s", "local",
		"-t", "agents",
		"pi-extension-development",
	)

	if code != 0 {
		t.Fatalf("expected install to succeed, got code=%d\nstdout=%s\nstderr=%s", code, stdout, stderr)
	}

	if strings.Contains(stderr, "not found in any cached repository") {
		t.Fatalf("regression: skill reported as not found in any cached repository\nstderr=%s", stderr)
	}

	// Verify the skill was actually linked
	linked := filepath.Join(env.tmpDir, ".agents", "skills", "pi-extension-development")
	if _, err := os.Stat(linked); err != nil {
		t.Fatalf("expected skill to be linked at %q: %v", linked, err)
	}
}
