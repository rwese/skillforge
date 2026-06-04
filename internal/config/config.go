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
	Cache   CacheConfig         `toml:"cache"`
	Targets map[string]Target   `toml:"targets"`
	Repos   map[string]RepoInfo `toml:"repos"`
}

// CacheConfig defines the cache settings.
type CacheConfig struct {
	Path string `toml:"path"`
}

// Target defines an agent skill directory target.
// GlobalPath is used when installing with -s global (global agent).
// LocalPath is used when installing with -s local (local/project agent).
type Target struct {
	Name        string            `toml:"name"`
	GlobalPath  string            `toml:"globalPath"`
	GlobalPaths map[string]string `toml:"globalPaths"`
	LocalPath   string            `toml:"localPath"`
	Enabled     bool              `toml:"enabled"`
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
	ScopeLocal Scope = iota
	ScopeGlobal
)

// String returns the string representation of a Scope.
func (s Scope) String() string {
	if s == ScopeGlobal {
		return "global"
	}
	return "local"
}

// IsLocal returns true if this is a local scope.
func (s Scope) IsLocal() bool {
	return s == ScopeLocal
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

// DetectGitRoot returns the git root for cwd, or empty when cwd is not in a git repo.
func DetectGitRoot() string {
	cwd, err := os.Getwd()
	if err != nil {
		return ""
	}

	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	cmd.Dir = cwd
	if out, err := cmd.Output(); err == nil {
		return strings.TrimSpace(string(out))
	}
	return ""
}

// DetectLocalPath finds the local config path by searching from cwd to git root.
func DetectLocalPath() string {
	cwd, err := os.Getwd()
	if err != nil {
		return ""
	}

	// Find git root
	gitRoot := DetectGitRoot()
	if gitRoot == "" {
		gitRoot = cwd
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
	default: // ScopeLocal
		localPath := DetectLocalPath()
		if localPath == "" {
			return cfg, nil
		}
		if err := l.loadFile(localPath, cfg); err != nil {
			return nil, err
		}
	}

	return cfg, nil
}

// DefaultCachePath returns the default cache directory path
// (under the user's home directory). It is exported so callers
// building an effective cache path can recognize the default.
func DefaultCachePath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".cache", "skillforge", "repos")
}

// EffectiveCachePathFromConfigs returns the effective cache directory path
// derived from two already-loaded *Config values: local override > global
// override > default.
//
// Load() always populates cfg.Cache.Path with the default, so a non-empty
// value does not by itself indicate an explicit override. We treat a value
// as an override only when it differs from the default. A user who writes
// the default into their config explicitly gets the default back, which is
// the same outcome as not setting it at all.
func EffectiveCachePathFromConfigs(globalCfg, localCfg *Config) string {
	defaultPath := DefaultCachePath()
	effective := defaultPath
	if globalCfg != nil && globalCfg.Cache.Path != "" && globalCfg.Cache.Path != defaultPath {
		effective = globalCfg.Cache.Path
	}
	if localCfg != nil && localCfg.Cache.Path != "" && localCfg.Cache.Path != defaultPath {
		effective = localCfg.Cache.Path
	}
	return effective
}

// EffectiveCachePath returns the cache directory path that should be used for
// the on-disk git cache, considering both the global and local configs.
//
// The cache is a single shared directory (a git checkout location), so the
// effective path is resolved as: local override > global override > default.
//
// Reading both files directly (rather than inspecting values already merged
// into a *Config) is required: Load() always populates cfg.Cache.Path with
// the default, which would otherwise hide whether a scope actually set the
// field. Only an explicit value in a config file is treated as an override.
func (l *Loader) EffectiveCachePath() (string, error) {
	effective := DefaultCachePath()

	if err := l.readCachePathField(l.globalPath, &effective); err != nil {
		return "", err
	}

	// Local config overrides the global one when it explicitly sets cache.path.
	if localPath := DetectLocalPath(); localPath != "" {
		if err := l.readCachePathField(localPath, &effective); err != nil {
			return "", err
		}
	}

	return effective, nil
}

// readCachePathField reads only the cache.path field from a config file and
// updates *effective if the file sets a non-empty value.
func (l *Loader) readCachePathField(path string, effective *string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	var fileCfg struct {
		Cache CacheConfig `toml:"cache"`
	}
	if _, err := toml.Decode(string(data), &fileCfg); err != nil {
		return err
	}

	if fileCfg.Cache.Path != "" {
		*effective = fileCfg.Cache.Path
	}
	return nil
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

		// When saving locally, preserve cache settings from existing config
		// but replace targets and repos completely with the new cfg
		existing := &Config{
			Targets: make(map[string]Target),
			Repos:   make(map[string]RepoInfo),
		}
		_ = l.loadFile(path, existing)

		// Preserve cache path from existing config, use new if set
		if existing.Cache.Path != "" && cfg.Cache.Path == "" {
			cfg.Cache.Path = existing.Cache.Path
		}

		// Completely replace targets and repos with new cfg
		// (new cfg represents the complete local state)
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
