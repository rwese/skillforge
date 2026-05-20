package cmd

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"

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
	enableFlag  bool
)

// rootLong returns the root command's long description with $0 replaced.
func rootLong() string {
	template := `A focused CLI for managing agent skills from git repositories.

Supports multiple agents via extensible target system with auto-detect config scope.

═══════════════════════════════════════════════════════════════════════════════
 QUICK START
═══════════════════════════════════════════════════════════════════════════════

  $0 setup                        # Interactive setup wizard
  $0 repo add https://github.com/user/skills
  $0 skill install <name>          # Install a skill
  $0 sync                          # Sync all repos and update skills

═══════════════════════════════════════════════════════════════════════════════
 COMMANDS
═══════════════════════════════════════════════════════════════════════════════

  repo    Manage cached skill repositories (add, list, update, remove)
  skill   Manage installed skills (install, list, search, update, remove)
  sync    Sync repositories + update skills + sync across agents
  target  Manage skill target directories for agents
  setup   Interactive wizard to configure skill directories

═══════════════════════════════════════════════════════════════════════════════
 COMMON WORKFLOWS
═══════════════════════════════════════════════════════════════════════════════

  # Add a skill repository
  $0 repo add https://github.com/user/agent-skills

  # Search for available skills
  $0 skill search git

  # Install a skill to all agents
  $0 skill install my-git-fu

  # Install a skill to a specific agent only
  $0 skill install my-git-fu --agent pi

  # Check and fix global target sync
  $0 sync            # Preview what would change
  $0 sync --fix      # Apply updates

  # Check installed skills
  $0 skill list
  $0 skill list --agent pi

═══════════════════════════════════════════════════════════════════════════════
 CONFIGURATION
═══════════════════════════════════════════════════════════════════════════════

  Config scopes (--scope flag):
    global  ~/.config/skillforge/config.toml  (shared across system)
    local   ./.skillforge.toml                (project-specific, default)

  Global flags:
    -s, --scope     Config scope (default: local)
    -n, --dry-run   Preview changes without applying
    -y, --yes       Skip confirmations
    -v, --verbose   Debug output

═══════════════════════════════════════════════════════════════════════════════
 EXAMPLES
═══════════════════════════════════════════════════════════════════════════════

  $0 repo add https://github.com/rwese/skillforge --alias skillforge
  $0 repo list --format json
  $0 skill search "docker" --format json
  $0 skill install git-fu --agent pi
  $0 skill remove git-fu --agent pi
  $0 sync --agent pi --fix
  $0 target list

═══════════════════════════════════════════════════════════════════════════════

Run "$0 [command] --help" for more details on a specific command.`
	return strings.ReplaceAll(template, "$0", cmdName())
}

// rootCmd represents the base command.
var rootCmd = &cobra.Command{
	Use:   "skillforge",
	Short: "Manage agent skills from git repositories",
	Long:  rootLong(),
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		// Configure viper
		if cfgFile != "" {
			viper.SetConfigFile(cfgFile)
		}
	},
}

// Execute runs the root command.
func Execute() {
	// Set the executable name based on os.Args[0] (actual binary name)
	// Strip path to get just the binary name
	binaryName := filepath.Base(os.Args[0])
	SetExecName(binaryName)

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	// Set help handler for completion command to refresh $0
	completionCmd.SetHelpFunc(func(cmd *cobra.Command, args []string) {
		cmd.Long = completionLong()
		// Render the command's own help
		fmt.Fprintln(os.Stdout, cmd.Long)
		if cmd.HasAvailableFlags() {
			fmt.Fprintln(os.Stdout, "\nFlags:")
			cmd.Flags().PrintDefaults()
		}
	})

	// Create custom template with functions
	funcMap := template.FuncMap{
		"rpad": func(s string, width int) string {
			if len(s) < width {
				return s + strings.Repeat(" ", width-len(s))
			}
			return s
		},
		"cleanUsage":     cleanUsage,
		"removeHelpFlag": removeHelpFlag,
	}
	tmpl, err := template.New("help").Funcs(funcMap).Parse(helpTemplate)
	if err != nil {
		panic(err)
	}
	rootCmd.SetHelpTemplate(helpTemplate)
	rootCmd.SetHelpFunc(func(cmd *cobra.Command, args []string) {
		// Refresh the Long description with current command name
		rootCmd.Long = rootLong()

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
	rootCmd.PersistentFlags().StringVarP(&scopeFlag, "scope", "s", "", "Config scope (global|local|all)")
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "Config file path (default: ~/.config/skillforge/config.toml)")
	rootCmd.PersistentFlags().BoolVarP(&dryRunFlag, "dry-run", "n", false, "Preview changes without applying")
	rootCmd.PersistentFlags().BoolVarP(&yesFlag, "yes", "y", false, "Skip confirmations and use defaults")
	rootCmd.PersistentFlags().BoolVarP(&verboseFlag, "verbose", "v", false, "Enable verbose debug output")
	_ = rootCmd.RegisterFlagCompletionFunc("scope", completeScopes)

	// Add shell completion command
	rootCmd.AddCommand(completionCmd)
}

func initConfig() {
	// Any additional initialization
}

// getScope returns the effective scope based on --scope flag.
func getScope() string {
	return scopeFlag
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
