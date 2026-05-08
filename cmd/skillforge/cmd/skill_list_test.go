package cmd

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/rwese/skillforge/internal/config"
	"github.com/rwese/skillforge/pkg/grimoire"
)

func TestFormatSkillListIncludesRepo(t *testing.T) {
	skills := []SkillOutput{
		{
			Name:   "docker",
			Target: "codex/local",
			Repo:   "agents-grimoire",
			Commit: "abc1234",
		},
	}

	table := formatSkillTable(skills)
	if !strings.Contains(table, "REPO") {
		t.Fatalf("formatSkillTable() missing repo header:\n%s", table)
	}
	if !strings.Contains(table, "agents-grimoire") {
		t.Fatalf("formatSkillTable() missing repo value:\n%s", table)
	}

	compact := formatSkillCompact(skills)
	if !strings.Contains(compact, "repo=agents-grimoire") {
		t.Fatalf("formatSkillCompact() missing repo value: %s", compact)
	}
}

func TestSkillListRepoResolverResolvesFromSource(t *testing.T) {
	cfg := &config.Config{
		Cache: config.CacheConfig{Path: t.TempDir()},
		Repos: map[string]config.RepoInfo{
			"agents-grimoire": {URL: "https://github.com/rwese/agents-grimoire"},
		},
	}

	resolver := newSkillListRepoResolver(cfg, nil)
	got := resolver.Resolve(grimoire.Skill{
		Name:   "docker",
		Source: "https://github.com/rwese/agents-grimoire",
	})

	if got != "agents-grimoire" {
		t.Fatalf("Resolve() = %q, want %q", got, "agents-grimoire")
	}
}

func TestSkillListRepoResolverResolvesFromCachePath(t *testing.T) {
	cachePath := t.TempDir()
	cfg := &config.Config{
		Cache: config.CacheConfig{Path: cachePath},
		Repos: map[string]config.RepoInfo{
			"agents-grimoire": {URL: "https://github.com/rwese/agents-grimoire"},
		},
	}

	resolver := newSkillListRepoResolver(cfg, nil)
	got := resolver.Resolve(grimoire.Skill{
		Name: "docker",
		Path: filepath.Join(cachePath, "agents-grimoire", "skills", "docker"),
	})

	if got != "agents-grimoire" {
		t.Fatalf("Resolve() = %q, want %q", got, "agents-grimoire")
	}
}
