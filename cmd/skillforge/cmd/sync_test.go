package cmd

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"github.com/rwese/skillforge/internal/config"
)

func TestSyncFlags(t *testing.T) {
	if syncCmd.Flags().Lookup("fix") != nil {
		t.Fatal("sync command should not expose broad --fix")
	}
	if syncCmd.Flags().Lookup("skip-agent-sync") != nil {
		t.Fatal("sync command should not expose --skip-agent-sync")
	}
	if syncCmd.Flags().Lookup("fix-outofsync-agents") == nil {
		t.Fatal("sync command missing --fix-outofsync-agents flag")
	}
	if syncCmd.Flags().Lookup("fix-sync-repos") == nil {
		t.Fatal("sync command missing --fix-sync-repos flag")
	}
	if syncCmd.Flags().Lookup("fix-all") == nil {
		t.Fatal("sync command missing --fix-all flag")
	}
	if syncCmd.Flags().Lookup("fix-broken-symlinks") == nil {
		t.Fatal("sync command missing --fix-broken-symlinks flag")
	}
	if syncCmd.Flags().Lookup("diff") == nil {
		t.Fatal("sync command missing --diff flag")
	}
	if syncCmd.Flags().Lookup("check") != nil {
		t.Fatal("sync command should not expose --check")
	}
	if syncCmd.Flags().Lookup("scope") != nil {
		t.Fatal("sync command should not expose --scope")
	}
}

func TestSyncRejectsScopeFlag(t *testing.T) {
	origScope := scopeFlag
	defer func() { scopeFlag = origScope }()

	scopeFlag = "global"
	if err := rejectSyncScopeFlag(syncCmd); err == nil {
		t.Fatal("expected sync scope rejection")
	}
}

type fakeSyncRepoCache struct {
	exists       map[string]bool
	localCommit  map[string]string
	remoteCommit map[string]string
	incomingLog  map[string]string
	nameStatus   map[string]string
	fetches      []string
	pulls        []string
	clones       []string
}

func newFakeSyncRepoCache() *fakeSyncRepoCache {
	return &fakeSyncRepoCache{
		exists:       make(map[string]bool),
		localCommit:  make(map[string]string),
		remoteCommit: make(map[string]string),
		incomingLog:  make(map[string]string),
		nameStatus:   make(map[string]string),
	}
}

func (f *fakeSyncRepoCache) Exists(name string) bool {
	return f.exists[name]
}

func (f *fakeSyncRepoCache) Clone(url, branch string) error {
	f.clones = append(f.clones, fmt.Sprintf("%s@%s", url, branch))
	return nil
}

func (f *fakeSyncRepoCache) Fetch(name string) error {
	f.fetches = append(f.fetches, name)
	return nil
}

func (f *fakeSyncRepoCache) Pull(name string) error {
	f.pulls = append(f.pulls, name)
	f.localCommit[name] = f.remoteCommit[name]
	return nil
}

func (f *fakeSyncRepoCache) GetCommit(name string) (string, error) {
	return f.localCommit[name], nil
}

func (f *fakeSyncRepoCache) GetRemoteCommit(name, branch string) (string, error) {
	return f.remoteCommit[name], nil
}

func (f *fakeSyncRepoCache) GetIncomingLog(name, branch string) (string, error) {
	return f.incomingLog[name], nil
}

func (f *fakeSyncRepoCache) GetIncomingNameStatus(name, branch string) (string, error) {
	return f.nameStatus[name], nil
}

