package cmd

import (
	"path/filepath"
	"reflect"
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

func TestResolvedGlobalPathsLegacyGlobalPath(t *testing.T) {
	got := resolvedGlobalPaths(config.Target{GlobalPath: "/legacy/global"})
	want := map[string]string{"default": "/legacy/global"}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("resolvedGlobalPaths() = %#v, want %#v", got, want)
	}
}

func TestResolvedGlobalPathsNamedGlobalPaths(t *testing.T) {
	got := resolvedGlobalPaths(config.Target{
		GlobalPath: "/legacy/global",
		GlobalPaths: map[string]string{
			"default": "/named/default",
			"shared":  "/named/shared",
		},
	})
	want := map[string]string{
		"default": "/named/default",
		"shared":  "/named/shared",
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("resolvedGlobalPaths() = %#v, want %#v", got, want)
	}
}

func TestAppendGlobalInstallPathsUsesAllNamedDirectories(t *testing.T) {
	target := config.Target{
		GlobalPaths: map[string]string{
			"default": "/named/default",
			"shared":  "/named/shared",
		},
	}

	got := appendGlobalInstallPaths(nil, "codex", target)
	seen := map[string]string{}
	for _, path := range got {
		seen[path.Label] = path.Path
	}

	if seen["codex (global:default)"] != "/named/default" {
		t.Fatalf("default install path missing from %#v", got)
	}
	if seen["codex (global:shared)"] != "/named/shared" {
		t.Fatalf("shared install path missing from %#v", got)
	}
}

func TestAppendGlobalRemovePathsUsesAllNamedDirectories(t *testing.T) {
	target := config.Target{
		GlobalPaths: map[string]string{
			"default": "/named/default",
			"shared":  "/named/shared",
		},
	}

	got := appendGlobalRemovePaths(nil, "codex", target)
	seen := map[string]string{}
	for _, path := range got {
		seen[path.Label] = path.Path
	}

	if seen["codex (global:default)"] != "/named/default" {
		t.Fatalf("default remove path missing from %#v", got)
	}
	if seen["codex (global:shared)"] != "/named/shared" {
		t.Fatalf("shared remove path missing from %#v", got)
	}
}

func TestFormatSkillListMultipleGlobalLabels(t *testing.T) {
	skills := []SkillOutput{
		{Name: "docker", Target: "codex/global:default", Repo: "agents-grimoire"},
		{Name: "docker", Target: "codex/global:shared", Repo: "agents-grimoire"},
	}

	table := formatSkillTable(skills)
	if !strings.Contains(table, "codex/global:default") {
		t.Fatalf("formatSkillTable() missing default global label:\n%s", table)
	}
	if !strings.Contains(table, "codex/global:shared") {
		t.Fatalf("formatSkillTable() missing shared global label:\n%s", table)
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
