package cmd

import (
	"testing"

	"github.com/rwese/skillforge-ng/internal/agents"
)

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

// TestParseTarget tests target string parsing.
func TestParseTarget(t *testing.T) {
	tests := []struct {
		target    string
		wantAgent string
		wantScope string
	}{
		{"pi/global", "pi", "global"},
		{"codex/local", "codex", "local"},
		{"claude/global", "claude", "global"},
		{"single", "single", ""},
	}

	for _, tt := range tests {
		t.Run(tt.target, func(t *testing.T) {
			agent, scope := parseTarget(tt.target)
			if agent != tt.wantAgent {
				t.Errorf("parseTarget(%q) agent = %q, want %q", tt.target, agent, tt.wantAgent)
			}
			if scope != tt.wantScope {
				t.Errorf("parseTarget(%q) scope = %q, want %q", tt.target, scope, tt.wantScope)
			}
		})
	}
}

// TestShouldSyncScope tests scope filtering.
func TestShouldSyncScope(t *testing.T) {
	path := &agents.Path{Value: "/some/path"}

	tests := []struct {
		name       string
		path       *agents.Path
		scopeFlag  string
		scopeValue string
		want       bool
	}{
		{"nil path returns false", nil, "auto", "global", false},
		{"global scope matches", path, "global", "global", true},
		{"global scope doesn't match local", path, "global", "local", false},
		{"local scope matches", path, "local", "local", true},
		{"local scope doesn't match global", path, "local", "global", false},
		{"auto scope matches global", path, "auto", "global", true},
		{"auto scope matches local", path, "auto", "local", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := shouldSyncScope(tt.path, tt.scopeFlag, tt.scopeValue)
			if got != tt.want {
				t.Errorf("shouldSyncScope() = %v, want %v", got, tt.want)
			}
		})
	}
}
