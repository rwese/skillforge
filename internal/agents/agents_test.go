package agents

import (
	"os"
	"path/filepath"
	"testing"
)

func TestExpandPath(t *testing.T) {
	home, _ := os.UserHomeDir()

	tests := []struct {
		input    string
		expected string
	}{
		{"~/.pi/agent/skills", filepath.Join(home, ".pi/agent/skills")},
		{"/absolute/path", "/absolute/path"},
		{"./relative/path", "./relative/path"},
	}

	for _, tt := range tests {
		got := ExpandPath(tt.input)
		if got != tt.expected {
			t.Errorf("ExpandPath(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}

func TestContractPath(t *testing.T) {
	home, _ := os.UserHomeDir()

	tests := []struct {
		input    string
		expected string
	}{
		{filepath.Join(home, ".pi/agent/skills"), "~/.pi/agent/skills"},
		{"/absolute/path", "/absolute/path"},
		{"./relative/path", "./relative/path"},
	}

	for _, tt := range tests {
		got := ContractPath(tt.input)
		if got != tt.expected {
			t.Errorf("ContractPath(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}

func TestDetectAgents(t *testing.T) {
	// This test checks that DetectAgents runs without error
	// and returns a map (may be empty or have entries depending on system)
	detected := DetectAgents()
	if detected == nil {
		t.Error("DetectAgents() returned nil, want map")
	}
}

func TestLoadAgents_Empty(t *testing.T) {
	// Test loading when no config exists
	cfg, err := LoadAgents()
	if err != nil {
		t.Fatalf("LoadAgents() error = %v", err)
	}
	if cfg == nil {
		t.Fatal("LoadAgents() returned nil")
	}
	if cfg.Agents == nil {
		t.Fatal("LoadAgents().Agents is nil")
	}
}

func TestMergeAgents(t *testing.T) {
	globalCfg := &AgentsConfig{
		Agents: map[string]Agent{
			"pi": {
				Name:   "pi",
				Global: &Path{Value: "~/.pi/agent/skills"},
				Local:  &Path{Value: ".pi/skills"},
			},
		},
	}

	localCfg := &AgentsConfig{
		Agents: map[string]Agent{
			"pi": {
				// Override only global, keep local
				Global: &Path{Value: "/custom/global"},
			},
			"codex": {
				// New agent from local
				Name:   "codex",
				Global: &Path{Value: "~/.codex/skills"},
				Local:  &Path{Value: ".codex/skills"},
			},
		},
	}

	mergeAgents(globalCfg, localCfg)

	// Check pi global was overridden
	if globalCfg.Agents["pi"].Global.Value != "/custom/global" {
		t.Errorf("pi.Global = %q, want %q", globalCfg.Agents["pi"].Global.Value, "/custom/global")
	}

	// Check pi local was preserved
	if globalCfg.Agents["pi"].Local == nil {
		t.Error("pi.Local was nil, want preserved")
	}

	// Check codex was added
	if _, exists := globalCfg.Agents["codex"]; !exists {
		t.Error("codex not in globalCfg.Agents after merge")
	}
}

func TestSaveAgents_Global(t *testing.T) {
	tmpDir := t.TempDir()
	home := filepath.Join(tmpDir, "home")
	os.MkdirAll(home, 0755)

	// Override home for this test
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", home)
	defer os.Setenv("HOME", origHome)

	cfg := &AgentsConfig{
		Agents: map[string]Agent{
			"pi": {
				Name:   "pi",
				Global: &Path{Value: "~/.pi/agent/skills"},
			},
		},
	}

	err := SaveAgents(cfg, ScopeGlobal)
	if err != nil {
		t.Fatalf("SaveAgents(ScopeGlobal) error = %v", err)
	}

	// Verify file was created
	agentsPath := filepath.Join(home, ".config", "skillforge", "agents.toml")
	if _, err := os.Stat(agentsPath); os.IsNotExist(err) {
		t.Errorf("agents.toml not created at %s", agentsPath)
	}

	// Verify content
	data, _ := os.ReadFile(agentsPath)
	if len(data) == 0 {
		t.Error("agents.toml is empty")
	}
}

func TestKnownAgents(t *testing.T) {
	for name, def := range KnownAgents {
		if def.Name != name {
			t.Errorf("KnownAgents[%q].Name = %q, want %q", name, def.Name, name)
		}
		if def.DefaultGlobal == "" {
			t.Errorf("KnownAgents[%q].DefaultGlobal is empty", name)
		}
		if def.DefaultLocal == "" {
			t.Errorf("KnownAgents[%q].DefaultLocal is empty", name)
		}
		if len(def.DetectionPaths) == 0 {
			t.Errorf("KnownAgents[%q].DetectionPaths is empty", name)
		}
	}
}
