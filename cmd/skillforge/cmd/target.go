package cmd

import (
	"fmt"
	"os"

	"github.com/rwese/skillforge/internal/config"
	"github.com/spf13/cobra"
)

// targetCmd represents the target command group.
var targetCmd = &cobra.Command{
	Use:   "target",
	Short: "Manage skill targets",
	Long: `Manage skill targets (agent skill directories).

Targets define where skills are installed. Each target has:
  - name: unique identifier
  - path: directory path for skills
  - scope: local, global, or both (where the target is usable)
  - enabled: whether the target receives skill installations`,
}

func init() {
	rootCmd.AddCommand(targetCmd)
	targetCmd.AddCommand(targetListCmd)
	targetCmd.AddCommand(targetAddCmd)
	targetCmd.AddCommand(targetRemoveCmd)
	targetCmd.AddCommand(targetEnableCmd)
	targetCmd.AddCommand(targetDisableCmd)
}

var targetListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all targets",
	RunE:  runTargetList,
}

func runTargetList(cmd *cobra.Command, args []string) error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	// Collect targets
	var targets []TargetOutput
	for name, target := range cfg.Targets {
		scopeStr := "both"
		if target.Scope == config.TargetScopeLocal {
			scopeStr = "local"
		} else if target.Scope == config.TargetScopeGlobal {
			scopeStr = "global"
		}
		targets = append(targets, TargetOutput{
			Name:    name,
			Path:    target.Path,
			Enabled: target.Enabled,
			Scope:   scopeStr,
		})
	}

	if len(targets) == 0 {
		if parseFormat(formatFlag) == formatJSON {
			return printJSON([]TargetOutput{})
		}
		fmt.Println("No targets configured.")
		PrintHint(HintNoTargets)
		return nil
	}

	fmtmt := parseFormat(formatFlag)

	if fmtmt == formatJSON {
		return printJSON(targets)
	}

	if fmtmt == formatCompact {
		fmt.Println(formatTargetCompact(targets))
		return nil
	}

	// Default: table format
	fmt.Println(formatTargetTable(targets))
	return nil
}

var formatFlag string

func init() {
	targetListCmd.Flags().StringVarP(&formatFlag, "format", "f", "text", "Output format: text, json")
}

var targetAddCmd = &cobra.Command{
	Use:   "add [name] [path]",
	Short: "Add a new target",
	Args:  cobra.ExactArgs(2),
	RunE:  runTargetAdd,
}

var enableFlag bool
var forLocalFlag bool
var forGlobalFlag bool

func init() {
	targetAddCmd.Flags().BoolVarP(&enableFlag, "enable", "e", false, "Enable target after creation")
	targetAddCmd.Flags().BoolVar(&forLocalFlag, "for-local", false, "Target is for local scope only")
	targetAddCmd.Flags().BoolVar(&forGlobalFlag, "for-global", false, "Target is for global scope only")
}

func runTargetAdd(cmd *cobra.Command, args []string) error {
	name := args[0]
	path := config.ExpandPath(args[1])

	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	if _, exists := cfg.Targets[name]; exists {
		PrintHint(HintTargetExists)
		return fmt.Errorf("target %q already exists", name)
	}

	// Determine target scope
	if forLocalFlag && forGlobalFlag {
		return fmt.Errorf("cannot specify both --for-local and --for-global")
	}
	if !forLocalFlag && !forGlobalFlag {
		return fmt.Errorf("must specify either --for-local or --for-global")
	}

	targetScope := config.TargetScopeLocal
	if forGlobalFlag {
		targetScope = config.TargetScopeGlobal
	}

	cfg.Targets[name] = config.Target{
		Path:    path,
		Enabled: enableFlag,
		Scope:   targetScope,
	}

	if err := saveConfig(cfg); err != nil {
		return err
	}

	if parseFormat(formatFlag) == formatJSON {
		return printJSON(TargetOutput{
			Name:    name,
			Path:    path,
			Enabled: enableFlag,
			Scope:   targetScope.String(),
		})
	}

	fmt.Printf("✓ Added target %s (scope: %s)\n", name, targetScope.String())
	return nil
}

var targetRemoveCmd = &cobra.Command{
	Use:   "remove [name]",
	Short: "Remove a target",
	Args:  cobra.ExactArgs(1),
	RunE:  runTargetRemove,
}

