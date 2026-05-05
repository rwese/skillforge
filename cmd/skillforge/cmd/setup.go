package cmd

import (
	"fmt"
	"os"

	"github.com/rwese/skillforge/internal/agents"
	"github.com/spf13/cobra"
)

// setupCmd represents the setup command group.
var setupCmd = &cobra.Command{
	Use:   "setup",
	Short: "Setup skillforge for agents",
	Long: `Setup wizard to configure skill directories for known agents.

Detects common agents (pi, codex, claude) and helps configure their skill paths.`,
}

func init() {
	rootCmd.AddCommand(setupCmd)
	setupCmd.AddCommand(setupDetectCmd)
	setupCmd.AddCommand(setupListCmd)
	setupCmd.AddCommand(setupAddCmd)
}

// setupDetectCmd runs the detection wizard.
var setupDetectCmd = &cobra.Command{
	Use:   "detect",
	Short: "Detect and configure known agents",
	RunE:  runSetupDetect,
}

func runSetupDetect(cmd *cobra.Command, args []string) error {
	// Load existing config to know which agents are already configured
	existingCfg, err := agents.LoadAgents()
	if err != nil {
		return fmt.Errorf("loading existing config: %w", err)
	}

	// Detect which agent skill directories exist
	detected := agents.DetectAgents()

	// Build selectable agent list
	var selectableAgents []agents.SelectableAgent
	for name, def := range agents.KnownAgents {
		agent := agents.SelectableAgent{
			Name:          name,
			DefaultGlobal: def.DefaultGlobal,
			DefaultLocal:  def.DefaultLocal,
			Detected:      detected[name],
			Configured:    existingCfg.Agents[name].Global != nil,
			// Pre-select detected agents (default paths exist)
			Selected: detected[name],
		}
		selectableAgents = append(selectableAgents, agent)
	}

	if len(selectableAgents) == 0 {
		fmt.Println("No known agents configured.")
		fmt.Println("\nYou can add agents manually with:")
		fmt.Printf("  %s setup add <name>\n", cmdName())
		return nil
	}

	// With --yes flag, skip interactive selector and use pre-selected agents
	var selectedNames []string
	if yesFlag {
		for _, agent := range selectableAgents {
			if agent.Selected {
				selectedNames = append(selectedNames, agent.Name)
			}
		}
		if len(selectedNames) == 0 {
			fmt.Println("No agents selected (use interactive mode to select agents).")
			return nil
		}
	} else {
		// Run interactive selector
		selectedNames = agents.SelectAgents(selectableAgents)

		// User quit
		if selectedNames == nil {
			return fmt.Errorf("cancelled")
		}

		// No agents selected
		if len(selectedNames) == 0 {
			fmt.Println("No agents selected. Exiting.")
			return nil
		}
	}

	// Build selected agents config
	selectedCfg := &agents.AgentsConfig{
		Agents: make(map[string]agents.Agent),
	}
	for _, name := range selectedNames {
		def := agents.KnownAgents[name]
		selectedCfg.Agents[name] = agents.Agent{
			Name:   name,
			Global: &agents.Path{Value: def.DefaultGlobal},
			Local:  &agents.Path{Value: def.DefaultLocal},
		}
	}

	// Show summary unless --yes flag is set
	if !yesFlag {
		fmt.Println("\n" + Highlight.Render("Selected agents to configure:"))
		for name, agent := range selectedCfg.Agents {
			fmt.Printf("  • %s\n", name)
			if agent.Global != nil {
				fmt.Printf("      global: %s\n", agents.ContractPath(agents.ExpandPath(agent.Global.Value)))
			}
			if agent.Local != nil {
				fmt.Printf("      local:  %s\n", agents.ContractPath(agents.ExpandPath(agent.Local.Value)))
			}
		}

		fmt.Println()
		if !confirm("Save to global config?") {
			return fmt.Errorf("cancelled")
		}
	}

	// Merge with existing config and save
	mergedCfg := &agents.AgentsConfig{
		Agents: make(map[string]agents.Agent),
	}

	// Start with existing
	for k, v := range existingCfg.Agents {
		mergedCfg.Agents[k] = v
	}

	// Override selected agents
	for k, v := range selectedCfg.Agents {
		mergedCfg.Agents[k] = v
	}

	// Save to global config
	if err := agents.SaveAgents(mergedCfg, agents.ScopeGlobal); err != nil {
		return fmt.Errorf("saving agents config: %w", err)
	}

	fmt.Println("\n" + Success.Render("✓") + " Agents configured successfully.")
	return nil
}

// setupListCmd lists configured agents.
var setupListCmd = &cobra.Command{
	Use:   "list",
	Short: "List configured agents",
	RunE:  runSetupList,
}

