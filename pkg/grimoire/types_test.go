package grimoire

import (
	"testing"
	"time"
)

func TestGrimoireStruct(t *testing.T) {
	now := time.Now()
	g := Grimoire{
		Version:     1,
		Source:      "https://github.com/test/repo",
		Commit:      "abc123",
		InstalledAt: now,
	}

	if g.Version != 1 {
		t.Errorf("Grimoire.Version = %d, want 1", g.Version)
	}
	if g.Source != "https://github.com/test/repo" {
		t.Errorf("Grimoire.Source = %q, want %q", g.Source, "https://github.com/test/repo")
	}
	if g.Commit != "abc123" {
		t.Errorf("Grimoire.Commit = %q, want %q", g.Commit, "abc123")
	}
	if !g.InstalledAt.Equal(now) {
		t.Errorf("Grimoire.InstalledAt = %v, want %v", g.InstalledAt, now)
	}
}

func TestSkillStruct(t *testing.T) {
	s := Skill{
		Name:        "docker-build",
		Description: "Build Docker images",
		Path:        "/path/to/skill",
		Source:      "https://github.com/test/repo",
		Commit:      "def456",
	}

	if s.Name != "docker-build" {
		t.Errorf("Skill.Name = %q, want %q", s.Name, "docker-build")
	}
	if s.Description != "Build Docker images" {
		t.Errorf("Skill.Description = %q, want %q", s.Description, "Build Docker images")
	}
	if s.Path != "/path/to/skill" {
		t.Errorf("Skill.Path = %q, want %q", s.Path, "/path/to/skill")
	}
	if s.Source != "https://github.com/test/repo" {
		t.Errorf("Skill.Source = %q, want %q", s.Source, "https://github.com/test/repo")
	}
	if s.Commit != "def456" {
		t.Errorf("Skill.Commit = %q, want %q", s.Commit, "def456")
	}
}

func TestInstalledSkillStruct(t *testing.T) {
	g := Grimoire{
		Version: 1,
		Source:  "https://github.com/test/repo",
		Commit:  "abc123",
	}

	is := InstalledSkill{
		Name:     "test-skill",
		Path:     "/targets/my-agent/test-skill",
		Grimoire: g,
		Target:   "my-agent",
	}

	if is.Name != "test-skill" {
		t.Errorf("InstalledSkill.Name = %q, want %q", is.Name, "test-skill")
	}
	if is.Path != "/targets/my-agent/test-skill" {
		t.Errorf("InstalledSkill.Path = %q, want %q", is.Path, "/targets/my-agent/test-skill")
	}
	if is.Grimoire != g {
		t.Errorf("InstalledSkill.Grimoire = %+v, want %+v", is.Grimoire, g)
	}
	if is.Target != "my-agent" {
		t.Errorf("InstalledSkill.Target = %q, want %q", is.Target, "my-agent")
	}
}
