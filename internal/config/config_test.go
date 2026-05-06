package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestExpandPath(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantHome bool
	}{
		{
			name:     "tilde path",
			input:    "~/test",
			wantHome: true,
		},
		{
			name:     "absolute path",
			input:    "/absolute/path",
			wantHome: false,
		},
		{
			name:     "relative path",
			input:    "relative/path",
			wantHome: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExpandPath(tt.input)
			home, _ := os.UserHomeDir()

			if tt.wantHome {
				if home == "" {
					t.Skip("no home directory")
				}
				if result == tt.input {
					t.Errorf("ExpandPath(%q) = %q, want path with home directory", tt.input, result)
				}
			} else {
				if result != tt.input {
					t.Errorf("ExpandPath(%q) = %q, want %q", tt.input, result, tt.input)
				}
			}
		})
	}
}

func TestContractPath(t *testing.T) {
	home, _ := os.UserHomeDir()
	if home == "" {
		t.Skip("no home directory")
	}

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "path with home",
			input: filepath.Join(home, "test"),
			want:  "~/test",
		},
		{
			name:  "absolute path",
			input: "/absolute/path",
			want:  "/absolute/path",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ContractPath(tt.input)
			if result != tt.want {
				t.Errorf("ContractPath(%q) = %q, want %q", tt.input, result, tt.want)
			}
		})
	}
}

func TestNewLoader(t *testing.T) {
	tests := []struct {
		name  string
		scope Scope
	}{
		{"global scope", ScopeGlobal},
		{"local scope", ScopeLocal},
		{"auto scope", ScopeLocal},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			loader := NewLoader(tt.scope)
			if loader == nil {
				t.Errorf("NewLoader() returned nil")
			}
			if loader.scope != tt.scope {
				t.Errorf("NewLoader().scope = %v, want %v", loader.scope, tt.scope)
			}
		})
	}
}

func TestLoaderLoad(t *testing.T) {
	loader := NewLoader(ScopeGlobal)
	cfg, err := loader.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	// Check defaults are set
	if cfg.Cache.Path == "" {
		t.Error("Load() returned empty Cache.Path, expected default")
	}
	if cfg.Targets == nil {
		t.Error("Load() returned nil Targets")
	}
	if cfg.Repos == nil {
		t.Error("Load() returned nil Repos")
	}
}

func TestLoaderLoadAuto(t *testing.T) {
	loader := NewLoader(ScopeLocal)
	cfg, err := loader.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	// Should load global config (with defaults if no file)
	if cfg.Cache.Path == "" {
		t.Error("Load() returned empty Cache.Path for ScopeLocal")
	}
}

func TestLoaderSaveAndLoad(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.toml")

	// Create a loader with custom global path
	loader := &Loader{
		scope:      ScopeGlobal,
		globalPath: configPath,
	}

	cfg := &Config{
		Cache: CacheConfig{
			Path: "/test/cache",
		},
		Targets: map[string]Target{
			"test-target": {
				Name:       "test-target",
				GlobalPath: "/test/path",
				LocalPath:  "/test/local",
				Enabled:    true,
			},
		},
		Repos: map[string]RepoInfo{
			"test-repo": {
				URL:    "https://github.com/test/repo",
				Branch: "main",
			},
		},
	}

	// Save
	if err := loader.Save(cfg, false); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	// Load in new loader
	loader2 := &Loader{
		scope:      ScopeGlobal,
		globalPath: configPath,
	}
	cfg2, err := loader2.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg2.Cache.Path != cfg.Cache.Path {
		t.Errorf("Cache.Path = %q, want %q", cfg2.Cache.Path, cfg.Cache.Path)
	}
	if len(cfg2.Targets) != len(cfg.Targets) {
		t.Errorf("Targets count = %d, want %d", len(cfg2.Targets), len(cfg.Targets))
	}
	if len(cfg2.Repos) != len(cfg.Repos) {
		t.Errorf("Repos count = %d, want %d", len(cfg2.Repos), len(cfg.Repos))
	}
}

