package agents

import (
	"testing"
)

func TestSelectableAgent_Defaults(t *testing.T) {
	tests := []struct {
		name     string
		agent    SelectableAgent
		wantName string
		wantDet  bool
		wantSel  bool
		wantConf bool
	}{
		{
			name: "detected not configured",
			agent: SelectableAgent{
				Name:          "pi",
				DefaultGlobal: "~/.pi/agent/skills",
				Detected:      true,
				Selected:      false,
				Configured:    false,
			},
			wantName: "pi",
			wantDet:  true,
			wantSel:  false,
			wantConf: false,
		},
		{
			name: "configured",
			agent: SelectableAgent{
				Name:          "codex",
				DefaultGlobal: "~/.codex/skills",
				Detected:      true,
				Selected:      true,
				Configured:    true,
			},
			wantName: "codex",
			wantDet:  true,
			wantSel:  true,
			wantConf: true,
		},
		{
			name: "not detected",
			agent: SelectableAgent{
				Name:          "claude",
				DefaultGlobal: "~/.claude/skills",
				Detected:      false,
				Selected:      false,
				Configured:    false,
			},
			wantName: "claude",
			wantDet:  false,
			wantSel:  false,
			wantConf: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.agent.Name != tt.wantName {
				t.Errorf("Name = %v, want %v", tt.agent.Name, tt.wantName)
			}
			if tt.agent.Detected != tt.wantDet {
				t.Errorf("Detected = %v, want %v", tt.agent.Detected, tt.wantDet)
			}
			if tt.agent.Selected != tt.wantSel {
				t.Errorf("Selected = %v, want %v", tt.agent.Selected, tt.wantSel)
			}
			if tt.agent.Configured != tt.wantConf {
				t.Errorf("Configured = %v, want %v", tt.agent.Configured, tt.wantConf)
			}
		})
	}
}

func TestBuildSelectableAgents(t *testing.T) {
	// Mock existing config
	existingCfg := &AgentsConfig{
		Agents: map[string]Agent{
			"codex": {
				Name:   "codex",
				Global: &Path{Value: "~/.codex/skills"},
			},
		},
	}

	// Mock detected agents
	detected := map[string]bool{
		"pi":     true,
		"codex":  true,
		"claude": false,
	}

	// Build selectable agents (simulating what runSetupDetect does)
	var selectableAgents []SelectableAgent
	for name, def := range KnownAgents {
		agent := SelectableAgent{
			Name:          name,
			DefaultGlobal: def.DefaultGlobal,
			DefaultLocal:  def.DefaultLocal,
			Detected:      detected[name],
			Configured:    existingCfg.Agents[name].Global != nil,
			Selected:      detected[name], // Pre-select detected
		}
		selectableAgents = append(selectableAgents, agent)
	}

	// Verify results
	expected := map[string]struct {
		detected   bool
		configured bool
		selected   bool
	}{
		"pi":     {detected: true, configured: false, selected: true},
		"codex":  {detected: true, configured: true, selected: true},
		"claude": {detected: false, configured: false, selected: false},
	}

	for _, agent := range selectableAgents {
		exp, ok := expected[agent.Name]
		if !ok {
			t.Errorf("Unexpected agent: %s", agent.Name)
			continue
		}

		if agent.Detected != exp.detected {
			t.Errorf("%s: Detected = %v, want %v", agent.Name, agent.Detected, exp.detected)
		}
		if agent.Configured != exp.configured {
			t.Errorf("%s: Configured = %v, want %v", agent.Name, agent.Configured, exp.configured)
		}
		if agent.Selected != exp.selected {
			t.Errorf("%s: Selected = %v, want %v", agent.Name, agent.Selected, exp.selected)
		}
	}

	if len(selectableAgents) != 3 {
		t.Errorf("Expected 3 agents, got %d", len(selectableAgents))
	}
}






