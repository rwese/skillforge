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
	targetCmd.AddCommand(targetGlobalCmd)
}

var targetGlobalCmd = &cobra.Command{
	Use:   "global",
	Short: "Manage named global directories for a target",
}

func init() {
	targetGlobalCmd.AddCommand(targetGlobalAddCmd)
	targetGlobalCmd.AddCommand(targetGlobalRemoveCmd)
}

func targetToOutput(name string, target config.Target) TargetOutput {
	return TargetOutput{
		Name:        name,
		GlobalPath:  target.GlobalPath,
		GlobalPaths: target.GlobalPaths,
		LocalPath:   target.LocalPath,
		Enabled:     target.Enabled,
	}
}

var targetListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all targets",
	RunE:  runTargetList,
}

func runTargetList(cmd *cobra.Command, args []string) error {
	// When scope is empty, show targets from both local and global configs
	var targets []TargetOutput

	// Load global targets
	globalCfg, err := loadConfigScope(config.ScopeGlobal)
	if err != nil {
		return err
	}
	for name, target := range globalCfg.Targets {
		targets = append(targets, TargetOutput{
			Name:        name,
			GlobalPath:  target.GlobalPath,
			GlobalPaths: target.GlobalPaths,
			LocalPath:   target.LocalPath,
			Enabled:     target.Enabled,
		})
	}

	// Load local targets (only if local config exists)
	localCfg, err := loadConfigScope(config.ScopeLocal)
	if err == nil {
		for name, target := range localCfg.Targets {
			// Skip if already in global (local takes precedence for same name)
			if _, exists := globalCfg.Targets[name]; exists {
				continue
			}
			targets = append(targets, TargetOutput{
				Name:        name,
				GlobalPath:  target.GlobalPath,
				GlobalPaths: target.GlobalPaths,
				LocalPath:   target.LocalPath,
				Enabled:     target.Enabled,
			})
		}
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
	_ = targetListCmd.RegisterFlagCompletionFunc("format", completeFormats)
}

var targetAddCmd = &cobra.Command{
	Use:   "add [name] [globalPath] [localPath]",
	Short: "Add a new target",
	Args:  cobra.ExactArgs(3),
	RunE:  runTargetAdd,
}

func init() {
	targetAddCmd.Flags().BoolVarP(&enableFlag, "enable", "e", false, "Enable target after creation")
}

func runTargetAdd(cmd *cobra.Command, args []string) error {
	name := args[0]
	globalPath := config.ExpandPath(args[1])
	localPath := config.ExpandPath(args[2])

	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	if _, exists := cfg.Targets[name]; exists {
		PrintHint(HintTargetExists)
		return fmt.Errorf("target %q already exists", name)
	}

	cfg.Targets[name] = config.Target{
		GlobalPath: globalPath,
		LocalPath:  localPath,
		Enabled:    enableFlag,
	}

	if err := saveConfig(cfg); err != nil {
		return err
	}

	if parseFormat(formatFlag) == formatJSON {
		return printJSON(targetToOutput(name, cfg.Targets[name]))
	}

	fmt.Printf("✓ Added target %s\n", name)
	return nil
}

var targetRemoveCmd = &cobra.Command{
	Use:               "remove [name]",
	Short:             "Remove a target",
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: completeTargets,
	RunE:              runTargetRemove,
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
		fmt.Printf("Remove target %q (%s)? ", name, config.ContractPath(target.GlobalPath))
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
	Use:               "enable [name]",
	Short:             "Enable a target",
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: completeTargets,
	RunE:              runTargetEnable,
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
		return printJSON(targetToOutput(name, target))
	}

	fmt.Printf("✓ Enabled target %s\n", name)
	return nil
}

var targetDisableCmd = &cobra.Command{
	Use:               "disable [name]",
	Short:             "Disable a target",
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: completeTargets,
	RunE:              runTargetDisable,
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
		return printJSON(targetToOutput(name, target))
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

var targetGlobalAddCmd = &cobra.Command{
	Use:               "add [target] [name] [path]",
	Short:             "Add a named global directory to a target",
	Args:              cobra.ExactArgs(3),
	ValidArgsFunction: completeTargets,
	RunE:              runTargetGlobalAdd,
}

func runTargetGlobalAdd(cmd *cobra.Command, args []string) error {
	targetName := args[0]
	globalName := args[1]
	globalPath := config.ExpandPath(args[2])

	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	target, exists := cfg.Targets[targetName]
	if !exists {
		PrintHint(HintTargetNotFound)
		return fmt.Errorf("target %q not found", targetName)
	}

	if target.GlobalPaths == nil {
		target.GlobalPaths = make(map[string]string)
	}
	if _, exists := target.GlobalPaths[globalName]; exists {
		return fmt.Errorf("global directory %q already exists for target %q", globalName, targetName)
	}

	target.GlobalPaths[globalName] = globalPath
	cfg.Targets[targetName] = target

	if err := saveConfig(cfg); err != nil {
		return err
	}

	if parseFormat(formatFlag) == formatJSON {
		return printJSON(targetToOutput(targetName, target))
	}

	fmt.Printf("✓ Added global directory %s to target %s\n", globalName, targetName)
	return nil
}

var targetGlobalRemoveCmd = &cobra.Command{
	Use:               "remove [target] [name]",
	Short:             "Remove a named global directory from a target",
	Args:              cobra.ExactArgs(2),
	ValidArgsFunction: completeTargets,
	RunE:              runTargetGlobalRemove,
}

func runTargetGlobalRemove(cmd *cobra.Command, args []string) error {
	targetName := args[0]
	globalName := args[1]

	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	target, exists := cfg.Targets[targetName]
	if !exists {
		PrintHint(HintTargetNotFound)
		return fmt.Errorf("target %q not found", targetName)
	}
	if _, exists := target.GlobalPaths[globalName]; !exists {
		return fmt.Errorf("global directory %q not found for target %q", globalName, targetName)
	}
	if len(target.GlobalPaths) == 1 && target.GlobalPath == "" {
		return fmt.Errorf("cannot remove last global directory from target %q", targetName)
	}

	delete(target.GlobalPaths, globalName)
	if len(target.GlobalPaths) == 0 {
		target.GlobalPaths = nil
	}
	cfg.Targets[targetName] = target

	if err := saveConfig(cfg); err != nil {
		return err
	}

	if parseFormat(formatFlag) == formatJSON {
		return printJSON(targetToOutput(targetName, target))
	}

	fmt.Printf("✓ Removed global directory %s from target %s\n", globalName, targetName)
	return nil
}
