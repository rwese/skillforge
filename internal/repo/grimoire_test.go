package repo

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/rwese/skillforge/pkg/grimoire"
)

func TestNewCache(t *testing.T) {
	cache := NewCache("/test/path")
	if cache == nil {
		t.Fatal("NewCache() returned nil")
	}
	if cache.Path != "/test/path" {
		t.Errorf("Cache.Path = %q, want %q", cache.Path, "/test/path")
	}
}

func TestCacheEnsure(t *testing.T) {
	tmpDir := t.TempDir()
	cache := NewCache(filepath.Join(tmpDir, "cache"))

	if err := cache.Ensure(); err != nil {
		t.Fatalf("Ensure() error = %v", err)
	}

	if _, err := os.Stat(cache.Path); os.IsNotExist(err) {
		t.Error("Cache directory not created")
	}
}

func TestCacheExists(t *testing.T) {
	tmpDir := t.TempDir()
	cache := NewCache(filepath.Join(tmpDir, "cache"))

	// Should not exist initially
	if cache.Exists("test-repo") {
		t.Error("Exists() = true for nonexistent repo")
	}

	// Create it
	if err := os.MkdirAll(cache.PathFor("test-repo"), 0755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}

	// Should exist now
	if !cache.Exists("test-repo") {
		t.Error("Exists() = false for existing repo")
	}
}

func TestCachePathFor(t *testing.T) {
	cache := NewCache("/base/cache")
	path := cache.PathFor("my-repo")
	if path != "/base/cache/my-repo" {
		t.Errorf("PathFor() = %q, want %q", path, "/base/cache/my-repo")
	}
}

func TestCacheGetUpdated(t *testing.T) {
	tmpDir := t.TempDir()
	cache := NewCache(tmpDir)

	// Create a fake repo directory
	repoPath := filepath.Join(tmpDir, "test-repo")
	if err := os.MkdirAll(repoPath, 0755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}

	updated, err := cache.GetUpdated("test-repo")
	if err != nil {
		t.Fatalf("GetUpdated() error = %v", err)
	}

	// Should be close to now
	if time.Since(updated) > time.Second {
		t.Errorf("GetUpdated() returned old time: %v", updated)
	}
}

func TestCacheGetUpdatedNotExist(t *testing.T) {
	tmpDir := t.TempDir()
	cache := NewCache(tmpDir)

	_, err := cache.GetUpdated("nonexistent")
	if err == nil {
		t.Error("GetUpdated() should return error for nonexistent repo")
	}
}

func TestCacheRemove(t *testing.T) {
	tmpDir := t.TempDir()
	cache := NewCache(tmpDir)

	// Create a fake repo
	repoPath := filepath.Join(tmpDir, "to-remove")
	if err := os.MkdirAll(repoPath, 0755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}

	// Remove it
	if err := cache.Remove("to-remove"); err != nil {
		t.Fatalf("Remove() error = %v", err)
	}

	// Verify gone
	if _, err := os.Stat(repoPath); !os.IsNotExist(err) {
		t.Error("Repo still exists after Remove()")
	}
}

func TestRepoName(t *testing.T) {
	tests := []struct {
		url  string
		want string
	}{
		{"https://github.com/user/repo", "repo"},
		{"https://github.com/user/repo.git", "repo"},
		{"git@github.com:user/repo.git", "repo"},
		{"https://gitlab.com/org/project/repo", "repo"},
	}

	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			// Use internal repoName function via Cache
			name := repoNameFromURL(tt.url)
			if name != tt.want {
				t.Errorf("repoName(%q) = %q, want %q", tt.url, name, tt.want)
			}
		})
	}
}

// repoNameFromURL is a test helper that mirrors the internal function
func repoNameFromURL(url string) string {
	url = filepath.ToSlash(url)
	url = strings.TrimSuffix(url, ".git")
	parts := strings.Split(url, "/")
	return parts[len(parts)-1]
}

