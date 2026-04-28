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
