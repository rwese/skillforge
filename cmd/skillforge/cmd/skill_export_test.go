package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestSkillExportFlatSkill is the happy-path regression: a flat skill
// is exported to a new directory; the skill files land directly under
// the destination (no skill-name wrapper).
func TestSkillExportFlatSkill(t *testing.T) {
	env := newBaselineEnv(t)
	t.Chdir(env.tmpDir)

	writeRepoSkill(t, env, "grimoire", "https://example.com/grimoire",
		"docker", "Build images with docker")

	destination := filepath.Join(env.tmpDir, "out", "docker-skill")

	stdout, stderr, code := env.run("skill", "export", "docker", destination)
	if code != 0 {
		t.Fatalf("expected export to succeed, got code=%d\nstdout=%s\nstderr=%s", code, stdout, stderr)
	}

	// Destination must have been created and contain SKILL.md directly
	// under it (no skill-name wrapper).
	exported, err := os.ReadFile(filepath.Join(destination, "SKILL.md"))
	if err != nil {
		t.Fatalf("expected SKILL.md at %s: %v", destination, err)
	}
	if !strings.Contains(string(exported), "Build images with docker") {
		t.Errorf("exported SKILL.md body = %q, want it to contain description", exported)
	}

	// Success line mentions the resolved destination.
	if !strings.Contains(stdout, destination) {
		t.Errorf("expected stdout to mention destination %q, got:\n%s", destination, stdout)
	}
}

// TestSkillExportNestedSkill confirms nested skills export with their
// source files landing directly under the destination (NOT inside a
// `<destination>/architecture/...` subdirectory). The slash-joined
// skill name is the internal skill identifier; the user-supplied
// destination is the on-disk layout.
func TestSkillExportNestedSkill(t *testing.T) {
	env := newBaselineEnv(t)
	t.Chdir(env.tmpDir)

	writeRepoSkill(t, env, "grimoire", "https://example.com/grimoire",
		"architecture/event-sourced-commands", "ES + CQRS patterns")

	destination := filepath.Join(env.tmpDir, "out", "es-skill")

	_, stderr, code := env.run("skill", "export",
		"architecture/event-sourced-commands", destination)
	if code != 0 {
		t.Fatalf("expected export to succeed, got code=%d\nstderr=%s", code, stderr)
	}

	// Files must land directly under destination (no
	// <destination>/architecture/event-sourced-commands/ wrapping).
	exported, err := os.ReadFile(filepath.Join(destination, "SKILL.md"))
	if err != nil {
		t.Fatalf("expected SKILL.md at %s/SKILL.md: %v", destination, err)
	}
	if !strings.Contains(string(exported), "ES + CQRS patterns") {
		t.Errorf("exported SKILL.md body = %q, want it to contain description", exported)
	}

	// And the nested wrapper must NOT have been created.
	wrapped := filepath.Join(destination, "architecture", "event-sourced-commands")
	if _, err := os.Stat(wrapped); err == nil {
		t.Errorf("destination must NOT wrap the skill under its category path; found %s", wrapped)
	}
}

// TestSkillExportRecursiveCopy verifies the export copies the full
// skill tree (subdirectories + files), not just SKILL.md.
func TestSkillExportRecursiveCopy(t *testing.T) {
	env := newBaselineEnv(t)
	t.Chdir(env.tmpDir)

	// Build a skill with a subdirectory + nested file.
	repoCache := filepath.Join(env.tmpDir, "cache", "grimoire")
	skillDir := filepath.Join(repoCache, "skills", "deep")
	if err := os.MkdirAll(skillDir, 0755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte("# deep\n\nbody\n"), 0644); err != nil {
		t.Fatalf("WriteFile(SKILL.md): %v", err)
	}
	subDir := filepath.Join(skillDir, "sub")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatalf("MkdirAll(sub): %v", err)
	}
	if err := os.WriteFile(filepath.Join(subDir, "extra.txt"), []byte("extra\n"), 0644); err != nil {
		t.Fatalf("WriteFile(extra.txt): %v", err)
	}

	env.writeGlobalConfig(t, `
cache.path = "`+filepath.Join(env.tmpDir, "cache")+`"

[repos.grimoire]
url = "https://example.com/grimoire"
branch = "main"
`)

	destination := filepath.Join(env.tmpDir, "out", "deep-skill")
	_, stderr, code := env.run("skill", "export", "deep", destination)
	if code != 0 {
		t.Fatalf("expected export to succeed, got code=%d\nstderr=%s", code, stderr)
	}

	copied, err := os.ReadFile(filepath.Join(destination, "sub", "extra.txt"))
	if err != nil {
		t.Fatalf("expected sub/extra.txt to be copied recursively: %v", err)
	}
	if string(copied) != "extra\n" {
		t.Errorf("copied extra.txt = %q, want %q", copied, "extra\n")
	}
}

