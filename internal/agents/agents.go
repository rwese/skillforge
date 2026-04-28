// Package agents handles agent configuration loading and merging.
package agents

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
)

// Agent represents a single agent's skill paths.
type Agent struct {
	Name   string `toml:"name"`
	Global *Path  `toml:"global"`
	Local  *Path  `toml:"local"`
}

// Path represents a skill directory path (absolute or relative).
type Path struct {
	Value string `toml:"value"`
}

// AgentsConfig represents the agents.toml configuration.
type AgentsConfig struct {
	Agents map[string]Agent `toml:"agents"`
}

// KnownAgents defines default paths for common agents.
var KnownAgents = map[string]AgentDefinition{
	"pi": {
		Name:            "pi",
		DefaultGlobal:   "~/.pi/agent/skills",
		DefaultLocal:    ".pi/skills",
		DetectionPaths:  []string{"~/.pi/agent/skills"},
		DetectionMarker: ".pi",
	},
	"codex": {
		Name:            "codex",
		DefaultGlobal:   "~/.codex/skills",
		DefaultLocal:    ".codex/skills",
		DetectionPaths:  []string{"~/.codex/skills"},
		DetectionMarker: ".codex",
	},
	"claude": {
		Name:            "claude",
		DefaultGlobal:   "~/.claude/skills",
		DefaultLocal:    ".claude/skills",
		DetectionPaths:  []string{"~/.claude/skills"},
		DetectionMarker: ".claude",
	},
}

// AgentDefinition defines how to detect and configure an agent.
type AgentDefinition struct {
	Name            string
	DefaultGlobal   string
	DefaultLocal    string
	DetectionPaths  []string
	DetectionMarker string
}

// Scope represents config scope.
type Scope int

const (
	ScopeGlobal Scope = iota
	ScopeLocal
)

// agentsPaths returns the paths to global and local agents.toml files.
func agentsPaths() (globalPath string, localPath string) {
	home, _ := os.UserHomeDir()
	globalPath = filepath.Join(home, ".config", "skillforge", "agents.toml")

	// Find local path by searching from cwd to git root
	cwd, _ := os.Getwd()
	if cwd != "" {
		gitRoot := cwd
		cmd := exec.Command("git", "rev-parse", "--show-toplevel")
		cmd.Dir = cwd
		if out, err := cmd.Output(); err == nil {
			gitRoot = strings.TrimSpace(string(out))
		}

		for {
			localCfg := filepath.Join(cwd, ".skillforge", "agents.toml")
			if _, err := os.Stat(localCfg); err == nil {
				localPath = localCfg
				break
			}

			if cwd == gitRoot || cwd == filepath.Dir(cwd) {
				break
			}
			cwd = filepath.Dir(cwd)
		}
	}

	return globalPath, localPath
}

// LoadAgents loads agent configuration, merging global and local.
func LoadAgents() (*AgentsConfig, error) {
	globalPath, localPath := agentsPaths()

	cfg := &AgentsConfig{
		Agents: make(map[string]Agent),
	}

	// Load global first
	if err := loadFile(globalPath, cfg); err != nil {
		return nil, err
	}

	// Merge local overrides
	if localPath != "" {
		localCfg := &AgentsConfig{Agents: make(map[string]Agent)}
		if err := loadFile(localPath, localCfg); err != nil {
			return nil, err
		}
		mergeAgents(cfg, localCfg)
	}

	return cfg, nil
}

// loadFile loads a single agents.toml file.
func loadFile(path string, cfg *AgentsConfig) error {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	var fileCfg AgentsConfig
	if _, err := toml.Decode(string(data), &fileCfg); err != nil {
		return err
	}

	for k, v := range fileCfg.Agents {
		cfg.Agents[k] = v
	}

	return nil
}

// mergeAgents merges localCfg into cfg, with local taking precedence.
// Per-key merge: nil in local means skip, non-nil overrides.
func mergeAgents(cfg *AgentsConfig, localCfg *AgentsConfig) {
	for name, localAgent := range localCfg.Agents {
		existing, exists := cfg.Agents[name]

		if !exists {
			// New agent from local
			cfg.Agents[name] = localAgent
			continue
		}

		// Merge per-key: local overrides, nil means keep existing
		if localAgent.Global != nil {
			existing.Global = localAgent.Global
		}
		if localAgent.Local != nil {
			existing.Local = localAgent.Local
		}

		cfg.Agents[name] = existing
	}
}

// SaveAgents saves agent configuration to the appropriate scope.
func SaveAgents(cfg *AgentsConfig, scope Scope) error {
	home, _ := os.UserHomeDir()

	var path string
	switch scope {
	case ScopeGlobal:
		path = filepath.Join(home, ".config", "skillforge", "agents.toml")
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			return err
		}
	case ScopeLocal:
		cwd, _ := os.Getwd()
		if cwd == "" {
			return &ErrNotInProject{}
		}

		gitRoot := cwd
		cmd := exec.Command("git", "rev-parse", "--show-toplevel")
		cmd.Dir = cwd
		if out, err := cmd.Output(); err == nil {
			gitRoot = strings.TrimSpace(string(out))
		}

		localDir := filepath.Join(gitRoot, ".skillforge")
		if err := os.MkdirAll(localDir, 0755); err != nil {
			return err
		}
		path = filepath.Join(localDir, "agents.toml")

		// Merge with existing global config
		globalCfg := &AgentsConfig{Agents: make(map[string]Agent)}
		if err := loadFile(filepath.Join(home, ".config", "skillforge", "agents.toml"), globalCfg); err != nil {
			return err
		}
		for k, v := range cfg.Agents {
			globalCfg.Agents[k] = v
		}
		cfg = globalCfg
	}

	data, err := toml.Marshal(cfg)
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}

// DetectAgents scans the filesystem for known agents.
func DetectAgents() map[string]bool {
	detected := make(map[string]bool)

	for name, def := range KnownAgents {
		for _, path := range def.DetectionPaths {
			expanded := ExpandPath(path)
			if _, err := os.Stat(expanded); err == nil {
				detected[name] = true
				break
			}
		}
	}

	return detected
}

// ExpandPath expands ~ to home directory.
func ExpandPath(path string) string {
	if strings.HasPrefix(path, "~/") {
		home, _ := os.UserHomeDir()
		return filepath.Join(home, path[2:])
	}
	return path
}

// ContractPath replaces home directory with ~.
func ContractPath(path string) string {
	home, _ := os.UserHomeDir()
	if home != "" && strings.HasPrefix(path, home) {
		return "~" + path[len(home):]
	}
	return path
}

// ErrNotInProject indicates the user is not in a git project.
type ErrNotInProject struct{}

func (e *ErrNotInProject) Error() string {
	return "not in a git project"
}
