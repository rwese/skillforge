package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	globalFlag  bool
	localFlag   bool
	cfgFile     string
	dryRunFlag  bool
	yesFlag     bool
	verboseFlag bool
)

// rootCmd represents the base command.
var rootCmd = &cobra.Command{
	Use:   "skillforge",
	Short: "Manage agent skills from git repositories",
	Long:  `A focused CLI for managing agent skills from git repositories.

Supports multiple agents via extensible target system with auto-detect config scope.`,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		// Configure viper
		if cfgFile != "" {
			viper.SetConfigFile(cfgFile)
		}
	},
}

// Execute runs the root command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().BoolVarP(&globalFlag, "global", "g", false, "Use global config only")
	rootCmd.PersistentFlags().BoolVarP(&localFlag, "local", "l", false, "Use local config only")
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "Config file path")
	rootCmd.PersistentFlags().BoolVarP(&dryRunFlag, "dry-run", "n", false, "Preview changes without applying")
	rootCmd.PersistentFlags().BoolVarP(&yesFlag, "yes", "y", false, "Skip confirmations and use defaults")
	rootCmd.PersistentFlags().BoolVarP(&verboseFlag, "verbose", "v", false, "Enable verbose debug output")

	// Add shell completion command
	rootCmd.AddCommand(completionCmd)
}

func initConfig() {
	// Any additional initialization
}

// getScope determines the config scope based on flags.
func getScope() string {
	if globalFlag {
		return "global"
	}
	if localFlag {
		return "local"
	}
	return "auto"
}

// verbose prints debug output when verbose flag is set.
func verbose(format string, args ...interface{}) {
	if verboseFlag {
		fmt.Printf("[DEBUG] "+format+"\n", args...)
	}
}

// confirm asks for user confirmation. Returns true if confirmed.
func confirm(prompt string) bool {
	if yesFlag {
		return true
	}

	fmt.Printf("%s [y/N] ", prompt)
	reader := bufio.NewReader(os.Stdin)
	answer, _ := reader.ReadString('\n')
	answer = strings.TrimSpace(strings.ToLower(answer))
	return answer == "y" || answer == "yes"
}

// dryRunCheck prints a message and returns an error if in dry-run mode.
func dryRunCheck(action string) bool {
	if dryRunFlag {
		fmt.Printf("[DRY-RUN] Would %s\n", action)
		return true
	}
	return false
}
