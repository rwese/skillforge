package search

import (
	"testing"

	"github.com/rwese/skillforge/pkg/grimoire"
)

func TestMatchScore(t *testing.T) {
	skill := grimoire.Skill{
		Name:        "docker-build",
		Description: "Build Docker images efficiently",
	}

	tests := []struct {
		name   string
		words  []string
		minScore int
	}{
		{"exact name match", []string{"docker-build"}, 10},
		{"prefix match", []string{"docker"}, 5},
		{"partial name match", []string{"build"}, 3},
		{"description match", []string{"efficiently"}, 1},
		{"no match", []string{"kubernetes"}, 0},
		{"multiple words", []string{"docker", "build"}, 8},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := matchScore(skill, tt.words)
			if score < tt.minScore {
				t.Errorf("matchScore() = %d, want >= %d", score, tt.minScore)
			}
		})
	}
}

func TestSimpleKeywordSearch(t *testing.T) {
	skills := []grimoire.Skill{
		{Name: "docker-build", Description: "Build Docker images"},
		{Name: "docker-deploy", Description: "Deploy with Docker"},
		{Name: "kubernetes-setup", Description: "Setup Kubernetes cluster"},
		{Name: "git-commit", Description: "Conventional commits"},
	}

	tests := []struct {
		name    string
		query   string
		wantLen int
		wantFirst string
	}{
		{"docker query", "docker", 2, "docker-build"},
		{"git query", "git", 1, "git-commit"},
		{"kubernetes query", "kubernetes", 1, "kubernetes-setup"},
		{"no results", "python", 0, ""},
		{"empty query", "", 0, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results := SimpleKeywordSearch(skills, tt.query)
			if len(results) != tt.wantLen {
				t.Errorf("SimpleKeywordSearch() returned %d results, want %d", len(results), tt.wantLen)
			}
			if tt.wantLen > 0 && results[0].Name != tt.wantFirst {
				t.Errorf("First result = %q, want %q", results[0].Name, tt.wantFirst)
			}
		})
	}
}

func TestSearchAll(t *testing.T) {
	skillSets := map[string][]grimoire.Skill{
		"repo1": {
			{Name: "docker-build", Description: "Build images"},
			{Name: "git-commit", Description: "Commit changes"},
		},
		"repo2": {
			{Name: "docker-deploy", Description: "Deploy containers"},
			{Name: "kubernetes-setup", Description: "Setup cluster"},
		},
	}

	results := SearchAll(skillSets, "docker")
	if len(results) != 2 {
		t.Errorf("SearchAll() returned %d results, want 2", len(results))
	}

	// Test that results come from multiple repos
	found := make(map[string]bool)
	for _, s := range results {
		found[s.Name] = true
	}
	if !found["docker-build"] || !found["docker-deploy"] {
		t.Errorf("SearchAll() missing expected results")
	}
}

func TestSearchAllEmpty(t *testing.T) {
	skillSets := map[string][]grimoire.Skill{}

	results := SearchAll(skillSets, "docker")
	if len(results) != 0 {
		t.Errorf("SearchAll() with empty map returned %d results, want 0", len(results))
	}
}

func TestSearchCaseInsensitive(t *testing.T) {
	skills := []grimoire.Skill{
		{Name: "Docker-Build", Description: "BUILD docker images"},
	}

	results := SimpleKeywordSearch(skills, "docker")
	if len(results) != 1 {
		t.Errorf("SimpleKeywordSearch() case insensitive returned %d results, want 1", len(results))
	}

	results = SimpleKeywordSearch(skills, "DOCKER")
	if len(results) != 1 {
		t.Errorf("SimpleKeywordSearch() uppercase query returned %d results, want 1", len(results))
	}
}

func TestMatchScoreExactName(t *testing.T) {
	skill := grimoire.Skill{Name: "test", Description: ""}
	score := matchScore(skill, []string{"test"})
	if score != 10 {
		t.Errorf("matchScore() exact name = %d, want 10", score)
	}
}

func TestMatchScorePrefix(t *testing.T) {
	skill := grimoire.Skill{Name: "testing", Description: ""}
	score := matchScore(skill, []string{"test"})
	if score != 5 {
		t.Errorf("matchScore() prefix = %d, want 5", score)
	}
}

func TestMatchScoreContains(t *testing.T) {
	skill := grimoire.Skill{Name: "my-testing", Description: ""}
	score := matchScore(skill, []string{"test"})
	if score != 3 {
		t.Errorf("matchScore() contains = %d, want 3", score)
	}
}

func TestMatchScoreDescription(t *testing.T) {
	skill := grimoire.Skill{Name: "other", Description: "testing keyword"}
	score := matchScore(skill, []string{"testing"})
	if score != 1 {
		t.Errorf("matchScore() description = %d, want 1", score)
	}
}

func TestMatchScoreMultiple(t *testing.T) {
	skill := grimoire.Skill{Name: "docker-build", Description: "build docker"}
	score := matchScore(skill, []string{"docker", "build"})
	// docker: prefix match (5) + build: prefix match (5) + description (1+1) = 12
	if score < 10 {
		t.Errorf("matchScore() multiple = %d, want >= 10", score)
	}
}

func TestSimpleKeywordSearchEmptySkills(t *testing.T) {
	results := SimpleKeywordSearch([]grimoire.Skill{}, "test")
	if len(results) != 0 {
		t.Errorf("SimpleKeywordSearch() with empty skills returned %d results", len(results))
	}
}