// TestSkillExportRefusesExistingDirectory: when the destination
// already exists as a directory, the command MUST fail and MUST NOT
// modify the destination's contents.
func TestSkillExportRefusesExistingDirectory(t *testing.T) {
	env := newBaselineEnv(t)
	t.Chdir(env.tmpDir)

	writeRepoSkill(t, env, "grimoire", "https://example.com/grimoire",
		"docker", "Build images with docker")

	destination := filepath.Join(env.tmpDir, "out", "docker-skill")
	if err := os.MkdirAll(destination, 0755); err != nil {
		t.Fatalf("pre-create destination: %v", err)
	}
	// Sentinel file inside the existing destination; if the command
	// wrote into it we'd overwrite this sentinel.
	sentinel := filepath.Join(destination, "sentinel.txt")
	if err := os.WriteFile(sentinel, []byte("untouched\n"), 0644); err != nil {
		t.Fatalf("WriteFile(sentinel): %v", err)
	}

	_, stderr, code := env.run("skill", "export", "docker", destination)
	if code == 0 {
		t.Fatalf("expected export to fail because destination already exists, got code=0\nstderr=%s", stderr)
	}

	// Error must clearly say the destination already exists.
	if !strings.Contains(stderr, "already exists") {
		t.Errorf("expected 'already exists' in error, got:\n%s", stderr)
	}

	// Sentinel must still be untouched.
	got, err := os.ReadFile(sentinel)
	if err != nil {
		t.Fatalf("read sentinel after failed export: %v", err)
	}
	if string(got) != "untouched\n" {
		t.Errorf("sentinel was modified: got %q, want %q", got, "untouched\n")
	}
}

// TestSkillExportRefusesExistingFile: the destination check must fire
// even when the existing entry is a regular file (not a directory).
func TestSkillExportRefusesExistingFile(t *testing.T) {
	env := newBaselineEnv(t)
	t.Chdir(env.tmpDir)

	writeRepoSkill(t, env, "grimoire", "https://example.com/grimoire",
		"docker", "Build images with docker")

	destination := filepath.Join(env.tmpDir, "out", "docker-skill")
	if err := os.MkdirAll(filepath.Dir(destination), 0755); err != nil {
		t.Fatalf("MkdirAll(parent): %v", err)
	}
	if err := os.WriteFile(destination, []byte("regular file\n"), 0644); err != nil {
		t.Fatalf("WriteFile(destination as file): %v", err)
	}

	_, stderr, code := env.run("skill", "export", "docker", destination)
	if code == 0 {
		t.Fatalf("expected export to fail when destination is a regular file, got code=0\nstderr=%s", stderr)
	}
	if !strings.Contains(stderr, "already exists") {
		t.Errorf("expected 'already exists' in error, got:\n%s", stderr)
	}
	// Original file must be untouched.
	got, err := os.ReadFile(destination)
	if err != nil {
		t.Fatalf("read destination after failed export: %v", err)
	}
	if string(got) != "regular file\n" {
		t.Errorf("destination file was modified: got %q", got)
	}
}

// TestSkillExportRefusesExistingSymlink: a symlink at the destination
// (even a broken one) must also trigger the "already exists" failure.
// Lstat-based check ensures broken symlinks are detected.
func TestSkillExportRefusesExistingSymlink(t *testing.T) {
	env := newBaselineEnv(t)
	t.Chdir(env.tmpDir)

	writeRepoSkill(t, env, "grimoire", "https://example.com/grimoire",
		"docker", "Build images with docker")

	destination := filepath.Join(env.tmpDir, "out", "docker-skill")
	if err := os.MkdirAll(filepath.Dir(destination), 0755); err != nil {
		t.Fatalf("MkdirAll(parent): %v", err)
	}
	// Broken symlink: target does not exist.
	if err := os.Symlink(filepath.Join(env.tmpDir, "does-not-exist"), destination); err != nil {
		t.Fatalf("Symlink: %v", err)
	}

	_, stderr, code := env.run("skill", "export", "docker", destination)
	if code == 0 {
		t.Fatalf("expected export to fail when destination is a symlink, got code=0\nstderr=%s", stderr)
	}
	if !strings.Contains(stderr, "already exists") {
		t.Errorf("expected 'already exists' in error, got:\n%s", stderr)
	}
	// Symlink must still be there (untouched).
	if _, err := os.Lstat(destination); err != nil {
		t.Errorf("expected destination symlink to remain after failed export: %v", err)
	}
}

