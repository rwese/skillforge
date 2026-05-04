// Package config handles configuration loading, writing, and scope detection.
package config

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
)

// Config represents the main configuration structure.
type Config struct {
	Cache   CacheConfig          `toml:"cache"`
	Targets map[string]Target   `toml:"targets"`
	Repos   map[string]RepoInfo `toml:"repos"`
}

// CacheConfig defines the cache settings.
type CacheConfig struct {
	Path string `toml:"path"`
}

// Target defines an agent skill directory target.
type Target struct {
	Name    string
	Path    string `toml:"path"`
	Enabled bool   `toml:"enabled"`
}

// RepoInfo tracks a cached repository.
type RepoInfo struct {
	URL     string `toml:"url"`
	Branch  string `toml:"branch"`
	Updated string `toml:"updated"`
	Alias   string `toml:"alias"`
}

// Scope represents the config scope.
type Scope int

const (
	ScopeGlobal Scope = iota
	ScopeLocal
	ScopeAuto
)

// String returns the string representation of a Scope.
func (s Scope) String() string {
	switch s {
	case ScopeGlobal:
		return "global"
	case ScopeLocal:
		return "local"
	case ScopeAuto:
		return "auto"
	default:
		return "unknown"
	}
}

// Loader handles configuration loading with scope detection.
type Loader struct {
	scope      Scope
	globalPath string
}

// NewLoader creates a new config loader.
func NewLoader(scope Scope) *Loader {
	home, _ := os.UserHomeDir()
	globalPath := filepath.Join(home, ".config", "skillforge", "config.toml")

	return &Loader{
		scope:      scope,
		globalPath: globalPath,
	}
}

// GlobalPath returns the global config file path.
func (l *Loader) GlobalPath() string {
	return l.globalPath
}

// DetectLocalPath finds the local config path by searching from cwd to git root.
func DetectLocalPath() string {
	cwd, err := os.Getwd()
	if err != nil {
		return ""
	}

	// Find git root
	gitRoot := cwd
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	cmd.Dir = cwd
	if out, err := cmd.Output(); err == nil {
		gitRoot = strings.TrimSpace(string(out))
	}

	// Search from cwd to git root
	for {
		cfgPath := filepath.Join(cwd, ".skillforge", "config.toml")
		if _, err := os.Stat(cfgPath); err == nil {
			return cfgPath
		}

		if cwd == gitRoot || cwd == filepath.Dir(cwd) {
			break
		}
		cwd = filepath.Dir(cwd)
	}

	return ""
}

// Load loads the configuration based on scope.
func (l *Loader) Load() (*Config, error) {
	cfg := &Config{
		Targets: make(map[string]Target),
		Repos:   make(map[string]RepoInfo),
	}

	// Set defaults
	home, _ := os.UserHomeDir()
	cfg.Cache.Path = filepath.Join(home, ".cache", "skillforge", "repos")

	switch l.scope {
	case ScopeGlobal:
		if err := l.loadFile(l.globalPath, cfg); err != nil {
			return nil, err
		}
	case ScopeLocal:
		localPath := DetectLocalPath()
		if localPath == "" {
			return cfg, nil
		}
		if err := l.loadFile(localPath, cfg); err != nil {
			return nil, err
		}
	case ScopeAuto:
		// Load global first, then local overrides
		_ = l.loadFile(l.globalPath, cfg)

		localPath := DetectLocalPath()
		if localPath != "" {
			_ = l.loadFile(localPath, cfg)
		}
	}

	return cfg, nil
}

func (l *Loader) loadFile(path string, cfg *Config) error {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	// Parse TOML manually to merge
	var fileCfg Config
	if _, err := toml.Decode(string(data), &fileCfg); err != nil {
		return err
	}

	// Merge
	if fileCfg.Cache.Path != "" {
		cfg.Cache.Path = fileCfg.Cache.Path
	}
	for k, v := range fileCfg.Targets {
		cfg.Targets[k] = v
	}
	for k, v := range fileCfg.Repos {
		cfg.Repos[k] = v
	}

	return nil
}

// LoadAllRepos loads repos from both global and local configs.
// This is needed for skill discovery since repos are cached globally.
func (l *Loader) LoadAllRepos() (map[string]RepoInfo, CacheConfig, error) {
	repos := make(map[string]RepoInfo)
	cache := CacheConfig{
		Path: filepath.Join(os.Getenv("HOME"), ".cache", "skillforge", "repos"),
	}

	// Load global repos first
	if err := l.loadFile(l.globalPath, &Config{
		Repos:   repos,
		Targets: make(map[string]Target),
		Cache:   cache,
	}); err != nil {
		return nil, cache, err
	}

	// Override with local repos (local takes precedence for duplicates)
	localPath := DetectLocalPath()
	if localPath != "" {
		_ = l.loadFile(localPath, &Config{
			Repos:   repos,
			Targets: make(map[string]Target),
			Cache:   cache,
		})
	}

	return repos, cache, nil
}

// Save writes the configuration to a file.
func (l *Loader) Save(cfg *Config, local bool) error {
	var path string
	if local {
		path = DetectLocalPath()
		if path == "" {
			// Create local config in cwd
			cwd, _ := os.Getwd()
			localDir := filepath.Join(cwd, ".skillforge")
			if err := os.MkdirAll(localDir, 0755); err != nil {
				return err
			}
			path = filepath.Join(localDir, "config.toml")
		}
		// When saving locally, merge with existing local config
		// to preserve global-only data
		existing := &Config{
			Targets: make(map[string]Target),
			Repos:   make(map[string]RepoInfo),
		}
		_ = l.loadFile(path, existing)
		// Merge: new cfg takes precedence for its keys
		for k, v := range cfg.Targets {
			existing.Targets[k] = v
		}
		for k, v := range cfg.Repos {
			existing.Repos[k] = v
		}
		cfg = existing
	} else {
		path = l.globalPath
		// Ensure directory exists
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			return err
		}
	}

	data, err := toml.Marshal(cfg)
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
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
