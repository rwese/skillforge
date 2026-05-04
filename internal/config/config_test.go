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
		{"auto scope", ScopeAuto},
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
	loader := NewLoader(ScopeAuto)
	cfg, err := loader.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	// Should load global config (with defaults if no file)
	if cfg.Cache.Path == "" {
		t.Error("Load() returned empty Cache.Path for ScopeAuto")
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
				Name:    "test-target",
				Path:    "/test/path",
				Enabled: true,
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
		scope: ScopeAuto,
	}

	cfg := &Config{
		Cache: CacheConfig{
			Path: "/cache",
		},
		Targets: map[string]Target{
			"local-target": {
				Name:    "local-target",
				Path:    "/local/path",
				Enabled: true,
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
		want  int
	}{
		{"ScopeGlobal", ScopeGlobal, 0},
		{"ScopeLocal", ScopeLocal, 1},
		{"ScopeAuto", ScopeAuto, 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if int(tt.scope) != tt.want {
				t.Errorf("Scope %s = %d, want %d", tt.name, int(tt.scope), tt.want)
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
path = "/global/pi"
enabled = true
`
	if err := os.WriteFile(globalPath, []byte(globalCfg), 0644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	loader := &Loader{
		scope:      ScopeAuto,
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
	if pi.Path != "/global/pi" {
		t.Errorf("pi.Path = %q, want %q", pi.Path, "/global/pi")
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
path = "/local/skills"
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
		scope:      ScopeAuto,
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
	if local.Path != "/local/skills" {
		t.Errorf("local.Path = %q, want %q", local.Path, "/local/skills")
	}
}

func TestLoad_Neither(t *testing.T) {
	tmpDir := t.TempDir()

	loader := &Loader{
		scope:      ScopeAuto,
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

func TestLoad_BothMerge(t *testing.T) {
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
path = "/global/pi"
enabled = true

[targets.global-only]
path = "/global/only"
enabled = true

[repos.grimoire]
url = "https://github.com/rwese/agents-grimoire"
branch = "main"
updated = "2026-04-28"
`
	if err := os.WriteFile(globalPath, []byte(globalCfg), 0644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	// Write local config
	localCfg := `cache.path = "/local/cache"

[targets.local]
path = "/local/skills"
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
		scope:      ScopeAuto,
		globalPath: globalPath,
	}

	cfg, err := loader.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	// Local should override global cache.path
	if cfg.Cache.Path != "/local/cache" {
		t.Errorf("Cache.Path = %q, want %q (local override)", cfg.Cache.Path, "/local/cache")
	}

	// Should have global-only target
	if _, ok := cfg.Targets["global-only"]; !ok {
		t.Error("Expected target 'global-only' not found")
	}

	// Should have pi from global
	if _, ok := cfg.Targets["pi"]; !ok {
		t.Error("Expected target 'pi' not found")
	}

	// Should have local target
	if _, ok := cfg.Targets["local"]; !ok {
		t.Error("Expected target 'local' not found")
	}

	// Should have both repos
	if _, ok := cfg.Repos["grimoire"]; !ok {
		t.Error("Expected repo 'grimoire' not found")
	}
	if _, ok := cfg.Repos["local-repo"]; !ok {
		t.Error("Expected repo 'local-repo' not found")
	}
}

func TestLoad_LocalOverrides(t *testing.T) {
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
path = "/global/pi"
enabled = true
`
	if err := os.WriteFile(globalPath, []byte(globalCfg), 0644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	// Write local config with same target
	localCfg := `cache.path = "/local/cache"

[targets.pi]
path = "/local/pi"
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
		scope:      ScopeAuto,
		globalPath: globalPath,
	}

	cfg, err := loader.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	// Local should override global cache.path
	if cfg.Cache.Path != "/local/cache" {
		t.Errorf("Cache.Path = %q, want %q", cfg.Cache.Path, "/local/cache")
	}

	// Local should override pi's path and enabled
	pi, ok := cfg.Targets["pi"]
	if !ok {
		t.Fatal("Expected target 'pi' not found")
	}
	if pi.Path != "/local/pi" {
		t.Errorf("pi.Path = %q, want %q (local override)", pi.Path, "/local/pi")
	}
	if pi.Enabled {
		t.Error("pi.Enabled = true, want false (local override)")
	}
}

func TestLoad_DifferentKeys(t *testing.T) {
	tmpDir := t.TempDir()
	globalPath := filepath.Join(tmpDir, "global.toml")
	localDir := filepath.Join(tmpDir, ".skillforge")
	if err := os.MkdirAll(localDir, 0755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	localPath := filepath.Join(localDir, "config.toml")

	// Write global config with cache and pi target
	globalCfg := `cache.path = "/global/cache"

[targets.pi]
path = "/global/pi"
enabled = true
`
	if err := os.WriteFile(globalPath, []byte(globalCfg), 0644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	// Write local config with different target (no cache override)
	localCfg := `
[targets.local]
path = "/local/skills"
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
		scope:      ScopeAuto,
		globalPath: globalPath,
	}

	cfg, err := loader.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	// Should have global cache (local didn't override)
	if cfg.Cache.Path != "/global/cache" {
		t.Errorf("Cache.Path = %q, want %q (preserved from global)", cfg.Cache.Path, "/global/cache")
	}

	// Should have pi from global
	if _, ok := cfg.Targets["pi"]; !ok {
		t.Error("Expected target 'pi' not found (preserved from global)")
	}

	// Should have local from local
	if _, ok := cfg.Targets["local"]; !ok {
		t.Error("Expected target 'local' not found")
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
path = "/global/pi"
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
path = "/local/skills"
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
		scope:      ScopeAuto,
		globalPath: globalPath,
	}

	// Save a new target locally
	cfg := &Config{
		Targets: map[string]Target{
			"local": {
				Name:    "local",
				Path:    "/local/skills",
				Enabled: true,
			},
			"new-local": {
				Name:    "new-local",
				Path:    "/new/local",
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
path = "/global/pi"
enabled = true
`
	if err := os.WriteFile(globalPath, []byte(globalCfg), 0644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	// Write local config
	localCfg := `[targets.local]
path = "/local/skills"
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
				Path:    "/global/pi",
				Enabled: true,
			},
			"new-global": {
				Name:    "new-global",
				Path:    "/new/global",
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
