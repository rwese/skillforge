package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeRepoSkill creates a cached repo with a single skill under
// <repoCache>/skills/<skillName>/SKILL.md and writes a config entry
// pointing at the cache. description becomes the body of SKILL.md.
func writeRepoSkill(t *testing.T, env *baselineEnv, repoName, repoURL, skillName, description string) {
	t.Helper()
	repoCache := filepath.Join(env.tmpDir, "cache", repoName)
	skillDir := filepath.Join(repoCache, "skills")
	if nested := strings.Contains(skillName, "/"); nested {
		skillDir = filepath.Join(skillDir, filepath.FromSlash(skillName))
	} else {
		skillDir = filepath.Join(skillDir, skillName)
	}
	if err := os.MkdirAll(skillDir, 0755); err != nil {
		t.Fatalf("MkdirAll(skillDir=%s): %v", skillDir, err)
	}
	body := "# " + filepath.Base(skillName) + "\n\n" + description + "\n"
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(body), 0644); err != nil {
		t.Fatalf("WriteFile(SKILL.md): %v", err)
	}

	env.writeGlobalConfig(t, `
cache.path = "`+filepath.Join(env.tmpDir, "cache")+`"

[repos.`+repoName+`]
url = "`+repoURL+`"
branch = "main"
`)
}

func TestSkillLoadFlatSkill(t *testing.T) {
	env := newBaselineEnv(t)
	t.Chdir(env.tmpDir)

	writeRepoSkill(t, env, "grimoire", "https://example.com/grimoire", "docker", "Build images with docker")

	stdout, stderr, code := env.run("skill", "load", "docker")
	if code != 0 {
		t.Fatalf("expected load to succeed, got code=%d\nstdout=%s\nstderr=%s", code, stdout, stderr)
	}

	// Locate the printed temp dir line
	var tempDir string
	for _, line := range strings.Split(stdout, "\n") {
		if strings.HasPrefix(line, "Skill loaded at: ") {
			tempDir = strings.TrimPrefix(line, "Skill loaded at: ")
			break
		}
	}
	if tempDir == "" {
		t.Fatalf("expected 'Skill loaded at:' line in stdout, got:\n%s", stdout)
	}
	// Layout: /tmp/skillforge-XXXXXX/docker/
	// The hash segment is xxx-xxx (3-3 hex chars) so the dir name is
	// "skillforge-XXXXXX" and the skill sits at <tempDir>/docker.
	if filepath.Base(tempDir) != "docker" {
		t.Errorf("expected tempDir to end with /docker, got %q", tempDir)
	}
	parent := filepath.Base(filepath.Dir(tempDir))
	if !strings.HasPrefix(parent, "skillforge-") {
		t.Errorf("expected parent dir to start with 'skillforge-', got %q", parent)
	}

	// SKILL.md body printed under the "SKILL.md:" header
	if !strings.Contains(stdout, "Build images with docker") {
		t.Errorf("expected SKILL.md contents in stdout, got:\n%s", stdout)
	}

	// Sanity: the copied file exists on disk and matches
	copied, err := os.ReadFile(filepath.Join(tempDir, "SKILL.md"))
	if err != nil {
		t.Fatalf("read copied SKILL.md: %v", err)
	}
	if !strings.Contains(string(copied), "Build images with docker") {
		t.Errorf("copied SKILL.md body = %q, want it to contain description", copied)
	}
}

func TestSkillLoadNestedSkillLayout(t *testing.T) {
	env := newBaselineEnv(t)
	t.Chdir(env.tmpDir)

	writeRepoSkill(t, env, "grimoire", "https://example.com/grimoire",
		"architecture/event-sourced-commands", "ES + CQRS patterns")

	stdout, stderr, code := env.run("skill", "load", "architecture/event-sourced-commands")
	if code != 0 {
		t.Fatalf("expected load to succeed, got code=%d\nstdout=%s\nstderr=%s", code, stdout, stderr)
	}

	var tempDir string
	for _, line := range strings.Split(stdout, "\n") {
		if strings.HasPrefix(line, "Skill loaded at: ") {
			tempDir = strings.TrimPrefix(line, "Skill loaded at: ")
			break
		}
	}
	if tempDir == "" {
		t.Fatalf("expected 'Skill loaded at:' line in stdout, got:\n%s", stdout)
	}

	// For a nested skill 'a/b', the printed tempDir already ends with
	// /a/b/ — the slash-joined skill name is appended below the
	// skillforge-<hash> parent. SKILL.md must be at <tempDir>/SKILL.md.
	if _, err := os.Stat(filepath.Join(tempDir, "SKILL.md")); err != nil {
		t.Fatalf("expected SKILL.md at %s: %v", tempDir, err)
	}
	if filepath.Base(tempDir) != "event-sourced-commands" {
		t.Errorf("expected tempDir to end with /event-sourced-commands, got %q", tempDir)
	}
	if !strings.Contains(stdout, "ES + CQRS patterns") {
		t.Errorf("expected SKILL.md contents in stdout, got:\n%s", stdout)
	}
}

func TestSkillLoadRecursiveCopy(t *testing.T) {
	env := newBaselineEnv(t)
	t.Chdir(env.tmpDir)

	repoCache := filepath.Join(env.tmpDir, "cache", "grimoire")
	skillDir := filepath.Join(repoCache, "skills", "deep")
	if err := os.MkdirAll(skillDir, 0755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte("# deep\n\nbody\n"), 0644); err != nil {
		t.Fatalf("WriteFile(SKILL.md): %v", err)
	}
	// Add a subdir + nested file so the copy must be recursive.
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

	stdout, stderr, code := env.run("skill", "load", "deep")
	if code != 0 {
		t.Fatalf("expected load to succeed, got code=%d\nstdout=%s\nstderr=%s", code, stdout, stderr)
	}

	var tempDir string
	for _, line := range strings.Split(stdout, "\n") {
		if strings.HasPrefix(line, "Skill loaded at: ") {
			tempDir = strings.TrimPrefix(line, "Skill loaded at: ")
			break
		}
	}
	if tempDir == "" {
		t.Fatalf("expected 'Skill loaded at:' line in stdout, got:\n%s", stdout)
	}
	copiedExtra, err := os.ReadFile(filepath.Join(tempDir, "sub", "extra.txt"))
	if err != nil {
		t.Fatalf("expected sub/extra.txt to be copied recursively: %v", err)
	}
	if string(copiedExtra) != "extra\n" {
		t.Errorf("copied extra.txt = %q, want %q", copiedExtra, "extra\n")
	}
}

func TestSkillLoadMissingSkill(t *testing.T) {
	env := newBaselineEnv(t)
	t.Chdir(env.tmpDir)

	env.writeGlobalConfig(t, `
cache.path = "`+filepath.Join(env.tmpDir, "cache")+`"
`)

	stdout, stderr, code := env.run("skill", "load", "nope")
	if code == 0 {
		t.Fatalf("expected load to fail for missing skill, got code=0\nstdout=%s\nstderr=%s", stdout, stderr)
	}
	if !strings.Contains(stderr, "nope") {
		t.Errorf("expected stderr to mention the skill name, got:\n%s", stderr)
	}
}