// TestSkillExportMissingSkill: exporting a skill that is not in any
// cached repo must fail with the standard "not found in any cached
// repository" error and a hint.
func TestSkillExportMissingSkill(t *testing.T) {
	env := newBaselineEnv(t)
	t.Chdir(env.tmpDir)

	env.writeGlobalConfig(t, `
cache.path = "`+filepath.Join(env.tmpDir, "cache")+`"
`)

	destination := filepath.Join(env.tmpDir, "out", "nope-skill")

	_, stderr, code := env.run("skill", "export", "nope", destination)
	if code == 0 {
		t.Fatalf("expected export to fail for missing skill, got code=0\nstderr=%s", stderr)
	}
	if !strings.Contains(stderr, "not found in any cached repository") {
		t.Errorf("expected 'not found in any cached repository' in error, got:\n%s", stderr)
	}

	// Destination must NOT have been created on failure.
	if _, err := os.Stat(destination); err == nil {
		t.Errorf("destination must not be created on failure, but %s exists", destination)
	}
}

// TestSkillExportCreatesMissingParents: when the destination's parent
// chain does not exist, the command must create the missing parents
// (and the leaf) before copying.
func TestSkillExportCreatesMissingParents(t *testing.T) {
	env := newBaselineEnv(t)
	t.Chdir(env.tmpDir)

	writeRepoSkill(t, env, "grimoire", "https://example.com/grimoire",
		"docker", "Build images with docker")

	// Two missing parent levels: <tmpDir>/out/deep/docker-skill
	destination := filepath.Join(env.tmpDir, "out", "deep", "docker-skill")

	_, stderr, code := env.run("skill", "export", "docker", destination)
	if code != 0 {
		t.Fatalf("expected export to create missing parents and succeed, got code=%d\nstderr=%s", code, stderr)
	}

	if _, err := os.Stat(filepath.Join(destination, "SKILL.md")); err != nil {
		t.Errorf("expected SKILL.md at %s: %v", destination, err)
	}
}

