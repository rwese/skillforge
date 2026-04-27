package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	globalFlag bool
	localFlag  bool
	cfgFile    string
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
