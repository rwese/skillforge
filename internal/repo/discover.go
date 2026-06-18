package repo

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/rwese/skillforge/pkg/grimoire"
)

// Discover finds all skills in a cached repository.
//
// Skill directories are located by walking <cachePath>/skills and
// <cachePath>/.agents/skills recursively. Any directory containing a
// SKILL.md (and no .grimoire marker) is a skill. The Skill.Name is
// the path of that directory relative to the skills root, using
// forward slashes, so a nested skill at
//
//	<cachePath>/skills/architecture/event-sourced-commands
//
// is named "architecture/event-sourced-commands", and a flat skill at
//
//	<cachePath>/skills/foo
//
// is named "foo". The walk stops descending into a skill once it is
// found (skills never contain nested skills).
//
// For backwards compatibility, when <cachePath>/skills does not exist
// the function also accepts skills placed directly under cachePath
// (one level deep, no skills/ wrapper) and uses their bare directory
// name as Skill.Name. The "skills" and ".agents" entries are skipped
// in that fallback to avoid double-counting with the recursive walk.
func DiscoverSkills(cachePath, source string) ([]grimoire.Skill, error) {
	var skills []grimoire.Skill

	roots := []string{
		filepath.Join(cachePath, "skills"),
		filepath.Join(cachePath, ".agents", "skills"),
	}
	for _, root := range roots {
		if err := walkSkillsRecursive(root, source, &skills); err != nil {
			return nil, err
		}
	}

	// Legacy fallback: skills placed directly under cachePath, used
	// by repos that pre-date the skills/ wrapper convention. Only one
	// level deep; deeper nesting requires the skills/ wrapper.
	entries, err := os.ReadDir(cachePath)
	if err != nil {
		if os.IsNotExist(err) {
			return skills, nil
		}
		return nil, fmt.Errorf("reading cache directory %s: %w", cachePath, err)
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		// Skip the recursive roots so the legacy fallback does not
		// double-count skills already discovered by walkSkillsRecursive.
		if name == "skills" || name == ".agents" {
			continue
		}
		skillPath := filepath.Join(cachePath, name)
		if isSkillDir(skillPath) {
			skills = append(skills, grimoire.Skill{
				Name:        name,
				Description: readSkillDescription(skillPath),
				Path:        skillPath,
				Source:      source,
			})
		}
	}

	return skills, nil
}

// walkSkillsRecursive walks root looking for skill directories. A
// directory that contains SKILL.md (and no .grimoire marker) is a
// skill; the walker records it with Skill.Name equal to its path
// relative to root (forward-slash separated) and does NOT recurse
// into it. A directory that is NOT a skill is recursed into one
// level deeper. This naturally bounds the search at the depth at
// which SKILL.md first appears, which is the skill boundary.
//
// A missing root is treated as "no skills here" (not an error), so
// callers can pass either or both of the conventional roots without
// first checking for existence.
func walkSkillsRecursive(root, source string, skills *[]grimoire.Skill) error {
	return walkSkillsRecursiveFrom(root, root, source, skills)
}

// walkSkillsRecursiveFrom is the worker behind walkSkillsRecursive.
// baseRoot is the top-level walk root; relPath for a found skill is
// always computed relative to baseRoot so nested-skill names carry
// their full category path (e.g. "architecture/event-sourced-commands").
func walkSkillsRecursiveFrom(baseRoot, dir, source string, skills *[]grimoire.Skill) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("reading skills root %s: %w", dir, err)
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		skillPath := filepath.Join(dir, entry.Name())
		if isSkillDir(skillPath) {
			relPath, err := filepath.Rel(baseRoot, skillPath)
			if err != nil {
				continue
			}
			*skills = append(*skills, grimoire.Skill{
				Name:        filepath.ToSlash(relPath),
				Description: readSkillDescription(skillPath),
				Path:        skillPath,
				Source:      source,
			})
			continue
		}
		// Not a skill; descend one level. We do not use filepath.Walk
		// here because we want to skip *into* a skill directory once it
		// is found (skills never contain skills), and explicit
		// recursion makes that boundary obvious.
		if err := walkSkillsRecursiveFrom(baseRoot, skillPath, source, skills); err != nil {
			return err
		}
	}
	return nil
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
