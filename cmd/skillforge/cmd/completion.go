package cmd

import (
	"os"
	"strings"

	"github.com/spf13/cobra"
)

var completionCmd = &cobra.Command{
	Use:   "completion [bash|zsh|fish|powershell]",
	Short: "Generate shell completion scripts",
	Long:  "", // Set dynamically
	DisableFlagsInUseLine: true,
	ValidArgs:             []string{"bash", "zsh", "fish", "powershell"},
	Args:                  cobra.ExactValidArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		// Refresh Long with current command name
		cmd.Long = completionLong()

		switch args[0] {
		case "bash":
			return rootCmd.GenBashCompletion(os.Stdout)
		case "zsh":
			return rootCmd.GenZshCompletion(os.Stdout)
		case "fish":
			return rootCmd.GenFishCompletion(os.Stdout, true)
		case "powershell":
			return rootCmd.GenPowerShellCompletionWithDesc(os.Stdout)
		}
		return nil
	},
}

func completionLong() string {
	template := `Generate shell completion scripts.

To load completions:

Bash:

  $ source <($0 completion bash)

  # To load completions for each session, execute once:
  # Linux:
  $ $0 completion bash > /etc/bash_completion.d/$0
  # macOS:
  $ $0 completion bash > /usr/local/etc/bash_completion.d/$0

Zsh:

  # If shell completion is not already enabled in your environment,
  # you will need to enable it.  You can execute the following once:

  $ echo "autoload -U compinit; compinit" >> ~/.zshrc

  # To load completions for each session, execute once:
  $ $0 completion zsh > "${fpath[1]}/_$0"

  # You will need to start a new shell for this setup to take effect.

Fish:

  $ $0 completion fish | source

  # To load completions for each session, execute once:
  $ $0 completion fish > ~/.config/fish/completions/$0.fish

PowerShell:

  $ $0 completion powershell | Out-String | Invoke-Expression

  # To load completions for each session, execute once:
  $ $0 completion powershell > $0.ps1
  # and source this file from your PowerShell profile.
`
	return strings.ReplaceAll(template, "$0", cmdName())
}
