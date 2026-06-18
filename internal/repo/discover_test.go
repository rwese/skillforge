package repo

import (
	"os"
	"path/filepath"
	"sort"
	"testing"
)

// writeSkillFile creates a directory with a SKILL.md at path.
// description is written as the first non-heading line in the file.
func writeSkillFile(t *testing.T, path, description string) {
	t.Helper()
	if err := os.MkdirAll(path, 0755); err != nil {
		t.Fatalf("MkdirAll(%s): %v", path, err)
	}
	content := "# " + filepath.Base(path) + "\n\n" + description + "\n"
	if err := os.WriteFile(filepath.Join(path, "SKILL.md"), []byte(content), 0644); err != nil {
		t.Fatalf("WriteFile(SKILL.md) in %s: %v", path, err)
	}
}

// TestDiscoverSkillsFlatUnderSkillsRoot verifies the conventional
// flat layout (skills/<name>/SKILL.md) is reported with Skill.Name
// equal to the bare directory name.
func TestDiscoverSkillsFlatUnderSkillsRoot(t *testing.T) {
	tmpDir := t.TempDir()
	writeSkillFile(t, filepath.Join(tmpDir, "skills", "docker"), "Build images")

	skills, err := DiscoverSkills(tmpDir, "https://example.com/repo")
	if err != nil {
		t.Fatalf("DiscoverSkills: %v", err)
	}
	if len(skills) != 1 {
		t.Fatalf("DiscoverSkills returned %d skills, want 1: %#v", len(skills), skills)
	}
	if skills[0].Name != "docker" {
		t.Errorf("Skill.Name = %q, want %q", skills[0].Name, "docker")
	}
}

// TestDiscoverSkillsNestedUnderSkillsRoot is the regression test for
// nested-skill support: a skill at skills/<category>/<name>/ must be
// discovered with Skill.Name = "<category>/<name>".
func TestDiscoverSkillsNestedUnderSkillsRoot(t *testing.T) {
	tmpDir := t.TempDir()
	writeSkillFile(t, filepath.Join(tmpDir, "skills", "architecture", "event-sourced-commands"), "ES+CQRS")

	skills, err := DiscoverSkills(tmpDir, "https://example.com/repo")
	if err != nil {
		t.Fatalf("DiscoverSkills: %v", err)
	}
	if len(skills) != 1 {
		t.Fatalf("DiscoverSkills returned %d skills, want 1: %#v", len(skills), skills)
	}
	if skills[0].Name != "architecture/event-sourced-commands" {
		t.Errorf("Skill.Name = %q, want %q", skills[0].Name, "architecture/event-sourced-commands")
	}
	// Path should still be the on-disk directory, not the relative name.
	wantPath := filepath.Join(tmpDir, "skills", "architecture", "event-sourced-commands")
	if skills[0].Path != wantPath {
		t.Errorf("Skill.Path = %q, want %q", skills[0].Path, wantPath)
	}
}

// TestDiscoverSkillsUnderAgentsSkillsRoot verifies that skills placed
// under .agents/skills/ (the cache-mode layout) are discovered with
// the same name semantics as skills under skills/.
func TestDiscoverSkillsUnderAgentsSkillsRoot(t *testing.T) {
	tmpDir := t.TempDir()
	writeSkillFile(t, filepath.Join(tmpDir, ".agents", "skills", "langfuse"), "Tracing")

	skills, err := DiscoverSkills(tmpDir, "https://example.com/repo")
	if err != nil {
		t.Fatalf("DiscoverSkills: %v", err)
	}
	if len(skills) != 1 {
		t.Fatalf("DiscoverSkills returned %d skills, want 1: %#v", len(skills), skills)
	}
	if skills[0].Name != "langfuse" {
		t.Errorf("Skill.Name = %q, want %q", skills[0].Name, "langfuse")
	}
}

// TestDiscoverSkillsNestedUnderAgentsSkillsRoot combines the
// .agents/skills/ root with a nested layout to confirm both work
// together.
func TestDiscoverSkillsNestedUnderAgentsSkillsRoot(t *testing.T) {
	tmpDir := t.TempDir()
	writeSkillFile(t, filepath.Join(tmpDir, ".agents", "skills", "ops", "deploy"), "Deploy ops")

	skills, err := DiscoverSkills(tmpDir, "https://example.com/repo")
	if err != nil {
		t.Fatalf("DiscoverSkills: %v", err)
	}
	if len(skills) != 1 {
		t.Fatalf("DiscoverSkills returned %d skills, want 1: %#v", len(skills), skills)
	}
	if skills[0].Name != "ops/deploy" {
		t.Errorf("Skill.Name = %q, want %q", skills[0].Name, "ops/deploy")
	}
}

