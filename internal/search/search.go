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
//
// To support nested-skill names ("architecture/event-sourced-commands")
// each path segment of skill.Name is scored independently. A flat
// skill ("docker-build") has a single segment and scores identically
// to the legacy behaviour. A nested skill can be matched by either
// its category or its leaf name, so a query of "architecture" finds
// the architecture-class skills and "event-sourced" finds
// "architecture/event-sourced-commands". Description matches add
// 1 per word regardless.
func matchScore(skill grimoire.Skill, words []string) int {
	score := 0
	desc := strings.ToLower(skill.Description)

	segments := strings.Split(skill.Name, "/")
	for _, word := range words {
		for _, seg := range segments {
			segLower := strings.ToLower(seg)
			if segLower == word {
				score += 10
			} else if strings.HasPrefix(segLower, word) {
				score += 5
			} else if strings.Contains(segLower, word) {
				score += 3
			}
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