func TestSyncRepositoriesFetchesAndReportsRemoteChangesWithoutFix(t *testing.T) {
	cache := newFakeSyncRepoCache()
	cache.exists["grimoire"] = true
	cache.localCommit["grimoire"] = "111111111111"
	cache.remoteCommit["grimoire"] = "222222222222"

	var out bytes.Buffer
	syncRepositories(cache, map[string]config.RepoInfo{
		"grimoire": {URL: "https://example.test/grimoire.git", Branch: "main"},
	}, false, false, &out)

	if len(cache.fetches) != 1 || cache.fetches[0] != "grimoire" {
		t.Fatalf("fetches = %#v, want grimoire fetched once", cache.fetches)
	}
	if len(cache.pulls) != 0 {
		t.Fatalf("pulls = %#v, want no pull without repo fix", cache.pulls)
	}
	got := out.String()
	if !strings.Contains(got, "grimoire has remote changes: 1111111 -> 2222222") {
		t.Fatalf("output missing remote change report:\n%s", got)
	}
}

func TestSyncRepositoriesFetchesThenPullsWithFix(t *testing.T) {
	cache := newFakeSyncRepoCache()
	cache.exists["grimoire"] = true
	cache.localCommit["grimoire"] = "111111111111"
	cache.remoteCommit["grimoire"] = "222222222222"

	var out bytes.Buffer
	syncRepositories(cache, map[string]config.RepoInfo{
		"grimoire": {URL: "https://example.test/grimoire.git", Branch: "main"},
	}, true, false, &out)

	if len(cache.fetches) != 1 || cache.fetches[0] != "grimoire" {
		t.Fatalf("fetches = %#v, want grimoire fetched once", cache.fetches)
	}
	if len(cache.pulls) != 1 || cache.pulls[0] != "grimoire" {
		t.Fatalf("pulls = %#v, want grimoire pulled once", cache.pulls)
	}
	got := out.String()
	if !strings.Contains(got, "grimoire has remote changes: 1111111 -> 2222222") {
		t.Fatalf("output missing remote change report:\n%s", got)
	}
	if !strings.Contains(got, "Updated grimoire to 2222222") {
		t.Fatalf("output missing update report:\n%s", got)
	}
}

func TestSyncRepositoriesReportsUpToDateAfterFetch(t *testing.T) {
	cache := newFakeSyncRepoCache()
	cache.exists["grimoire"] = true
	cache.localCommit["grimoire"] = "111111111111"
	cache.remoteCommit["grimoire"] = "111111111111"

	var out bytes.Buffer
	syncRepositories(cache, map[string]config.RepoInfo{
		"grimoire": {URL: "https://example.test/grimoire.git", Branch: "main"},
	}, false, false, &out)

	if len(cache.fetches) != 1 || cache.fetches[0] != "grimoire" {
		t.Fatalf("fetches = %#v, want grimoire fetched once", cache.fetches)
	}
	if !strings.Contains(out.String(), "grimoire up to date") {
		t.Fatalf("output missing up-to-date report:\n%s", out.String())
	}
}

func TestSyncRepositoriesDiffShowsIncomingBreakdown(t *testing.T) {
	cache := newFakeSyncRepoCache()
	cache.exists["grimoire"] = true
	cache.localCommit["grimoire"] = "111111111111"
	cache.remoteCommit["grimoire"] = "222222222222"
	cache.incomingLog["grimoire"] = "2222222 add skill\n3333333 update docs"
	cache.nameStatus["grimoire"] = "A\tskills/new/SKILL.md\nM\tREADME.md"

	var out bytes.Buffer
	syncRepositories(cache, map[string]config.RepoInfo{
		"grimoire": {URL: "https://example.test/grimoire.git", Branch: "main"},
	}, false, true, &out)

	if len(cache.fetches) != 1 || cache.fetches[0] != "grimoire" {
		t.Fatalf("fetches = %#v, want grimoire fetched once", cache.fetches)
	}
	got := out.String()
	for _, want := range []string{
		"grimoire has remote changes: 1111111 -> 2222222",
		"Commits:",
		"2222222 add skill",
		"3333333 update docs",
		"Files:",
		"A\tskills/new/SKILL.md",
		"M\tREADME.md",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("output missing %q:\n%s", want, got)
		}
	}
}