func runSetupList(cmd *cobra.Command, args []string) error {
	cfg, err := agents.LoadAgents()
	if err != nil {
		return fmt.Errorf("loading agents: %w", err)
	}

	if len(cfg.Agents) == 0 {
		fmt.Println("No agents configured.")
		fmt.Printf("\nRun '%s setup detect' to auto-detect known agents.\n", cmdName())
		fmt.Printf("Or '%s setup add <name>' to add an agent manually.\n", cmdName())
		return nil
	}

	fmt.Printf("Configured agents (%d):\n", len(cfg.Agents))
	for name, agent := range cfg.Agents {
		fmt.Printf("  • %s\n", name)
		if agent.Global != nil {
			fmt.Printf("      global: %s\n", agents.ContractPath(agents.ExpandPath(agent.Global.Value)))
		}
		if agent.Local != nil {
			fmt.Printf("      local:  %s\n", agents.ContractPath(agents.ExpandPath(agent.Local.Value)))
		}
	}

	return nil
}

// setupAddCmd adds a new agent manually.
var setupAddCmd = &cobra.Command{
	Use:   "add [agent-name]",
	Short: "Add an agent manually",
	Args:  cobra.ExactArgs(1),
	RunE:  runSetupAdd,
}

var (
	globalPathFlag string
	localPathFlag  string
)

func init() {
	setupAddCmd.Flags().StringVar(&globalPathFlag, "global", "", "Global skills path (e.g., ~/.pi/agent/skills)")
	setupAddCmd.Flags().StringVar(&localPathFlag, "local", "", "Local skills path (e.g., .pi/skills)")
}

func runSetupAdd(cmd *cobra.Command, args []string) error {
	name := args[0]

	// Check if it's a known agent with defaults
	def, known := agents.KnownAgents[name]

	// Prompt for paths if not provided
	if globalPathFlag == "" {
		if known {
			fmt.Printf("Global path [%s]: ", agents.ContractPath(agents.ExpandPath(def.DefaultGlobal)))
		} else {
			fmt.Print("Global path: ")
		}
		var input string
		fmt.Scanln(&input)

		if input == "" && known {
			globalPathFlag = def.DefaultGlobal
		} else if input != "" {
			globalPathFlag = input
		}
	}

	if localPathFlag == "" {
		if known {
			fmt.Printf("Local path [%s]: ", agents.ContractPath(agents.ExpandPath(def.DefaultLocal)))
		} else {
			fmt.Print("Local path (optional, press enter to skip): ")
		}
		var input string
		fmt.Scanln(&input)

		if input == "" && known {
			localPathFlag = def.DefaultLocal
		}
	}

	// Build agent config
	agent := agents.Agent{
		Name: name,
	}

	if globalPathFlag != "" {
		agent.Global = &agents.Path{Value: globalPathFlag}
	}

	if localPathFlag != "" {
		agent.Local = &agents.Path{Value: localPathFlag}
	}

	// Load existing and merge
	cfg, err := agents.LoadAgents()
	if err != nil {
		return fmt.Errorf("loading agents: %w", err)
	}

	existing, exists := cfg.Agents[name]
	if exists {
		if !yesFlag {
			fmt.Printf("Agent %q already exists. Overwrite? ", name)
			if !confirm("") {
				return fmt.Errorf("cancelled")
			}
		}
		// Merge: keep existing if new value is empty
		if agent.Global == nil && existing.Global != nil {
			agent.Global = existing.Global
		}
		if agent.Local == nil && existing.Local != nil {
			agent.Local = existing.Local
		}
	}

	cfg.Agents[name] = agent

	// Determine save scope
	scope := agents.ScopeGlobal
	if isInProject() {
		scope = agents.ScopeLocal
		fmt.Println("In project directory, saving to local config.")
	}

	if err := agents.SaveAgents(cfg, scope); err != nil {
		return fmt.Errorf("saving agents: %w", err)
	}

	fmt.Printf("✓ Agent %q added\n", name)
	if agent.Global != nil {
		fmt.Printf("    global: %s\n", agents.ContractPath(agents.ExpandPath(agent.Global.Value)))
	}
	if agent.Local != nil {
		fmt.Printf("    local:  %s\n", agents.ContractPath(agents.ExpandPath(agent.Local.Value)))
	}

	return nil
}

// isInProject checks if current directory is inside a git project.
func isInProject() bool {
	// Check for .skillforge/agents.toml in current or parent dirs
	cwd, err := os.Getwd()
	if err != nil {
		return false
	}

	for {
		localCfg := cwd + "/.skillforge/agents.toml"
		if _, err := os.Stat(localCfg); err == nil {
			return true
		}

		// Check for .git
		if _, err := os.Stat(cwd + "/.git"); err == nil {
			return true
		}

		parent := cwd + "/.."
		if parent == cwd {
			break
		}
		cwd = parent
	}

	return false
}
