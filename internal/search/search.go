package search

import (
	"strings"

	"github.com/rwese/skillforge/pkg/grimoire"
)

// SimpleKeywordSearch performs a simple keyword match search.
func SimpleKeywordSearch(skills []grimoire.Skill, query string) []grimoire.Skill {
	query = strings.ToLower(query)
	words := strings.Fields(query)

	var results []grimoire.Skill

	for _, skill := range skills {
		score := matchScore(skill, words)
		if score > 0 {
			results = append(results, skill)
		}
	}

	return results
}

// matchScore calculates a simple match score.
func matchScore(skill grimoire.Skill, words []string) int {
	score := 0
	name := strings.ToLower(skill.Name)
	desc := strings.ToLower(skill.Description)

	for _, word := range words {
		// Exact name match is highest priority
		if name == word {
			score += 10
		} else if strings.HasPrefix(name, word) {
			score += 5
		} else if strings.Contains(name, word) {
			score += 3
		}

		// Description matches
		if strings.Contains(desc, word) {
			score += 1
		}
	}

	return score
}

// SearchAll searches across all skill collections.
func SearchAll(skillSets map[string][]grimoire.Skill, query string) []grimoire.Skill {
	var allSkills []grimoire.Skill

	for _, skills := range skillSets {
		allSkills = append(allSkills, skills...)
	}

	return SimpleKeywordSearch(allSkills, query)
}