func TestLoaderSaveLocalCreatesDir(t *testing.T) {
	tmpDir := t.TempDir()
	localDir := filepath.Join(tmpDir, ".skillforge")
	configPath := filepath.Join(localDir, "config.toml")

	// Create loader that will detect this as local
	loader := &Loader{
		scope: ScopeLocal,
	}

	cfg := &Config{
		Cache: CacheConfig{
			Path: "/cache",
		},
		Targets: map[string]Target{
			"local-target": {
				Name:       "local-target",
				GlobalPath: "/local/global",
				LocalPath:  "/local/path",
				Enabled:    true,
			},
		},
	}

	// Change to temp dir so DetectLocalPath finds it
	oldCwd, _ := os.Getwd()
	defer os.Chdir(oldCwd)
	os.Chdir(tmpDir)

	// Create .skillforge dir
	if err := os.MkdirAll(localDir, 0755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}

	// Save with local=true
	if err := loader.Save(cfg, true); err != nil {
		t.Fatalf("Save(local=true) error = %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Errorf("Config file not created at %s", configPath)
	}
}

func TestDetectLocalPath(t *testing.T) {
	// Test that it returns empty when no local config exists
	result := DetectLocalPath()
	// Should return empty or a valid path, not error
	_ = result
}

func TestDetectLocalPathWithConfig(t *testing.T) {
	tmpDir := t.TempDir()
	localDir := filepath.Join(tmpDir, ".skillforge")
	configPath := filepath.Join(localDir, "config.toml")

	// Create the config file
	if err := os.MkdirAll(localDir, 0755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(configPath, []byte(""), 0644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	// Change to temp dir
	oldCwd, _ := os.Getwd()
	defer os.Chdir(oldCwd)
	os.Chdir(tmpDir)

	result := DetectLocalPath()
	// On macOS, /var is a symlink to /private/var, so compare resolved paths
	resultResolved, _ := filepath.EvalSymlinks(result)
	configPathResolved, _ := filepath.EvalSymlinks(configPath)
	if resultResolved != configPathResolved {
		t.Errorf("DetectLocalPath() = %q, want %q (resolved: %q)", result, configPath, resultResolved)
	}
}

func TestScopeConstants(t *testing.T) {
	tests := []struct {
		name  string
		scope Scope
	}{
		{"ScopeGlobal", ScopeGlobal},
		{"ScopeLocal", ScopeLocal},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Just verify the scope constants exist and have valid string representations
			if tt.scope.String() == "" {
				t.Errorf("Scope %s has empty String()", tt.name)
			}
		})
	}
}

// --- Config Merge Tests ---

func TestLoad_GlobalOnly(t *testing.T) {
	tmpDir := t.TempDir()
	globalPath := filepath.Join(tmpDir, "global.toml")

	// Write global config
	globalCfg := `cache.path = "/global/cache"

[targets.pi]
globalPath = "/global/pi"
localPath = "/local/pi"
enabled = true
`
	if err := os.WriteFile(globalPath, []byte(globalCfg), 0644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	loader := &Loader{
		scope:      ScopeGlobal,
		globalPath: globalPath,
	}

	cfg, err := loader.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.Cache.Path != "/global/cache" {
		t.Errorf("Cache.Path = %q, want %q", cfg.Cache.Path, "/global/cache")
	}

	pi, ok := cfg.Targets["pi"]
	if !ok {
		t.Fatal("Expected target 'pi' not found")
	}
	if pi.GlobalPath != "/global/pi" {
		t.Errorf("pi.GlobalPath = %q, want %q", pi.GlobalPath, "/global/pi")
	}
	if !pi.Enabled {
		t.Error("pi.Enabled = false, want true")
	}
}

func TestLoad_LocalOnly(t *testing.T) {
	tmpDir := t.TempDir()
	localDir := filepath.Join(tmpDir, ".skillforge")
	if err := os.MkdirAll(localDir, 0755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	localPath := filepath.Join(localDir, "config.toml")

	// Write local config
	localCfg := `cache.path = "/local/cache"

[targets.local]
globalPath = "/global/local"
localPath = "/local/skills"
enabled = true
`
	if err := os.WriteFile(localPath, []byte(localCfg), 0644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	// Change to tmpDir so DetectLocalPath finds it
	oldCwd, _ := os.Getwd()
	defer os.Chdir(oldCwd)
	os.Chdir(tmpDir)

	loader := &Loader{
		scope:      ScopeLocal,
		globalPath: filepath.Join(tmpDir, "global.toml"), // doesn't exist
	}

	cfg, err := loader.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.Cache.Path != "/local/cache" {
		t.Errorf("Cache.Path = %q, want %q", cfg.Cache.Path, "/local/cache")
	}

	local, ok := cfg.Targets["local"]
	if !ok {
		t.Fatal("Expected target 'local' not found")
	}
	if local.LocalPath != "/local/skills" {
		t.Errorf("local.LocalPath = %q, want %q", local.LocalPath, "/local/skills")
	}
}

func TestLoad_Neither(t *testing.T) {
	tmpDir := t.TempDir()

	loader := &Loader{
		scope:      ScopeLocal,
		globalPath: filepath.Join(tmpDir, "global.toml"), // doesn't exist
	}

	// Change to tmpDir so DetectLocalPath won't find anything
	oldCwd, _ := os.Getwd()
	defer os.Chdir(oldCwd)
	os.Chdir(tmpDir)

	cfg, err := loader.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	// Should use defaults
	home, _ := os.UserHomeDir()
	expectedCache := filepath.Join(home, ".cache", "skillforge", "repos")
	if cfg.Cache.Path != expectedCache {
		t.Errorf("Cache.Path = %q, want default %q", cfg.Cache.Path, expectedCache)
	}

	if len(cfg.Targets) != 0 {
		t.Errorf("Targets count = %d, want 0", len(cfg.Targets))
	}
}

// TestLoad_LocalOnlyScope verifies that ScopeLocal only loads local config.
func TestLoad_LocalOnlyScope(t *testing.T) {
	tmpDir := t.TempDir()
	globalPath := filepath.Join(tmpDir, "global.toml")
	localDir := filepath.Join(tmpDir, ".skillforge")
	if err := os.MkdirAll(localDir, 0755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	localPath := filepath.Join(localDir, "config.toml")

	// Write global config
	globalCfg := `cache.path = "/global/cache"

[targets.global-target]
globalPath = "/global/only"
localPath = "/local/only"
enabled = true

[repos.global-repo]
url = "https://github.com/test/grimoire"
branch = "main"
updated = "2026-04-28"
`
	if err := os.WriteFile(globalPath, []byte(globalCfg), 0644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	// Write local config
	localCfg := `cache.path = "/local/cache"

[targets.local-target]
globalPath = "/global/local"
localPath = "/local/skills"
enabled = true

[repos.local-repo]
url = "https://github.com/test/local"
branch = "develop"
updated = "2026-04-27"
`
	if err := os.WriteFile(localPath, []byte(localCfg), 0644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	// Change to tmpDir so DetectLocalPath finds it
	oldCwd, _ := os.Getwd()
	defer os.Chdir(oldCwd)
	os.Chdir(tmpDir)

	loader := &Loader{
		scope:      ScopeLocal,
		globalPath: globalPath,
	}

	cfg, err := loader.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	// Should only have local cache
	if cfg.Cache.Path != "/local/cache" {
		t.Errorf("Cache.Path = %q, want %q (local only)", cfg.Cache.Path, "/local/cache")
	}

	// Should NOT have global-target (only in global config)
	if _, ok := cfg.Targets["global-target"]; ok {
		t.Error("Should not have target 'global-target' when loading local only")
	}

	// Should have local-target from local config
	if _, ok := cfg.Targets["local-target"]; !ok {
		t.Error("Expected target 'local-target' not found")
	}

	// Should NOT have global repo
	if _, ok := cfg.Repos["global-repo"]; ok {
		t.Error("Should not have repo 'global-repo' when loading local only")
	}
	// Should have local repo
	if _, ok := cfg.Repos["local-repo"]; !ok {
		t.Error("Expected repo 'local-repo' not found")
	}
}

// TestLoad_LocalScopeTarget verifies that ScopeLocal correctly loads a target from local config.
func TestLoad_LocalScopeTarget(t *testing.T) {
	tmpDir := t.TempDir()
	globalPath := filepath.Join(tmpDir, "global.toml")
	localDir := filepath.Join(tmpDir, ".skillforge")
	if err := os.MkdirAll(localDir, 0755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	localPath := filepath.Join(localDir, "config.toml")

	// Write global config
	globalCfg := `cache.path = "/global/cache"

[targets.pi]
globalPath = "/global/pi"
localPath = "/local/pi"
enabled = true
`
	if err := os.WriteFile(globalPath, []byte(globalCfg), 0644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	// Write local config with same target
	localCfg := `cache.path = "/local/cache"

[targets.pi]
globalPath = "/global/pi"
localPath = "/local/pi"
enabled = false
`
	if err := os.WriteFile(localPath, []byte(localCfg), 0644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	// Change to tmpDir
	oldCwd, _ := os.Getwd()
	defer os.Chdir(oldCwd)
	os.Chdir(tmpDir)

	loader := &Loader{
		scope:      ScopeLocal,
		globalPath: globalPath,
	}

	cfg, err := loader.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	// Should only have local cache
	if cfg.Cache.Path != "/local/cache" {
		t.Errorf("Cache.Path = %q, want %q", cfg.Cache.Path, "/local/cache")
	}

	// Should have pi from local config with local values
	pi, ok := cfg.Targets["pi"]
	if !ok {
		t.Fatal("Expected target 'pi' not found")
	}
	if pi.GlobalPath != "/global/pi" {
		t.Errorf("pi.GlobalPath = %q, want %q", pi.GlobalPath, "/global/pi")
	}
	if pi.Enabled {
		t.Error("pi.Enabled = true, want false")
	}
}

// TestLoad_GlobalOnlyScope verifies that ScopeGlobal only loads global config.
func TestLoad_GlobalOnlyScope(t *testing.T) {
	tmpDir := t.TempDir()
	globalPath := filepath.Join(tmpDir, "global.toml")
	localDir := filepath.Join(tmpDir, ".skillforge")
	if err := os.MkdirAll(localDir, 0755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	localPath := filepath.Join(localDir, "config.toml")

	// Write global config
	globalCfg := `cache.path = "/global/cache"

[targets.global-target]
globalPath = "/global/only"
localPath = "/local/only"
enabled = true

[repos.global-repo]
url = "https://github.com/test/grimoire"
branch = "main"
updated = "2026-04-28"
`
	if err := os.WriteFile(globalPath, []byte(globalCfg), 0644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	// Write local config
	localCfg := `cache.path = "/local/cache"

[targets.local-target]
globalPath = "/global/local"
localPath = "/local/skills"
enabled = true

[repos.local-repo]
url = "https://github.com/test/local"
branch = "develop"
updated = "2026-04-27"
`
	if err := os.WriteFile(localPath, []byte(localCfg), 0644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	// Change to tmpDir so DetectLocalPath finds it
	oldCwd, _ := os.Getwd()
	defer os.Chdir(oldCwd)
	os.Chdir(tmpDir)

	loader := &Loader{
		scope:      ScopeGlobal,
		globalPath: globalPath,
	}

	cfg, err := loader.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	// Should only have global cache
	if cfg.Cache.Path != "/global/cache" {
		t.Errorf("Cache.Path = %q, want %q (global only)", cfg.Cache.Path, "/global/cache")
	}

	// Should have global-target from global config
	if _, ok := cfg.Targets["global-target"]; !ok {
		t.Error("Expected target 'global-target' not found")
	}

	// Should NOT have local-target (only in local config)
	if _, ok := cfg.Targets["local-target"]; ok {
		t.Error("Should not have target 'local-target' when loading global only")
	}

	// Should have global repo
	if _, ok := cfg.Repos["global-repo"]; !ok {
		t.Error("Expected repo 'global-repo' not found")
	}
	// Should NOT have local repo
	if _, ok := cfg.Repos["local-repo"]; ok {
		t.Error("Should not have repo 'local-repo' when loading global only")
	}
}

func TestSave_LocalPreservesGlobal(t *testing.T) {
	tmpDir := t.TempDir()
	globalPath := filepath.Join(tmpDir, "global.toml")
	localDir := filepath.Join(tmpDir, ".skillforge")
	if err := os.MkdirAll(localDir, 0755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	localPath := filepath.Join(localDir, "config.toml")

	// Write global config
	globalCfg := `[targets.pi]
globalPath = "/global/pi"
localPath = "/local/pi"
enabled = true

[repos.grimoire]
url = "https://github.com/rwese/agents-grimoire"
branch = "main"
`
	if err := os.WriteFile(globalPath, []byte(globalCfg), 0644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	// Write existing local config
	localCfg := `[targets.local]
globalPath = "/global/local"
localPath = "/local/skills"
enabled = true
`
	if err := os.WriteFile(localPath, []byte(localCfg), 0644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	// Change to tmpDir
	oldCwd, _ := os.Getwd()
	defer os.Chdir(oldCwd)
	os.Chdir(tmpDir)

	loader := &Loader{
		scope:      ScopeLocal,
		globalPath: globalPath,
	}

	// Save a new target locally
	cfg := &Config{
		Targets: map[string]Target{
			"local": {
				Name:    "local",
				GlobalPath: "/local/skills",
				Enabled: true,
			},
			"new-local": {
				Name:    "new-local",
				GlobalPath: "/new/local",
				Enabled: true,
			},
		},
	}

	if err := loader.Save(cfg, true); err != nil {
		t.Fatalf("Save(local=true) error = %v", err)
	}

	// Global should still have pi and grimoire
	loader2 := &Loader{
		scope:      ScopeGlobal,
		globalPath: globalPath,
	}
	globalCfg2, err := loader2.Load()
	if err != nil {
		t.Fatalf("Load(global) error = %v", err)
	}

	if _, ok := globalCfg2.Targets["pi"]; !ok {
		t.Error("Global target 'pi' was removed after local save")
	}
	if _, ok := globalCfg2.Repos["grimoire"]; !ok {
		t.Error("Global repo 'grimoire' was removed after local save")
	}
}

func TestSave_GlobalPreservesLocal(t *testing.T) {
	tmpDir := t.TempDir()
	globalPath := filepath.Join(tmpDir, "global.toml")
	localDir := filepath.Join(tmpDir, ".skillforge")
	if err := os.MkdirAll(localDir, 0755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	localPath := filepath.Join(localDir, "config.toml")

	// Write global config
	globalCfg := `[targets.pi]
globalPath = "/global/pi"
localPath = "/local/pi"
enabled = true
`
	if err := os.WriteFile(globalPath, []byte(globalCfg), 0644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	// Write local config
	localCfg := `[targets.local]
globalPath = "/global/local"
localPath = "/local/skills"
enabled = true

[repos.local-repo]
url = "https://github.com/test/local"
branch = "main"
`
	if err := os.WriteFile(localPath, []byte(localCfg), 0644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	// Change to tmpDir
	oldCwd, _ := os.Getwd()
	defer os.Chdir(oldCwd)
	os.Chdir(tmpDir)

	loader := &Loader{
		scope:      ScopeGlobal,
		globalPath: globalPath,
	}

	// Save a new target globally
	cfg := &Config{
		Targets: map[string]Target{
			"pi": {
				Name:    "pi",
				GlobalPath: "/global/pi",
				Enabled: true,
			},
			"new-global": {
				Name:    "new-global",
				GlobalPath: "/new/global",
				Enabled: true,
			},
		},
	}

	if err := loader.Save(cfg, false); err != nil {
		t.Fatalf("Save(global) error = %v", err)
	}

	// Local should still have local target and repo
	loader3 := &Loader{
		scope:      ScopeLocal,
		globalPath: globalPath,
	}
	localCfg3, err := loader3.Load()
	if err != nil {
		t.Fatalf("Load(local) error = %v", err)
	}

	if _, ok := localCfg3.Targets["local"]; !ok {
		t.Error("Local target 'local' was removed after global save")
	}
	if _, ok := localCfg3.Repos["local-repo"]; !ok {
		t.Error("Local repo 'local-repo' was removed after global save")
	}
}

func TestLoadAllRepos(t *testing.T) {
	// Create temp directories
	tmpDir, err := os.MkdirTemp("", "skillforge-test-*")
	if err != nil {
		t.Fatalf("MkdirTemp() error = %v", err)
	}
	defer os.RemoveAll(tmpDir)

	globalPath := filepath.Join(tmpDir, "global.toml")
	localDir := filepath.Join(tmpDir, "project")
	localPath := filepath.Join(localDir, ".skillforge", "config.toml")

	// Write global config with repos
	globalCfg := `
[repos.global-repo]
url = "https://github.com/global/repo"
branch = "main"
`
	if err := os.WriteFile(globalPath, []byte(globalCfg), 0644); err != nil {
		t.Fatalf("WriteFile(global) error = %v", err)
	}

	// Write local config with different repos
	if err := os.MkdirAll(filepath.Dir(localPath), 0755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	localCfg := `
[repos.local-repo]
url = "https://github.com/local/repo"
branch = "develop"
`
	if err := os.WriteFile(localPath, []byte(localCfg), 0644); err != nil {
		t.Fatalf("WriteFile(local) error = %v", err)
	}

	// Change to tmpDir
	oldCwd, _ := os.Getwd()
	defer os.Chdir(oldCwd)
	os.Chdir(localDir)

	loader := &Loader{
		scope:      ScopeLocal, // Should still load both
		globalPath: globalPath,
	}

	repos, _, err := loader.LoadAllRepos()
	if err != nil {
		t.Fatalf("LoadAllRepos() error = %v", err)
	}

	// Should have both repos
	if _, ok := repos["global-repo"]; !ok {
		t.Error("LoadAllRepos() missing global-repo")
	}
	if _, ok := repos["local-repo"]; !ok {
		t.Error("LoadAllRepos() missing local-repo")
	}

	// Local should override global if same name
	localOverrideCfg := `
[repos.global-repo]
url = "https://github.com/local/override"
`
	if err := os.WriteFile(localPath, []byte(localOverrideCfg), 0644); err != nil {
		t.Fatalf("WriteFile(local override) error = %v", err)
	}

	repos2, _, err := loader.LoadAllRepos()
	if err != nil {
		t.Fatalf("LoadAllRepos() error = %v", err)
	}

	if repos2["global-repo"].URL != "https://github.com/local/override" {
		t.Errorf("LoadAllRepos() local override not applied, got %s", repos2["global-repo"].URL)
	}
}

// --- Regression Tests ---

// TestSave_LocalRemoveTarget verifies that deleting a target actually removes it.
func TestSave_LocalRemoveTarget(t *testing.T) {
	tmpDir := t.TempDir()
	localDir := filepath.Join(tmpDir, ".skillforge")
	if err := os.MkdirAll(localDir, 0755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	localPath := filepath.Join(localDir, "config.toml")

	// Write initial config with two targets
	initialCfg := `[cache]
  path = "/cache"

[targets.target-a]
globalPath = "/path/a/global"
localPath = "/path/a"
enabled = true

[targets.target-b]
globalPath = "/path/b/global"
localPath = "/path/b"
enabled = true
`
	if err := os.WriteFile(localPath, []byte(initialCfg), 0644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	// Change to tmpDir
	oldCwd, _ := os.Getwd()
	defer os.Chdir(oldCwd)
	os.Chdir(tmpDir)

	loader := &Loader{scope: ScopeLocal}

	// Simulate removing target-b
	cfg := &Config{
		Cache: CacheConfig{Path: "/cache"},
		Targets: map[string]Target{
			"target-a": {
				Name:       "target-a",
				GlobalPath: "/path/a/global",
				LocalPath:  "/path/a",
				Enabled:    true,
			},
		},
	}

	if err := loader.Save(cfg, true); err != nil {
		t.Fatalf("Save(local=true) error = %v", err)
	}

	loader2 := &Loader{scope: ScopeLocal}
	loadedCfg, err := loader2.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if _, ok := loadedCfg.Targets["target-a"]; !ok {
		t.Error("target-a should still exist")
	}
	if _, ok := loadedCfg.Targets["target-b"]; ok {
		t.Error("REGRESSION: target-b was not removed")
	}
}

// TestSave_LocalRemoveAllTargets verifies removing all targets works.
func TestSave_LocalRemoveAllTargets(t *testing.T) {
	tmpDir := t.TempDir()
	localDir := filepath.Join(tmpDir, ".skillforge")
	if err := os.MkdirAll(localDir, 0755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	localPath := filepath.Join(localDir, "config.toml")

	initialCfg := `[cache]
  path = "/cache"

[targets.target-1]
globalPath = "/path/1/global"
localPath = "/path/1"
enabled = true

[targets.target-2]
globalPath = "/path/2/global"
localPath = "/path/2"
enabled = true
`
	if err := os.WriteFile(localPath, []byte(initialCfg), 0644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	oldCwd, _ := os.Getwd()
	defer os.Chdir(oldCwd)
	os.Chdir(tmpDir)

	loader := &Loader{scope: ScopeLocal}
	cfg := &Config{Cache: CacheConfig{Path: "/cache"}, Targets: map[string]Target{}}

	if err := loader.Save(cfg, true); err != nil {
		t.Fatalf("Save(local=true) error = %v", err)
	}

	loader2 := &Loader{scope: ScopeLocal}
	loadedCfg, err := loader2.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if len(loadedCfg.Targets) != 0 {
		t.Errorf("REGRESSION: Expected 0 targets, got %d", len(loadedCfg.Targets))
	}
}

// TestSave_LocalPreservesCache verifies cache path is preserved.
func TestSave_LocalPreservesCache(t *testing.T) {
	tmpDir := t.TempDir()
	localDir := filepath.Join(tmpDir, ".skillforge")
	if err := os.MkdirAll(localDir, 0755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	localPath := filepath.Join(localDir, "config.toml")

	initialCfg := `[cache]
  path = "/original/cache"

[targets.target-a]
globalPath = "/path/global"
localPath = "/path"
enabled = true
`
	if err := os.WriteFile(localPath, []byte(initialCfg), 0644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	oldCwd, _ := os.Getwd()
	defer os.Chdir(oldCwd)
	os.Chdir(tmpDir)

	loader := &Loader{scope: ScopeLocal}
	cfg := &Config{Cache: CacheConfig{Path: ""}, Targets: map[string]Target{}}

	if err := loader.Save(cfg, true); err != nil {
		t.Fatalf("Save(local=true) error = %v", err)
	}

	loader2 := &Loader{scope: ScopeLocal}
	loadedCfg, err := loader2.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if loadedCfg.Cache.Path != "/original/cache" {
		t.Errorf("Cache.Path = %q, want %q", loadedCfg.Cache.Path, "/original/cache")
	}
}