// TestCollectSkillNames tests skill name collection from directories.
func TestCollectSkillNames(t *testing.T) {
	// Test with non-existent directory - should return nil
	skills := collectSkillNames("/nonexistent/path")
	if skills != nil {
		t.Errorf("expected nil for nonexistent path, got map with %d entries", len(skills))
	}
}

// TestFindMissingSkills tests the missing skills detection algorithm.
func TestFindMissingSkills(t *testing.T) {
	catalog := map[string]SkillInfo{
		"skill-a": {},
		"skill-b": {},
		"skill-c": {},
		"skill-d": {},
	}

	tests := []struct {
		name            string
		installedSkills map[string]map[string]bool
		wantMissing     map[string][]string
	}{
		{
			name: "all skills installed everywhere",
			installedSkills: map[string]map[string]bool{
				"pi/global":   {"skill-a": true, "skill-b": true, "skill-c": true, "skill-d": true},
				"codex/local": {"skill-a": true, "skill-b": true, "skill-c": true, "skill-d": true},
			},
			wantMissing: map[string][]string{
				"pi/global":   nil,
				"codex/local": nil,
			},
		},
		{
			name: "pi missing skills",
			installedSkills: map[string]map[string]bool{
				"pi/global":   {"skill-a": true},
				"codex/local": {"skill-a": true, "skill-b": true, "skill-c": true},
			},
			wantMissing: map[string][]string{
				"pi/global":   {"skill-b", "skill-c"}, // missing from pi
				"codex/local": nil,                    // has all
			},
		},
		{
			name: "multiple agents missing different skills",
			installedSkills: map[string]map[string]bool{
				"pi/global":     {"skill-a": true},
				"codex/local":   {"skill-b": true},
				"claude/global": {"skill-c": true},
			},
			wantMissing: map[string][]string{
				"pi/global":     {"skill-b", "skill-c"},
				"codex/local":   {"skill-a", "skill-c"},
				"claude/global": {"skill-a", "skill-b"},
			},
		},
		{
			name: "multiple named global directories sync independently",
			installedSkills: map[string]map[string]bool{
				"pi/global:default":    {"skill-a": true, "skill-b": true},
				"codex/global:default": {"skill-a": true},
				"codex/global:shared":  {"skill-b": true},
			},
			wantMissing: map[string][]string{
				"pi/global:default":    nil,
				"codex/global:default": {"skill-b"},
				"codex/global:shared":  {"skill-a"},
			},
		},
		{
			name: "skill not in catalog is not reported missing",
			installedSkills: map[string]map[string]bool{
				"pi/global": {"skill-a": true, "unknown-skill": true},
			},
			wantMissing: map[string][]string{
				"pi/global": nil,
			},
		},
		{
			name:            "no agents configured",
			installedSkills: map[string]map[string]bool{},
			wantMissing:     map[string][]string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := findMissingSkills(tt.installedSkills, catalog)

			// Check each expected target
			for target, expectedMissing := range tt.wantMissing {
				gotMissing, ok := got[target]
				if !ok {
					if len(expectedMissing) > 0 {
						t.Errorf("target %s not in result", target)
					}
					continue
				}

				if len(expectedMissing) == 0 && len(gotMissing) == 0 {
					continue
				}

				// Convert to set for comparison
				gotSet := make(map[string]bool)
				for _, s := range gotMissing {
					gotSet[s] = true
				}

				for _, s := range expectedMissing {
					if !gotSet[s] {
						t.Errorf("target %s: expected %s to be missing, but wasn't", target, s)
					}
					delete(gotSet, s)
				}

				// Check for unexpected missing skills
				if len(gotSet) > 0 {
					for s := range gotSet {
						// Check if it's in the catalog
						if _, inCatalog := catalog[s]; !inCatalog {
							continue // ok, skill not in catalog
						}
						t.Errorf("target %s: unexpected missing skill %s", target, s)
					}
				}
			}
		})
	}
}