func runTargetRemove(cmd *cobra.Command, args []string) error {
	name := args[0]

	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	target, exists := cfg.Targets[name]
	if !exists {
		PrintHint(HintTargetNotFound)
		return fmt.Errorf("target %q not found", name)
	}

	// Confirm if not using --yes
	if !yesFlag {
		fmt.Printf("Remove target %q (%s)? ", name, config.ContractPath(target.Path))
		if !confirm("") {
			return fmt.Errorf("cancelled")
		}
	}

	delete(cfg.Targets, name)
	if err := saveConfig(cfg); err != nil {
		return err
	}

	if parseFormat(formatFlag) == formatJSON {
		return printJSON(map[string]string{"removed": name})
	}

	fmt.Printf("✓ Removed target %s\n", name)
	return nil
}

var targetEnableCmd = &cobra.Command{
	Use:   "enable [name]",
	Short: "Enable a target",
	Args:  cobra.ExactArgs(1),
	RunE:  runTargetEnable,
}

func runTargetEnable(cmd *cobra.Command, args []string) error {
	name := args[0]
	scope := parseScope(scopeFlag)

	// Load config based on scope
	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	// Check if target exists in loaded config
	target, exists := cfg.Targets[name]

	// If enabling locally and target not in local config, check global
	if scope == config.ScopeLocal && !exists {
		// Load global config to find the target
		globalLoader := config.NewLoader(config.ScopeGlobal)
		globalCfg, err := globalLoader.Load()
		if err == nil {
			if globalTarget, globalExists := globalCfg.Targets[name]; globalExists {
				target = globalTarget
				exists = true
				// Will create local entry with global path and local enabled status
				fmt.Printf("→ Creating local override for target %q (defined globally)\n", name)
			}
		}
	}

	if !exists {
		PrintHint(HintTargetNotFound)
		return fmt.Errorf("target %q not found", name)
	}

	target.Enabled = true
	cfg.Targets[name] = target

	if err := saveConfig(cfg); err != nil {
		return err
	}

	if parseFormat(formatFlag) == formatJSON {
		return printJSON(TargetOutput{
			Name:    name,
			Path:    target.Path,
			Enabled: true,
		})
	}

	fmt.Printf("✓ Enabled target %s\n", name)
	return nil
}

var targetDisableCmd = &cobra.Command{
	Use:   "disable [name]",
	Short: "Disable a target",
	Args:  cobra.ExactArgs(1),
	RunE:  runTargetDisable,
}

func runTargetDisable(cmd *cobra.Command, args []string) error {
	name := args[0]
	scope := parseScope(scopeFlag)

	// Load config based on scope
	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	// Check if target exists in loaded config
	target, exists := cfg.Targets[name]

	// If disabling locally and target not in local config, check global
	if scope == config.ScopeLocal && !exists {
		// Load global config to find the target
		globalLoader := config.NewLoader(config.ScopeGlobal)
		globalCfg, err := globalLoader.Load()
		if err == nil {
			if globalTarget, globalExists := globalCfg.Targets[name]; globalExists {
				target = globalTarget
				exists = true
				// Will create local entry with global path and local enabled status
				fmt.Printf("→ Creating local override for target %q (defined globally)\n", name)
			}
		}
	}

	if !exists {
		PrintHint(HintTargetNotFound)
		return fmt.Errorf("target %q not found", name)
	}

	target.Enabled = false
	cfg.Targets[name] = target

	if err := saveConfig(cfg); err != nil {
		return err
	}

	if parseFormat(formatFlag) == formatJSON {
		return printJSON(TargetOutput{
			Name:    name,
			Path:    target.Path,
			Enabled: false,
		})
	}

	fmt.Printf("✓ Disabled target %s\n", name)
	return nil
}

func loadConfig() (*config.Config, error) {
	scope := parseScope(scopeFlag)

	loader := config.NewLoader(scope)

	// Verbose output: show config loading details
	verbose("Loading config with scope: %s", scope)
	verbose("Global config path: %s", loader.GlobalPath())
	if scope != config.ScopeGlobal {
		localPath := config.DetectLocalPath()
		if localPath != "" {
			verbose("Local config path: %s", localPath)
		} else {
			verbose("No local config found")
		}
	}

	return loader.Load()
}

// parseScope converts scope string to config.Scope.
func parseScope(s string) config.Scope {
	if s == "global" {
		return config.ScopeGlobal
	}
	return config.ScopeLocal
}

func saveConfig(cfg *config.Config) error {
	scope := parseScope(scopeFlag)
	local := scope == config.ScopeLocal

	loader := config.NewLoader(scope)
	return loader.Save(cfg, local)
}

func ensureTargetDir(path string) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return os.MkdirAll(path, 0755)
	}
	return nil
}
