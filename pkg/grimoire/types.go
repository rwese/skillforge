package grimoire

import (
	"time"
)

// Grimoire represents the metadata for an installed skill.
type Grimoire struct {
	Version     int       `toml:"version"`
	Source      string    `toml:"source"`
	Commit      string    `toml:"commit"`
	InstalledAt time.Time `toml:"installed_at"`
}

// InstalledSkill represents a skill installed in a target.
type InstalledSkill struct {
	Name     string
	Path     string
	Grimoire Grimoire
	Target   string
}

// Skill represents a skill found in a repository.
type Skill struct {
	Name        string
	Description string
	Path        string
	Source      string
	Commit      string
}

// SkillOutput represents a skill for display output.
type SkillOutput struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Source      string `json:"source,omitempty"`
	Commit      string `json:"commit,omitempty"`
	Target      string `json:"target,omitempty"`
}
