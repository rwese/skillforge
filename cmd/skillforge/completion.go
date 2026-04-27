package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var completionCmd = &cobra.Command{
	Use:   "completion [bash|zsh|fish|powershell]",
	Short: "Generate shell completion scripts",
	Long: fmt.Sprintf(`Generate shell completion scripts for skillforge.

To load completions:

Bash:

  $ source <(skillforge completion bash)

  # To load completions for each session, execute once:
  # Linux:
  $ skillforge completion bash > /etc/bash_completion.d/skillforge
  # macOS:
  $ skillforge completion bash > /usr/local/etc/bash_completion.d/skillforge

Zsh:

  # If shell completion is not already enabled in your environment,
  # you will need to enable it.  You can execute the following once:

  $ echo \"autoload -U compinit; compinit\" >> ~/.zshrc

  # To load completions for each session, execute once:
  $ skillforge completion zsh > \"${fpath[1]}/_skillforge\"

  # You will need to start a new shell for this setup to take effect.

Fish:

  $ skillforge completion fish | source

  # To load completions for each session, execute once:
  $ skillforge completion fish > ~/.config/fish/completions/skillforge.fish

PowerShell:

  $ skillforge completion powershell | Out-String | Invoke-Expression

  # To load completions for each session, execute once:
  $ skillforge completion powershell > skillforge.ps1
  # and source this file from your PowerShell profile.
`),
	DisableFlagsInUseLine: true,
	ValidArgs:             []string{"bash", "zsh", "fish", "powershell"},
	Args:                  cobra.ExactValidArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
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