// TestSkillExportRefusesOverwriteOfSubpath: when a *parent* path
// component of the destination is a regular file (i.e. the
// destination itself is not on disk, but the directory it should
// live under is obstructed by a file), the command must fail
// rather than silently try to clobber the file.
//
// The actual catching path is the ENOTDIR branch in
// `assertExportDestinationMissing` (skill_export.go:120-122), or
// `MkdirAll`'s ENOTDIR — both surface a clear refusal without
// touching the blocker file.
func TestSkillExportRefusesOverwriteOfParentFile(t *testing.T) {
	env := newBaselineEnv(t)
	t.Chdir(env.tmpDir)

	writeRepoSkill(t, env, "grimoire", "https://example.com/grimoire",
		"docker", "Build images with docker")

	// /tmp/out is a regular file, not a directory. The command must
	// not write through it.
	blocker := filepath.Join(env.tmpDir, "out")
	if err := os.MkdirAll(filepath.Dir(blocker), 0755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(blocker, []byte("blocker\n"), 0644); err != nil {
		t.Fatalf("WriteFile(blocker): %v", err)
	}
	destination := filepath.Join(blocker, "docker-skill")

	_, stderr, code := env.run("skill", "export", "docker", destination)
	if code == 0 {
		t.Fatalf("expected export to fail when parent is a regular file, got code=0\nstderr=%s", stderr)
	}
	// Either "already exists" (leaf caught), "not creatable" (parent
	// path is a regular file — surfaces a clean ENOTDIR-style error),
	// or "creating destination" (MkdirAll error) is acceptable. The
	// common requirement is that we refuse and the on-disk state is
	// untouched.
	if !strings.Contains(stderr, "already exists") &&
		!strings.Contains(stderr, "not creatable") &&
		!strings.Contains(stderr, "creating destination") {
		t.Errorf("expected 'already exists', 'not creatable', or 'creating destination' error, got:\n%s", stderr)
	}
	// Blocker file untouched.
	got, err := os.ReadFile(blocker)
	if err != nil {
		t.Fatalf("read blocker: %v", err)
	}
	if string(got) != "blocker\n" {
		t.Errorf("blocker file was modified: got %q", got)
	}
}

// TestSkillExportExpandsTilde: the destination argument supports
// `~`-prefixed paths via the same Expander used elsewhere in the CLI.
func TestSkillExportExpandsTilde(t *testing.T) {
	env := newBaselineEnv(t)
	t.Chdir(env.tmpDir)

	writeRepoSkill(t, env, "grimoire", "https://example.com/grimoire",
		"docker", "Build images with docker")

	// Use a destination inside the temp home so we don't pollute the
	// user's actual $HOME.
	destination := filepath.Join(env.homeDir, "docker-export-test")

	_, stderr, code := env.run("skill", "export", "docker", "~/"+filepath.Base(destination))
	if code != 0 {
		t.Fatalf("expected export with ~ path to succeed, got code=%d\nstderr=%s", code, stderr)
	}
	if _, err := os.Stat(filepath.Join(destination, "SKILL.md")); err != nil {
		t.Errorf("expected SKILL.md at %s: %v", destination, err)
	}
}

// TestSkillExportStripsGrimoire: a `.grimoire` marker nested in a
// subdirectory of the cached skill MUST be stripped from the
// exported tree. `.grimoire` carries install metadata that must
// never leak into an exported artifact, regardless of where in the
// source tree it appears.
//
// Note: a `.grimoire` at the top level of a cached skill directory
// is impossible by construction — `repo.isSkillDir` rejects any
// skill directory containing a top-level `.grimoire` to distinguish
// repos from skills. The interesting case is `.grimoire` nested in
// skill subdirectories.
func TestSkillExportStripsGrimoire(t *testing.T) {
	env := newBaselineEnv(t)
	t.Chdir(env.tmpDir)

	repoCache := filepath.Join(env.tmpDir, "cache", "grimoire")
	skillDir := filepath.Join(repoCache, "skills", "tagged")
	if err := os.MkdirAll(skillDir, 0755); err != nil {
		t.Fatalf("MkdirAll(skillDir): %v", err)
	}
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte("# tagged\n\nbody\n"), 0644); err != nil {
		t.Fatalf("WriteFile(SKILL.md): %v", err)
	}
	// Nested .grimoire in a subdirectory of the cached skill — the
	// realistic case where a `.grimoire` could leak into an export.
	subDir := filepath.Join(skillDir, "sub")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatalf("MkdirAll(sub): %v", err)
	}
	if err := os.WriteFile(filepath.Join(subDir, "nested.txt"), []byte("nested\n"), 0644); err != nil {
		t.Fatalf("WriteFile(sub/nested.txt): %v", err)
	}
	if err := os.WriteFile(filepath.Join(subDir, ".grimoire"), []byte("version = 1\n"), 0644); err != nil {
		t.Fatalf("WriteFile(sub/.grimoire): %v", err)
	}
	// Also exercise the deeper-nested case (sub/sub/.grimoire).
	deepDir := filepath.Join(subDir, "deep")
	if err := os.MkdirAll(deepDir, 0755); err != nil {
		t.Fatalf("MkdirAll(deep): %v", err)
	}
	if err := os.WriteFile(filepath.Join(deepDir, ".grimoire"), []byte("version = 1\n"), 0644); err != nil {
		t.Fatalf("WriteFile(deep/.grimoire): %v", err)
	}

	env.writeGlobalConfig(t, `
cache.path = "`+filepath.Join(env.tmpDir, "cache")+`"

[repos.grimoire]
url = "https://example.com/grimoire"
branch = "main"
`)

	destination := filepath.Join(env.tmpDir, "out", "tagged-skill")
	_, stderr, code := env.run("skill", "export", "tagged", destination)
	if code != 0 {
		t.Fatalf("expected export to succeed, got code=%d\nstderr=%s", code, stderr)
	}

	// Nested .grimoire must be stripped.
	if _, err := os.Lstat(filepath.Join(destination, "sub", ".grimoire")); err == nil {
		t.Errorf("nested .grimoire must be stripped from the exported tree")
	} else if !os.IsNotExist(err) {
		t.Errorf("unexpected error checking nested .grimoire: %v", err)
	}
	// Deeper-nested .grimoire must also be stripped.
	if _, err := os.Lstat(filepath.Join(destination, "sub", "deep", ".grimoire")); err == nil {
		t.Errorf("deeply-nested .grimoire must be stripped from the exported tree")
	} else if !os.IsNotExist(err) {
		t.Errorf("unexpected error checking deeply-nested .grimoire: %v", err)
	}
	// Other files (SKILL.md, nested.txt) must be preserved.
	if _, err := os.Stat(filepath.Join(destination, "SKILL.md")); err != nil {
		t.Errorf("expected SKILL.md to be preserved: %v", err)
	}
	if _, err := os.Stat(filepath.Join(destination, "sub", "nested.txt")); err != nil {
		t.Errorf("expected sub/nested.txt to be preserved: %v", err)
	}
}
