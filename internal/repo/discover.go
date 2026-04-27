package repo

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/rwese/skillforge-ng/pkg/grimoire"
)

// Discover finds all skills in a cached repository.
func DiscoverSkills(cachePath, source string) ([]grimoire.Skill, error) {
	var skills []grimoire.Skill

	entries, err := os.ReadDir(cachePath)
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		skillPath := filepath.Join(cachePath, entry.Name())
		if isSkillDir(skillPath) {
			desc := readSkillDescription(skillPath)
			skills = append(skills, grimoire.Skill{
				Name:        entry.Name(),
				Description: desc,
				Path:        skillPath,
				Source:      source,
			})
		}
	}

	return skills, nil
}

// isSkillDir determines if a directory is a valid skill.
func isSkillDir(path string) bool {
	// A skill must have at least a SKILL.md file
	skillFile := filepath.Join(path, "SKILL.md")
	if _, err := os.Stat(skillFile); err != nil {
		return false
	}

	// Must not contain a .grimoire file (it's a repo, not a skill)
	grimoireFile := filepath.Join(path, ".grimoire")
	if _, err := os.Stat(grimoireFile); err == nil {
		return false
	}

	return true
}

// readSkillDescription reads the first line from SKILL.md as description.
func readSkillDescription(path string) string {
	skillFile := filepath.Join(path, "SKILL.md")
	data, err := os.ReadFile(skillFile)
	if err != nil {
		return ""
	}

	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" && !strings.HasPrefix(line, "#") {
			// Truncate to reasonable length
			if len(line) > 100 {
				line = line[:97] + "..."
			}
			return line
		}
	}
	return ""
}

// DiscoverInCache discovers all skills across cached repos.
func DiscoverInCache(cache *Cache, repos map[string]string) (map[string][]grimoire.Skill, error) {
	result := make(map[string][]grimoire.Skill)

	for name := range repos {
		if !cache.Exists(name) {
			continue
		}

		path := cache.PathFor(name)
		skills, err := DiscoverSkills(path, repos[name])
		if err != nil {
			return nil, fmt.Errorf("discovering skills in %s: %w", name, err)
		}
		result[name] = skills
	}

	return result, nil
}
