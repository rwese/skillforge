package grimoire

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
