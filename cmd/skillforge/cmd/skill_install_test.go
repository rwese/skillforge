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
// The default branch is pinned to main so clone-based tests (which track
// `main`) are independent of the machine's init.defaultBranch.
func initGitRepo(t *testing.T, dir string) {
	t.Helper()
	cmd := exec.Command("git", "-c", "init.defaultBranch=main", "init", "-q")
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git init failed: %v\n%s", err, out)
	}
}

// TestSkillInstallSourcesFromLocalScope verifies that the installation
// source is always the local scope: even for `-s global` installs, the
// skill is looked up in the local scope's repos first. The --scope flag
// selects the install targets, not the source.
func TestSkillInstallSourcesFromLocalScope(t *testing.T) {
	env := newBaselineEnv(t)
	t.Chdir(env.tmpDir) // auto-restores cwd so the next test can find the project root
	initGitRepo(t, env.tmpDir)

	globalTarget := filepath.Join(env.tmpDir, "agents-global")

	// Global config: target + repo "grimoire" containing demo-skill.
	env.writeGlobalConfig(t, `
[targets.agents]
globalPath = "`+globalTarget+`"
localPath = ".agents/skills"
enabled = true

[repos.grimoire]
url = "https://github.com/rwese/agents-grimoire"
branch = "main"
`)

	// Local config: repo "local-repo" containing the SAME skill name.
	env.writeLocalConfig(t, `
[repos.local-repo]
url = "https://github.com/rwese/local-repo"
branch = "main"
`)

	// Seed the shared cache with both repos, each providing demo-skill.
	globalSkill := filepath.Join(env.homeDir, ".cache", "skillforge", "repos", "grimoire", "skills", "demo-skill")
	if err := os.MkdirAll(globalSkill, 0755); err != nil {
		t.Fatalf("MkdirAll(global skill) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(globalSkill, "SKILL.md"), []byte("# demo-skill (global)\n"), 0644); err != nil {
		t.Fatalf("WriteFile(global SKILL.md) error = %v", err)
	}
	localSkill := filepath.Join(env.homeDir, ".cache", "skillforge", "repos", "local-repo", "skills", "demo-skill")
	if err := os.MkdirAll(localSkill, 0755); err != nil {
		t.Fatalf("MkdirAll(local skill) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(localSkill, "SKILL.md"), []byte("# demo-skill (local)\n"), 0644); err != nil {
		t.Fatalf("WriteFile(local SKILL.md) error = %v", err)
	}

	// Install to the global target: the source must be the local scope's
	// repo even though the target is global.
	stdout, stderr, code := env.run("skill", "install", "-s", "global", "-t", "agents", "demo-skill")
	if code != 0 {
		t.Fatalf("expected install to succeed, got code=%d\nstdout=%s\nstderr=%s", code, stdout, stderr)
	}

	linked := filepath.Join(globalTarget, "demo-skill")
	linkTarget, err := os.Readlink(linked)
	if err != nil {
		t.Fatalf("expected demo-skill symlink at %q: %v", linked, err)
	}
	if !strings.Contains(linkTarget, "local-repo") {
		t.Fatalf("expected skill to be sourced from the local scope repo (local-repo), link points to: %s", linkTarget)
	}
}

// TestSkillInstallLocalFallsBackToGlobalRepos verifies that when the
// local scope has no matching repo (here: no local config at all), the
// global scope's repos still provide the installation source.
func TestSkillInstallLocalFallsBackToGlobalRepos(t *testing.T) {
	env := newBaselineEnv(t)
	t.Chdir(env.tmpDir)
	initGitRepo(t, env.tmpDir)

	env.writeGlobalConfig(t, `
[targets.agents]
globalPath = "/tmp/agents-global"
localPath = ".agents/skills"
enabled = true

[repos.grimoire]
url = "https://github.com/rwese/agents-grimoire"
branch = "main"
`)

	// No local config at all: the skill must still be found in the
	// global scope's repo and installed to the local target.
	repoSkill := filepath.Join(env.homeDir, ".cache", "skillforge", "repos", "grimoire", "skills", "demo-skill")
	if err := os.MkdirAll(repoSkill, 0755); err != nil {
		t.Fatalf("MkdirAll(repo skill) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(repoSkill, "SKILL.md"), []byte("# demo-skill\n"), 0644); err != nil {
		t.Fatalf("WriteFile(SKILL.md) error = %v", err)
	}

	stdout, stderr, code := env.run("skill", "install", "-s", "local", "-t", "agents", "demo-skill")
	if code != 0 {
		t.Fatalf("expected install to succeed, got code=%d\nstdout=%s\nstderr=%s", code, stdout, stderr)
	}

	linked := filepath.Join(env.tmpDir, ".agents", "skills", "demo-skill")
	if _, err := os.Stat(linked); err != nil {
		t.Fatalf("expected skill to be linked at %q: %v", linked, err)
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

// TestSkillInstallLocalTargetOutsideGitRootMentionsGitRoot is a regression test
// for the bug where `skillforge skill install -s local -t <target>` outside a
// git repository returned the misleading error:
//
//	Error: target "<target>" not found or not enabled for scope "local"
//
// even though `target list` reported the target as enabled with a `localPath`.
//
// Root cause: `getInstallPaths` short-circuited the local target lookup whenever
// `localGitRoot == ""`, so a configured-and-enabled target looked
// indistinguishable from one that did not exist. The user had no way to tell
// that they needed to run the command from inside a git repository (where
// relative `localPath` values are resolved).
//
// Fix: when a specific target is requested, the local scope is requested, and
// the cwd is not inside a git repository, detect the case where the target IS
// configured and enabled for local scope and return a clearer error that
// mentions the git repository requirement.
func TestSkillInstallLocalTargetOutsideGitRootMentionsGitRoot(t *testing.T) {
	env := newBaselineEnv(t)
	t.Chdir(env.tmpDir) // tmpDir is NOT a git repo, so DetectGitRoot returns ""

	// Global config defines an enabled local target with a relative localPath.
	env.writeGlobalConfig(t, `
[targets.agents]
globalPath = "/tmp/agents-global"
localPath = ".agents/skills"
enabled = true
`)

	_, stderr, code := env.run(
		"skill", "install",
		"-s", "local",
		"-t", "agents",
		"pi-extension-development",
	)

	if code == 0 {
		t.Fatalf("expected install to fail outside git root, got code=0\nstderr=%s", stderr)
	}

	// The error must NOT be the misleading "not found or not enabled".
	// That message is reserved for targets that genuinely don't exist or
	// are disabled; here the target is configured and enabled.
	if strings.Contains(stderr, "not found or not enabled for scope") {
		t.Fatalf("regression: error misleadingly says target is not found or not enabled\nstderr=%s", stderr)
	}

	// The error must explain the actual reason: cwd is not in a git repo.
	if !strings.Contains(stderr, "git repository") {
		t.Fatalf("error should mention git repository requirement, got: %s", stderr)
	}

	// The error must mention the failing target name so the user knows
	// which target they were trying to use.
	if !strings.Contains(stderr, `"agents"`) {
		t.Fatalf("error should mention the target name %q, got: %s", "agents", stderr)
	}
}

// TestSkillInstallLocalTargetOutsideGitRootKeepsGenericErrorForUnknownTarget
// is the companion regression: when the target does NOT exist, the original
// "not found or not enabled" error must still be returned, even outside a git
// repo. The new git-root error must only fire when the target is actually
// configured.
func TestSkillInstallLocalTargetOutsideGitRootKeepsGenericErrorForUnknownTarget(t *testing.T) {
	env := newBaselineEnv(t)
	t.Chdir(env.tmpDir)

	env.writeGlobalConfig(t, `
[targets.agents]
globalPath = "/tmp/agents-global"
localPath = ".agents/skills"
enabled = true
`)

	_, stderr, code := env.run(
		"skill", "install",
		"-s", "local",
		"-t", "nonexistent",
		"pi-extension-development",
	)

	if code == 0 {
		t.Fatalf("expected install to fail for unknown target, got code=0\nstderr=%s", stderr)
	}

	if !strings.Contains(stderr, "not found or not enabled for scope") {
		t.Fatalf("expected generic not-found error for unknown target, got: %s", stderr)
	}
}

// TestSkillInstallLocalAbsoluteLocalPathWorksOutsideGitRoot documents and
// verifies that an absolute `localPath` does not require a git root. The git
// root is only needed to resolve *relative* local paths; absolute paths are
// already self-locating.
func TestSkillInstallLocalAbsoluteLocalPathWorksOutsideGitRoot(t *testing.T) {
	env := newBaselineEnv(t)
	t.Chdir(env.tmpDir)

	// Pre-create a writable absolute local target directory under tmpDir
	// so the test does not pollute the user's home and so the link target
	// is obviously throwaway.
	absoluteLocal := filepath.Join(env.tmpDir, "absolute-local-skills")

	env.writeGlobalConfig(t, `
[targets.agents]
globalPath = "/tmp/agents-global"
localPath = "`+absoluteLocal+`"
enabled = true

[repos.grimoire]
url = "https://github.com/rwese/agents-grimoire"
branch = "main"
`)

	// Drop a fake skill into the default cache (the effective cache path
	// when neither global nor local config overrides `cache.path`).
	defaultCache := filepath.Join(env.homeDir, ".cache", "skillforge", "repos", "grimoire")
	skillDir := filepath.Join(defaultCache, "skills", "pi-extension-development")
	if err := os.MkdirAll(skillDir, 0755); err != nil {
		t.Fatalf("MkdirAll(skill) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte("# pi-extension-development\n"), 0644); err != nil {
		t.Fatalf("WriteFile(SKILL.md) error = %v", err)
	}

	_, stderr, code := env.run(
		"skill", "install",
		"-s", "local",
		"-t", "agents",
		"pi-extension-development",
	)

	if code != 0 {
		t.Fatalf("expected install to succeed with absolute localPath outside git root, got code=%d\nstderr=%s", code, stderr)
	}

	linked := filepath.Join(absoluteLocal, "pi-extension-development")
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
