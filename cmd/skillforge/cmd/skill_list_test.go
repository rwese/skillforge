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

func TestResolveLocalSkillDirUsesGitRootForRelativePaths(t *testing.T) {
	got := resolveLocalSkillDir(".codex/skills", "/repo/root")
	want := filepath.Join("/repo/root", ".codex", "skills")

	if got != want {
		t.Fatalf("resolveLocalSkillDir() = %q, want %q", got, want)
	}
}

func TestResolveLocalSkillDirKeepsAbsolutePaths(t *testing.T) {
	got := resolveLocalSkillDir("/absolute/skills", "/repo/root")
	if got != "/absolute/skills" {
		t.Fatalf("resolveLocalSkillDir() = %q, want %q", got, "/absolute/skills")
	}
}

func TestLocalPathResolvableRelativeWithoutGitRoot(t *testing.T) {
	// Relative local paths cannot be resolved when the cwd is not inside a
	// git repository — there is no base directory for the resolver.
	got := localPathResolvable(config.Target{
		Enabled:   true,
		LocalPath: ".agents/skills",
	}, "")
	if got {
		t.Fatalf("localPathResolvable() = true, want false for relative localPath without git root")
	}
}

func TestLocalPathResolvableRelativeWithGitRoot(t *testing.T) {
	got := localPathResolvable(config.Target{
		Enabled:   true,
		LocalPath: ".agents/skills",
	}, "/repo/root")
	if !got {
		t.Fatalf("localPathResolvable() = false, want true for relative localPath with git root")
	}
}

func TestLocalPathResolvableAbsoluteWithoutGitRoot(t *testing.T) {
	// Absolute local paths are self-locating, so they do not need a git
	// root. This is the case that lets `skill install -s local` succeed
	// from a non-git directory when the target uses an absolute localPath.
	got := localPathResolvable(config.Target{
		Enabled:   true,
		LocalPath: "/abs/.agents/skills",
	}, "")
	if !got {
		t.Fatalf("localPathResolvable() = false, want true for absolute localPath without git root")
	}
}

func TestLocalPathResolvableDisabled(t *testing.T) {
	got := localPathResolvable(config.Target{
		Enabled:   false,
		LocalPath: ".agents/skills",
	}, "/repo/root")
	if got {
		t.Fatalf("localPathResolvable() = true, want false for disabled target")
	}
}

func TestLocalPathResolvableMissingLocalPath(t *testing.T) {
	got := localPathResolvable(config.Target{
		Enabled:   true,
		LocalPath: "",
	}, "/repo/root")
	if got {
		t.Fatalf("localPathResolvable() = true, want false for target without localPath")
	}
}

func TestLocalTargetConfiguredFindsGlobalEntry(t *testing.T) {
	globalCfg := &config.Config{
		Targets: map[string]config.Target{
			"agents": {
				Name:      "agents",
				LocalPath: ".agents/skills",
				Enabled:   true,
			},
		},
	}

	_, ok := localTargetConfigured("agents", nil, false, globalCfg)
	if !ok {
		t.Fatalf("localTargetConfigured() = false, want true for global entry")
	}
}

func TestLocalTargetConfiguredPrefersLocalEntry(t *testing.T) {
	globalCfg := &config.Config{
		Targets: map[string]config.Target{
			"agents": {
				Name:      "agents",
				LocalPath: ".agents/skills",
				Enabled:   true,
			},
		},
	}
	localCfg := &config.Config{
		Targets: map[string]config.Target{
			"agents": {
				Name:      "agents",
				LocalPath: ".custom-agents/skills",
				Enabled:   true,
			},
		},
	}

	got, ok := localTargetConfigured("agents", localCfg, true, globalCfg)
	if !ok {
		t.Fatalf("localTargetConfigured() = false, want true")
	}
	if got.LocalPath != ".custom-agents/skills" {
		t.Fatalf("localTargetConfigured().LocalPath = %q, want %q (local entry should take precedence)", got.LocalPath, ".custom-agents/skills")
	}
}

func TestLocalTargetConfiguredRejectsMissingTarget(t *testing.T) {
	globalCfg := &config.Config{
		Targets: map[string]config.Target{},
	}
	_, ok := localTargetConfigured("nope", nil, false, globalCfg)
	if ok {
		t.Fatalf("localTargetConfigured() = true, want false for missing target")
	}
}

func TestLocalTargetsForScopeUsesGlobalTargetsWithoutLocalConfig(t *testing.T) {
	globalTargets := map[string]config.Target{
		"agents": {
			Name:      "agents",
			LocalPath: ".agents/skills",
			Enabled:   true,
		},
		"codex": {
			Name:      "codex",
			LocalPath: ".codex/skills",
			Enabled:   true,
		},
	}

	targets := localTargetsForScope(globalTargets, nil, false)
	paths := appendLocalInstallPaths(nil, targets, "/repo/root")

	seen := map[string]string{}
	for _, path := range paths {
		seen[path.Label] = path.Path
	}

	if seen["agents (local)"] != filepath.Join("/repo/root", ".agents", "skills") {
		t.Fatalf("agents local path missing from %#v", paths)
	}
	if seen["codex (local)"] != filepath.Join("/repo/root", ".codex", "skills") {
		t.Fatalf("codex local path missing from %#v", paths)
	}
}

func TestLocalTargetsForScopeMergesLocalOverrides(t *testing.T) {
	globalTargets := map[string]config.Target{
		"agents": {
			Name:      "agents",
			LocalPath: ".agents/skills",
			Enabled:   true,
		},
		"codex": {
			Name:      "codex",
			LocalPath: ".codex/skills",
			Enabled:   true,
		},
	}
	localCfg := &config.Config{
		Targets: map[string]config.Target{
			"codex": {
				Name:      "codex",
				LocalPath: ".custom-codex/skills",
				Enabled:   true,
			},
			"pi": {
				Name:      "pi",
				LocalPath: ".pi/skills",
				Enabled:   true,
			},
		},
	}

	targets := localTargetsForScope(globalTargets, localCfg, true)
	paths := appendLocalInstallPaths(nil, targets, "/repo/root")

	seen := map[string]string{}
	for _, path := range paths {
		seen[path.Label] = path.Path
	}

	if seen["agents (local)"] != filepath.Join("/repo/root", ".agents", "skills") {
		t.Fatalf("agents local path missing from %#v", paths)
	}
	if seen["codex (local)"] != filepath.Join("/repo/root", ".custom-codex", "skills") {
		t.Fatalf("codex local override missing from %#v", paths)
	}
	if seen["pi (local)"] != filepath.Join("/repo/root", ".pi", "skills") {
		t.Fatalf("pi local path missing from %#v", paths)
	}
}

func TestAppendLocalRemovePathsUsesResolvedLocalTargets(t *testing.T) {
	targets := map[string]config.Target{
		"agents": {
			Name:      "agents",
			LocalPath: ".agents/skills",
			Enabled:   true,
		},
		"disabled": {
			Name:      "disabled",
			LocalPath: ".disabled/skills",
			Enabled:   false,
		},
	}

	paths := appendLocalRemovePaths(nil, targets, "/repo/root")
	if len(paths) != 1 {
		t.Fatalf("appendLocalRemovePaths() returned %#v, want one enabled target", paths)
	}
	if paths[0].Label != "agents (local)" {
		t.Fatalf("remove label = %q, want %q", paths[0].Label, "agents (local)")
	}
	if paths[0].Path != filepath.Join("/repo/root", ".agents", "skills") {
		t.Fatalf("remove path = %q, want resolved agents path", paths[0].Path)
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