func TestDiscoverSkills(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a skill at root level
	skillPath := filepath.Join(tmpDir, "my-skill")
	if err := os.MkdirAll(skillPath, 0755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(skillPath, "SKILL.md"), []byte("# My Skill\n\nDescription here"), 0644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	// Discover
	skills, err := DiscoverSkills(tmpDir, "https://github.com/test/repo")
	if err != nil {
		t.Fatalf("DiscoverSkills() error = %v", err)
	}

	if len(skills) != 1 {
		t.Fatalf("DiscoverSkills() returned %d skills, want 1", len(skills))
	}

	if skills[0].Name != "my-skill" {
		t.Errorf("Skill.Name = %q, want %q", skills[0].Name, "my-skill")
	}
	if skills[0].Source != "https://github.com/test/repo" {
		t.Errorf("Skill.Source = %q, want %q", skills[0].Source, "https://github.com/test/repo")
	}
}

func TestDiscoverSkillsInSubdirs(t *testing.T) {
	tmpDir := t.TempDir()

	// Create skills in skills/ subdirectory
	skillsDir := filepath.Join(tmpDir, "skills")
	if err := os.MkdirAll(skillsDir, 0755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}

	skillPath := filepath.Join(skillsDir, "sub-skill")
	if err := os.MkdirAll(skillPath, 0755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(skillPath, "SKILL.md"), []byte("A skill in subdir"), 0644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	skills, err := DiscoverSkills(tmpDir, "https://github.com/test/repo")
	if err != nil {
		t.Fatalf("DiscoverSkills() error = %v", err)
	}

	if len(skills) != 1 {
		t.Errorf("DiscoverSkills() returned %d skills, want 1", len(skills))
	}
}

func TestDiscoverSkillsWithGrimoire(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a directory with .grimoire (should be ignored - it's a repo, not skill)
	repoPath := filepath.Join(tmpDir, "cached-repo")
	if err := os.MkdirAll(repoPath, 0755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(repoPath, ".grimoire"), []byte("[grimoire]"), 0644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	skills, err := DiscoverSkills(tmpDir, "https://github.com/test/repo")
	if err != nil {
		t.Fatalf("DiscoverSkills() error = %v", err)
	}

	if len(skills) != 0 {
		t.Errorf("DiscoverSkills() should return 0 for repo with .grimoire, got %d", len(skills))
	}
}

func TestDiscoverSkillsEmptyDir(t *testing.T) {
	tmpDir := t.TempDir()

	skills, err := DiscoverSkills(tmpDir, "https://github.com/test/repo")
	if err != nil {
		t.Fatalf("DiscoverSkills() error = %v", err)
	}

	if len(skills) != 0 {
		t.Errorf("DiscoverSkills() returned %d skills for empty dir, want 0", len(skills))
	}
}

func TestIsSkillDir(t *testing.T) {
	tmpDir := t.TempDir()

	// Create valid skill
	validPath := filepath.Join(tmpDir, "valid-skill")
	if err := os.MkdirAll(validPath, 0755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(validPath, "SKILL.md"), []byte("# Skill"), 0644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	if !isSkillDir(validPath) {
		t.Error("isSkillDir() = false for valid skill")
	}

	// Create invalid (no SKILL.md)
	invalidPath := filepath.Join(tmpDir, "invalid-skill")
	if err := os.MkdirAll(invalidPath, 0755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}

	if isSkillDir(invalidPath) {
		t.Error("isSkillDir() = true for directory without SKILL.md")
	}
}

func TestReadSkillDescription(t *testing.T) {
	tmpDir := t.TempDir()

	skillPath := filepath.Join(tmpDir, "skill")
	if err := os.MkdirAll(skillPath, 0755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}

	// Write SKILL.md with description
	content := `# My Amazing Skill

This is the description line

## More details`
	if err := os.WriteFile(filepath.Join(skillPath, "SKILL.md"), []byte(content), 0644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	desc := readSkillDescription(skillPath)
	if desc != "This is the description line" {
		t.Errorf("readSkillDescription() = %q, want %q", desc, "This is the description line")
	}
}

func TestReadSkillDescriptionOnlyHeaders(t *testing.T) {
	tmpDir := t.TempDir()

	skillPath := filepath.Join(tmpDir, "skill")
	if err := os.MkdirAll(skillPath, 0755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}

	// Write SKILL.md with only headers
	content := `# Title
## Section
`
	if err := os.WriteFile(filepath.Join(skillPath, "SKILL.md"), []byte(content), 0644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	desc := readSkillDescription(skillPath)
	if desc != "" {
		t.Errorf("readSkillDescription() = %q for header-only file, want empty", desc)
	}
}

func TestReadSkillDescriptionLong(t *testing.T) {
	tmpDir := t.TempDir()

	skillPath := filepath.Join(tmpDir, "skill")
	if err := os.MkdirAll(skillPath, 0755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}

	// Write SKILL.md with long description
	longDesc := "This is a very long description that exceeds 100 characters and should be truncated to fit within reasonable limits for display purposes"
	content := `# Skill
` + longDesc + "\n"
	if err := os.WriteFile(filepath.Join(skillPath, "SKILL.md"), []byte(content), 0644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	desc := readSkillDescription(skillPath)
	if len(desc) > 100 {
		t.Errorf("readSkillDescription() returned string of length %d, want <= 100", len(desc))
	}
	if desc[len(desc)-3:] != "..." {
		t.Errorf("readSkillDescription() should end with ... for truncated strings")
	}
}

func TestDiscoverInCache(t *testing.T) {
	tmpDir := t.TempDir()
	cache := NewCache(tmpDir)

	// Create a fake cached repo
	repoPath := filepath.Join(tmpDir, "test-repo")
	if err := os.MkdirAll(repoPath, 0755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}

	// Create a skill in it
	skillPath := filepath.Join(repoPath, "test-skill")
	if err := os.MkdirAll(skillPath, 0755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(skillPath, "SKILL.md"), []byte("A test skill"), 0644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	repos := map[string]string{
		"test-repo": "https://github.com/test/repo",
	}

	result, err := DiscoverInCache(cache, repos)
	if err != nil {
		t.Fatalf("DiscoverInCache() error = %v", err)
	}

	if len(result) != 1 {
		t.Errorf("DiscoverInCache() returned %d repos, want 1", len(result))
	}

	if len(result["test-repo"]) != 1 {
		t.Errorf("DiscoverInCache() returned %d skills, want 1", len(result["test-repo"]))
	}
}

func TestDiscoverInCacheNotExists(t *testing.T) {
	tmpDir := t.TempDir()
	cache := NewCache(tmpDir)

	repos := map[string]string{
		"nonexistent": "https://github.com/test/repo",
	}

	result, err := DiscoverInCache(cache, repos)
	if err != nil {
		t.Fatalf("DiscoverInCache() error = %v", err)
	}

	if len(result) != 0 {
		t.Errorf("DiscoverInCache() should return empty for nonexistent repos")
	}
}

func TestReadGrimoireNotExist(t *testing.T) {
	tmpDir := t.TempDir()
	skillPath := filepath.Join(tmpDir, "nonexistent")

	g, err := ReadGrimoire(skillPath)
	if err != nil {
		t.Fatalf("ReadGrimoire() error = %v", err)
	}
	if g != nil {
		t.Errorf("ReadGrimoire() returned %v, want nil for nonexistent", g)
	}
}

func TestReadWriteGrimoire(t *testing.T) {
	tmpDir := t.TempDir()
	skillPath := filepath.Join(tmpDir, "test-skill")

	// Create the directory
	if err := os.MkdirAll(skillPath, 0755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}

	now := time.Now()
	original := &grimoire.Grimoire{
		Version:     1,
		Source:      "https://github.com/test/repo",
		Commit:      "abc123def456",
		InstalledAt: now,
	}

	// Write grimoire
	if err := WriteGrimoire(skillPath, original); err != nil {
		t.Fatalf("WriteGrimoire() error = %v", err)
	}

	// Read it back
	g, err := ReadGrimoire(skillPath)
	if err != nil {
		t.Fatalf("ReadGrimoire() error = %v", err)
	}
	if g == nil {
		t.Fatal("ReadGrimoire() returned nil")
	}

	if g.Version != original.Version {
		t.Errorf("Version = %d, want %d", g.Version, original.Version)
	}
	if g.Source != original.Source {
		t.Errorf("Source = %q, want %q", g.Source, original.Source)
	}
	if g.Commit != original.Commit {
		t.Errorf("Commit = %q, want %q", g.Commit, original.Commit)
	}
}

func TestInstallSkill(t *testing.T) {
	tmpDir := t.TempDir()

	// Create source skill
	sourcePath := filepath.Join(tmpDir, "source", "my-skill")
	if err := os.MkdirAll(sourcePath, 0755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}

	// Create a file in the source
	testFile := filepath.Join(sourcePath, "test.txt")
	if err := os.WriteFile(testFile, []byte("test content"), 0644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	// Create skill metadata
	skill := grimoire.Skill{
		Name:        "my-skill",
		Description: "Test skill",
		Path:        sourcePath,
		Source:      "https://github.com/test/repo",
		Commit:      "abc123",
	}

	// Install
	targetPath := filepath.Join(tmpDir, "target", "my-skill")
	if err := InstallSkill(skill, targetPath, "abc123"); err != nil {
		t.Fatalf("InstallSkill() error = %v", err)
	}

	// Verify file was copied
	copiedFile := filepath.Join(targetPath, "test.txt")
	data, err := os.ReadFile(copiedFile)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if string(data) != "test content" {
		t.Errorf("Copied file content = %q, want %q", string(data), "test content")
	}

	// Verify grimoire was created
	g, err := ReadGrimoire(targetPath)
	if err != nil {
		t.Fatalf("ReadGrimoire() error = %v", err)
	}
	if g == nil {
		t.Fatal("Grimoire was not created")
	}
	if g.Commit != "abc123" {
		t.Errorf("Grimoire.Commit = %q, want %q", g.Commit, "abc123")
	}
	if g.Source != skill.Source {
		t.Errorf("Grimoire.Source = %q, want %q", g.Source, skill.Source)
	}
}

func TestRemoveSkill(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a skill directory
	skillPath := filepath.Join(tmpDir, "to-remove")
	if err := os.MkdirAll(skillPath, 0755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}

	// Verify it exists
	if _, err := os.Stat(skillPath); os.IsNotExist(err) {
		t.Fatal("Skill directory was not created")
	}

	// Remove it
	if err := RemoveSkill(skillPath); err != nil {
		t.Fatalf("RemoveSkill() error = %v", err)
	}

	// Verify it's gone
	if _, err := os.Stat(skillPath); !os.IsNotExist(err) {
		t.Error("Skill directory still exists after RemoveSkill()")
	}
}

func TestListInstalledSkills(t *testing.T) {
	tmpDir := t.TempDir()

	// Create target directory
	targetPath := filepath.Join(tmpDir, "target")
	if err := os.MkdirAll(targetPath, 0755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}

	// Create two skills with grimoires
	for _, name := range []string{"skill-one", "skill-two"} {
		skillPath := filepath.Join(targetPath, name)
		if err := os.MkdirAll(skillPath, 0755); err != nil {
			t.Fatalf("MkdirAll() error = %v", err)
		}

		g := &grimoire.Grimoire{
			Version: 1,
			Source:  "https://github.com/test/repo",
			Commit:  "abc123",
		}
		if err := WriteGrimoire(skillPath, g); err != nil {
			t.Fatalf("WriteGrimoire() error = %v", err)
		}
	}

	// Create a directory without grimoire (should be ignored)
	noGrimoirePath := filepath.Join(targetPath, "no-grimoire")
	if err := os.MkdirAll(noGrimoirePath, 0755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}

	// List skills
	skills, err := ListInstalledSkills(targetPath)
	if err != nil {
		t.Fatalf("ListInstalledSkills() error = %v", err)
	}

	if len(skills) != 2 {
		t.Errorf("ListInstalledSkills() returned %d skills, want 2", len(skills))
	}

	// Check names
	found := make(map[string]bool)
	for _, s := range skills {
		found[s.Name] = true
	}
	if !found["skill-one"] || !found["skill-two"] {
		t.Errorf("ListInstalledSkills() missing expected skills, found = %v", found)
	}
}

func TestListInstalledSkillsEmpty(t *testing.T) {
	tmpDir := t.TempDir()
	targetPath := filepath.Join(tmpDir, "target")

	skills, err := ListInstalledSkills(targetPath)
	if err != nil {
		t.Fatalf("ListInstalledSkills() error = %v", err)
	}

	if len(skills) != 0 {
		t.Errorf("ListInstalledSkills() returned %d skills, want 0", len(skills))
	}
}

func TestListInstalledSkillsNotExist(t *testing.T) {
	tmpDir := t.TempDir()
	targetPath := filepath.Join(tmpDir, "nonexistent")

	skills, err := ListInstalledSkills(targetPath)
	if err != nil {
		t.Fatalf("ListInstalledSkills() error = %v", err)
	}

	if len(skills) != 0 {
		t.Errorf("ListInstalledSkills() returned %d skills for nonexistent path, want 0", len(skills))
	}
}

func TestInstallSkillNestedDirs(t *testing.T) {
	tmpDir := t.TempDir()

	// Create source with nested structure
	sourcePath := filepath.Join(tmpDir, "source", "nested-skill", "subdir")
	if err := os.MkdirAll(sourcePath, 0755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}

	nestedFile := filepath.Join(sourcePath, "nested.txt")
	if err := os.WriteFile(nestedFile, []byte("nested content"), 0644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	skill := grimoire.Skill{
		Name:        "nested-skill",
		Path:        filepath.Join(tmpDir, "source", "nested-skill"),
		Source:      "https://github.com/test/repo",
		Commit:      "abc123",
	}

	targetPath := filepath.Join(tmpDir, "target", "nested-skill")
	if err := InstallSkill(skill, targetPath, "abc123"); err != nil {
		t.Fatalf("InstallSkill() error = %v", err)
	}

	// Verify nested file
	copiedNested := filepath.Join(targetPath, "subdir", "nested.txt")
	data, err := os.ReadFile(copiedNested)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if string(data) != "nested content" {
		t.Errorf("Nested file content = %q, want %q", string(data), "nested content")
	}
}

func TestCopyFile(t *testing.T) {
	tmpDir := t.TempDir()

	src := filepath.Join(tmpDir, "source.txt")
	dst := filepath.Join(tmpDir, "dest.txt")

	content := []byte("copy test content")
	if err := os.WriteFile(src, content, 0644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	if err := copyFile(src, dst); err != nil {
		t.Fatalf("copyFile() error = %v", err)
	}

	data, err := os.ReadFile(dst)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if string(data) != "copy test content" {
		t.Errorf("Copied content = %q, want %q", string(data), "copy test content")
	}
}

func TestCopyDir(t *testing.T) {
	tmpDir := t.TempDir()

	src := filepath.Join(tmpDir, "source")
	dst := filepath.Join(tmpDir, "dest")

	// Create source structure
	if err := os.MkdirAll(filepath.Join(src, "subdir"), 0755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(src, "file.txt"), []byte("file content"), 0644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(src, "subdir", "nested.txt"), []byte("nested"), 0644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	if err := copyDir(src, dst); err != nil {
		t.Fatalf("copyDir() error = %v", err)
	}

	// Verify files exist
	for _, name := range []string{"file.txt", "subdir/nested.txt"} {
		path := filepath.Join(dst, name)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("Expected file %s not found", name)
		}
	}
}

func TestLinkSkillCreatesAbsoluteSymlink(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a skill source directory.
	src := filepath.Join(tmpDir, "src", "my-skill")
	if err := os.MkdirAll(src, 0755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(src, "SKILL.md"), []byte("hello"), 0644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	// Create a target parent that itself is reached only through a symlink
	// (e.g. agents config might point ~/.pi/skills elsewhere). A relative
	// symlink under such a parent would resolve to the wrong place, so
	// LinkSkill must always use an absolute target.
	parentReal := filepath.Join(tmpDir, "parent-real")
	if err := os.MkdirAll(parentReal, 0755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	parentLink := filepath.Join(tmpDir, "parent-link")
	if err := os.Symlink(parentReal, parentLink); err != nil {
		t.Fatalf("Symlink() error = %v", err)
	}

	targetPath := filepath.Join(parentLink, "my-skill")
	skill := grimoire.Skill{Name: "my-skill", Path: src}

	if err := LinkSkill(skill, targetPath); err != nil {
		t.Fatalf("LinkSkill() error = %v", err)
	}

	linkTarget, err := os.Readlink(targetPath)
	if err != nil {
		t.Fatalf("Readlink() error = %v", err)
	}
	if !filepath.IsAbs(linkTarget) {
		t.Errorf("LinkSkill target = %q, want absolute path", linkTarget)
	}

	// Resolve through the symlinked parent and confirm the link still points
	// at the real source.
	resolved, err := filepath.EvalSymlinks(targetPath)
	if err != nil {
		t.Fatalf("EvalSymlinks() error = %v", err)
	}
	wantResolved, err := filepath.EvalSymlinks(src)
	if err != nil {
		t.Fatalf("EvalSymlinks(src) error = %v", err)
	}
	if resolved != wantResolved {
		t.Errorf("resolved target = %q, want %q", resolved, wantResolved)
	}
}

func TestLinkSkillRejectsExistingTarget(t *testing.T) {
	tmpDir := t.TempDir()

	src := filepath.Join(tmpDir, "src", "my-skill")
	if err := os.MkdirAll(src, 0755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}

	targetPath := filepath.Join(tmpDir, "parent", "my-skill")
	if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	// Pre-create something at the target so LinkSkill must refuse.
	if err := os.WriteFile(targetPath, []byte("x"), 0644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	skill := grimoire.Skill{Name: "my-skill", Path: src}
	if err := LinkSkill(skill, targetPath); err == nil {
		t.Fatal("LinkSkill() error = nil, want error for existing target")
	}
}
