package main

import (
	"bytes"
	"fmt"
	"os"
	"strings"
	"text/template"

	"github.com/rwese/skillforge-ng/internal/config"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// removeHelpFlag removes help flag from flag usage string.
func removeHelpFlag(s string) string {
	// Pattern: newline + spaces + "-h, --help   help for <cmdname>"
	lines := strings.Split(s, "\n")
	var filtered []string
	for _, line := range lines {
		if strings.Contains(line, "-h, --help") {
			continue
		}
		filtered = append(filtered, line)
	}
	return strings.Join(filtered, "\n")
}

// cleanUsage strips [flags] from cobra's generated usage line.
func cleanUsage(cmd *cobra.Command) string {
	usage := cmd.UseLine()
	usage = strings.ReplaceAll(usage, " [flags]", "")
	return usage
}

// helpTemplate customizes cobra's help output.
// - Shows global flags in root help
// - Shows "Global Flags:" section in subcommands
// - Only shows "Flags:" for commands with local flags
var helpTemplate = `{{if .Short}}{{.Short}}

{{end}}{{if .Long}}{{.Long}}

{{end}}Usage:
  {{cleanUsage .}}

{{if .HasAvailableSubCommands}}Available Commands:
{{range .Commands}}{{if .IsAvailableCommand}}
  {{rpad .Name .NamePadding}} {{.Short}}{{end}}{{end}}

{{end}}{{if .HasExample}}Examples:
{{.Example}}

{{end}}{{if .HasAvailableLocalFlags}}
Flags:
{{.LocalFlags.FlagUsages}}
{{end}}{{if .HasAvailableInheritedFlags}}
Global Flags:
{{.InheritedFlags.FlagUsages}}{{end}}{{if .HasHelpSubCommands}}

Additional help topics:{{range .HelpCommands}}{{if .IsAvailableCommand}}
  {{rpad .Name .NamePadding}} {{.Short}}{{end}}{{end}}{{end}}

{{if .HasAvailableSubCommands}}Use "{{.CommandPath}} [command] --help" for more information about a command.{{end}}`

var (
	scopeFlag   string
	cfgFile     string
	dryRunFlag  bool
	yesFlag     bool
	verboseFlag bool
)

// rootCmd represents the base command.
var rootCmd = &cobra.Command{
	Use:   "skillforge",
	Short: "Manage agent skills from git repositories",
	Long: `A focused CLI for managing agent skills from git repositories.

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

	// Create custom template with functions
	funcMap := template.FuncMap{
		"rpad": func(s string, width int) string {
			if len(s) < width {
				return s + strings.Repeat(" ", width-len(s))
			}
			return s
		},
		"cleanUsage": cleanUsage,
		"removeHelpFlag": removeHelpFlag,
	}
	tmpl, err := template.New("help").Funcs(funcMap).Parse(helpTemplate)
	if err != nil {
		panic(err)
	}
	rootCmd.SetHelpTemplate(helpTemplate)
	rootCmd.SetHelpFunc(func(cmd *cobra.Command, args []string) {
		buf := &bytes.Buffer{}
		if err := tmpl.Execute(buf, cmd); err != nil {
			cmd.Help()
			return
		}
		// Remove [flags] from output
		output := strings.ReplaceAll(buf.String(), " [flags]", "")
		// Remove help flag from Flags section (keep it in Global Flags)
		lines := strings.Split(output, "\n")
		var filtered []string
		inFlags := false
		for _, line := range lines {
			if strings.HasPrefix(line, "Flags:") {
				inFlags = true
				filtered = append(filtered, line)
				continue
			}
			if strings.HasPrefix(line, "Global Flags:") {
				inFlags = false
				filtered = append(filtered, line)
				continue
			}
			if inFlags && strings.Contains(line, "-h, --help") {
				continue
			}
			filtered = append(filtered, line)
		}
		fmt.Print(strings.Join(filtered, "\n"))
	})

	// Global flags - available to all commands
	rootCmd.PersistentFlags().StringVarP(&scopeFlag, "scope", "s", "auto", "Config scope (global|local|auto)")
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "Config file path (default: ~/.config/skillforge/config.toml)")
	rootCmd.PersistentFlags().BoolVarP(&dryRunFlag, "dry-run", "n", false, "Preview changes without applying")
	rootCmd.PersistentFlags().BoolVarP(&yesFlag, "yes", "y", false, "Skip confirmations and use defaults")
	rootCmd.PersistentFlags().BoolVarP(&verboseFlag, "verbose", "v", false, "Enable verbose debug output")

	// Add shell completion command
	rootCmd.AddCommand(completionCmd)

	// Update scope help text with detected value
	updateScopeHelp()
}

func initConfig() {
	// Any additional initialization
}

// getScope returns the effective scope based on --scope flag.
func getScope() string {
	return scopeFlag
}

// detectScope determines the auto-detected scope (for help display).
func detectScope() string {
	if scopeFlag != "auto" {
		return scopeFlag
	}
	if config.DetectLocalPath() != "" {
		return "local"
	}
	return "global"
}

// updateScopeHelp updates the --scope flag usage with detected value.
func updateScopeHelp() {
	flag := rootCmd.PersistentFlags().Lookup("scope")
	if flag != nil {
		detected := detectScope()
		// Only show detected if it differs from the flag value
		if detected != scopeFlag {
			flag.Usage = fmt.Sprintf("Config scope (global|local|auto), detected: %s", detected)
		} else {
			flag.Usage = "Config scope (global|local|auto)"
		}
	}
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

	// Read single character from stdin (works with raw mode)
	var buf [1]byte
	os.Stdin.Read(buf[:])
	answer := strings.ToLower(string(buf[:]))
	fmt.Println() // newline after input

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
