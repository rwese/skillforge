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

Targets define where skills are installed. Each target has a path and enabled status.`,
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
		targets = append(targets, TargetOutput{
			Name:    name,
			Path:    target.Path,
			Enabled: target.Enabled,
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

func init() {
	targetAddCmd.Flags().BoolVarP(&enableFlag, "enable", "e", false, "Enable target after creation")
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

	cfg.Targets[name] = config.Target{
		Path:    path,
		Enabled: enableFlag,
	}

	if err := saveConfig(cfg); err != nil {
		return err
	}

	if parseFormat(formatFlag) == formatJSON {
		return printJSON(TargetOutput{
			Name:    name,
			Path:    path,
			Enabled: enableFlag,
		})
	}

	fmt.Printf("✓ Added target %s\n", name)
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

	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	target, exists := cfg.Targets[name]
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

	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	target, exists := cfg.Targets[name]
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
	switch s {
	case "global":
		return config.ScopeGlobal
	case "local":
		return config.ScopeLocal
	default:
		return config.ScopeAuto
	}
}

func saveConfig(cfg *config.Config) error {
	scope := parseScope(scopeFlag)
	local := scope == config.ScopeLocal || (scope == config.ScopeAuto && config.DetectLocalPath() != "")

	loader := config.NewLoader(scope)
	return loader.Save(cfg, local)
}

func ensureTargetDir(path string) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return os.MkdirAll(path, 0755)
	}
	return nil
}
