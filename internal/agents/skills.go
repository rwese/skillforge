package agents

import (
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
	"github.com/rwese/skillforge-ng/pkg/grimoire"
)

// ResolvedSkill represents a skill with its resolved paths.
type ResolvedSkill struct {
	Name     string
	Agent    string
	Scope    string
	Path     string
	Grimoire grimoire.Grimoire
}

// ResolveSkills returns all skills from all configured agents.
func ResolveSkills() ([]ResolvedSkill, error) {
	cfg, err := LoadAgents()
	if err != nil {
		return nil, err
	}

	var skills []ResolvedSkill

	for name, agent := range cfg.Agents {
		// Check global
		if agent.Global != nil {
			path := ExpandPath(agent.Global.Value)
			if grimoire, err := loadGrimoire(path); err == nil {
				skills = append(skills, ResolvedSkill{
					Name:     grimoireFileToSkillName(path),
					Agent:    name,
					Scope:    "global",
					Path:     path,
					Grimoire: grimoire,
				})
			}
		}

		// Check local
		if agent.Local != nil {
			path := ExpandPath(agent.Local.Value)
			if grimoire, err := loadGrimoire(path); err == nil {
				skills = append(skills, ResolvedSkill{
					Name:     grimoireFileToSkillName(path),
					Agent:    name,
					Scope:    "local",
					Path:     path,
					Grimoire: grimoire,
				})
			}
		}
	}

	return skills, nil
}

// loadGrimoire loads .grimoire from a skill directory.
func loadGrimoire(skillPath string) (grimoire.Grimoire, error) {
	var g grimoire.Grimoire
	grimoirePath := filepath.Join(skillPath, ".grimoire")
	data, err := os.ReadFile(grimoirePath)
	if err != nil {
		return g, err
	}

	_, err = toml.Decode(string(data), &g)
	return g, err
}

// grimoireFileToSkillName extracts skill name from its path.
func grimoireFileToSkillName(skillPath string) string {
	return filepath.Base(skillPath)
}

// GetSkillPaths returns the global and local paths for an agent.
func GetSkillPaths(agentName string) (globalPath, localPath string, err error) {
	cfg, err := LoadAgents()
	if err != nil {
		return "", "", err
	}

	agent, exists := cfg.Agents[agentName]
	if !exists {
		return "", "", &ErrAgentNotFound{Name: agentName}
	}

	if agent.Global != nil {
		globalPath = ExpandPath(agent.Global.Value)
	}
	if agent.Local != nil {
		localPath = ExpandPath(agent.Local.Value)
	}

	return globalPath, localPath, nil
}

// ErrAgentNotFound indicates the requested agent is not configured.
type ErrAgentNotFound struct {
	Name string
}

func (e *ErrAgentNotFound) Error() string {
	return "agent not found: " + e.Name
}
