// Package grimoire defines the skill registry metadata format.
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

// Skill represents a skill found in a repository.
type Skill struct {
	Name        string
	Description string
	Path        string
	Source      string
	Commit      string
}

// InstalledSkill represents a skill installed in a target.
type InstalledSkill struct {
	Name      string
	Path      string
	Grimoire  Grimoire
	Target    string
}