// TestDiscoverSkillsSkipsGrimoireDir ensures a directory that has
// BOTH SKILL.md and .grimoire is treated as a repo (not a skill),
// matching the historical isSkillDir behaviour.
func TestDiscoverSkillsSkipsGrimoireDir(t *testing.T) {
	tmpDir := t.TempDir()
	repoDir := filepath.Join(tmpDir, "skills", "subrepo")
	if err := os.MkdirAll(repoDir, 0755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	// The directory contains SKILL.md but is really a repo (has .grimoire).
	if err := os.WriteFile(filepath.Join(repoDir, "SKILL.md"), []byte("# Subrepo"), 0644); err != nil {
		t.Fatalf("WriteFile(SKILL.md): %v", err)
	}
	if err := os.WriteFile(filepath.Join(repoDir, ".grimoire"), []byte("[metadata]\n"), 0644); err != nil {
		t.Fatalf("WriteFile(.grimoire): %v", err)
	}

	skills, err := DiscoverSkills(tmpDir, "https://example.com/repo")
	if err != nil {
		t.Fatalf("DiscoverSkills: %v", err)
	}
	if len(skills) != 0 {
		t.Errorf("DiscoverSkills returned %d skills, want 0 (got %#v)", len(skills), skills)
	}
}

// TestDiscoverSkillsStopsAtSkillBoundary confirms the recursive
// walker does not descend INTO a skill once it is found. If it did,
// an inner SKILL.md at skills/foo/bar/SKILL.md would be discovered
// alongside skills/foo itself, which is incorrect: skills never
// contain other skills.
func TestDiscoverSkillsStopsAtSkillBoundary(t *testing.T) {
	tmpDir := t.TempDir()
	writeSkillFile(t, filepath.Join(tmpDir, "skills", "foo"), "Outer")
	// Inner SKILL.md is a trap: it should NOT be discovered because
	// the outer `foo` is already a skill.
	writeSkillFile(t, filepath.Join(tmpDir, "skills", "foo", "sub"), "Inner (should not be discovered)")

	skills, err := DiscoverSkills(tmpDir, "https://example.com/repo")
	if err != nil {
		t.Fatalf("DiscoverSkills: %v", err)
	}
	if len(skills) != 1 {
		t.Fatalf("DiscoverSkills returned %d skills, want 1: %#v", len(skills), skills)
	}
	if skills[0].Name != "foo" {
		t.Errorf("Skill.Name = %q, want %q", skills[0].Name, "foo")
	}
}

// TestDiscoverSkillsMixedFlatAndNested makes sure the recursive walk
// picks up both flat and nested siblings of the same root in a
// single pass.
func TestDiscoverSkillsMixedFlatAndNested(t *testing.T) {
	tmpDir := t.TempDir()
	writeSkillFile(t, filepath.Join(tmpDir, "skills", "docker"), "Flat")
	writeSkillFile(t, filepath.Join(tmpDir, "skills", "zed"), "Flat")
	writeSkillFile(t, filepath.Join(tmpDir, "skills", "architecture", "event-sourced-commands"), "Nested")

	skills, err := DiscoverSkills(tmpDir, "https://example.com/repo")
	if err != nil {
		t.Fatalf("DiscoverSkills: %v", err)
	}
	got := make([]string, 0, len(skills))
	for _, s := range skills {
		got = append(got, s.Name)
	}
	sort.Strings(got)
	want := []string{"architecture/event-sourced-commands", "docker", "zed"}
	if len(got) != len(want) {
		t.Fatalf("DiscoverSkills returned %d skills, want %d: got=%v", len(got), len(want), got)
	}
	for i, w := range want {
		if got[i] != w {
			t.Errorf("skill[%d].Name = %q, want %q (all: %v)", i, got[i], w, got)
		}
	}
}

// TestDiscoverSkillsLegacyFlatRootFallback ensures skills placed
// directly in cachePath (no skills/ wrapper, as in very old caches)
// are still discoverable with their bare name, so the recursive
// walk does not break historical caches.
func TestDiscoverSkillsLegacyFlatRootFallback(t *testing.T) {
	tmpDir := t.TempDir()
	writeSkillFile(t, filepath.Join(tmpDir, "legacy"), "Old layout")

	skills, err := DiscoverSkills(tmpDir, "https://example.com/repo")
	if err != nil {
		t.Fatalf("DiscoverSkills: %v", err)
	}
	if len(skills) != 1 {
		t.Fatalf("DiscoverSkills returned %d skills, want 1: %#v", len(skills), skills)
	}
	if skills[0].Name != "legacy" {
		t.Errorf("Skill.Name = %q, want %q", skills[0].Name, "legacy")
	}
}

// TestDiscoverSkillsLegacyRootDoesNotDoubleCountSkillsSubdir
// guards against the legacy fallback double-recording skills that
// live under skills/ (which the recursive walk already picked up).
func TestDiscoverSkillsLegacyRootDoesNotDoubleCountSkillsSubdir(t *testing.T) {
	tmpDir := t.TempDir()
	writeSkillFile(t, filepath.Join(tmpDir, "skills", "modern"), "Modern")
	// No skills under the legacy root; the bare entry is "skills"
	// itself and must not be reported as a skill.

	skills, err := DiscoverSkills(tmpDir, "https://example.com/repo")
	if err != nil {
		t.Fatalf("DiscoverSkills: %v", err)
	}
	if len(skills) != 1 {
		t.Fatalf("DiscoverSkills returned %d skills, want 1: %#v", len(skills), skills)
	}
	if skills[0].Name != "modern" {
		t.Errorf("Skill.Name = %q, want %q", skills[0].Name, "modern")
	}
}
