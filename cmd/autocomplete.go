package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(completionCmd)
}

var completionCmd = &cobra.Command{
	Hidden:                true,
	DisableFlagsInUseLine: true,
	Use:                   "completion [bash|zsh|fish|powershell]",
	Args:                  cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
	ValidArgs:             []string{"bash", "zsh", "fish", "powershell"},
	Short:                 "Output shell completion code for bash or zsh. Defaults to bash",
	Long: `To load completion:

Bash:

  $ source <(minepkg completion bash)

  # To load completions for each session, execute once:
  # Linux:
  minepkg completion bash > /etc/bash_completion.d/minepkg
  # macOS:
  minepkg completion bash > /usr/local/etc/bash_completion.d/minepkg

Oh-my-zsh:
  mkdir -p ~/.oh-my-zsh/custom/plugins/minepkg
  minepkg completion zsh > ~/.oh-my-zsh/custom/plugins/minepkg/minepkg.plugin.zsh
  echo "compdef _minepkg minepkg" >> ~/.oh-my-zsh/custom/plugins/minepkg/minepkg.plugin.zsh

  # Then add minepkg to your plugins in your .zshrc

Zsh:

  # If shell completion is not already enabled in your environment,
  # you will need to enable it. To to that run the following once:

  echo "autoload -U compinit; compinit" >> ~/.zshrc

  # Add this to your .zshrc file:
  source <(minepkg completion zsh)
  # You will need to start a new shell for this setup to take effect.

fish:

  minepkg completion fish | source

  # To load completions for each session, execute once:
  minepkg completion fish > ~/.config/fish/completions/minepkg.fish

PowerShell:

  PS> minepkg completion powershell | Out-String | Invoke-Expression

  # To load completions for every new session, run:
  PS> minepkg completion powershell > minepkg.ps1
  # and source this file from your PowerShell profile.
`,
	Run: func(cmd *cobra.Command, args []string) {
		switch args[0] {
		case "bash":
			cmd.Root().GenBashCompletion(os.Stdout)
		case "zsh":
			// make this loadable
			os.Stdout.WriteString("#compdef minepkg\ncompdef _minepkg minepkg\n")
			cmd.Root().GenZshCompletion(os.Stdout)

		case "fish":
			cmd.Root().GenFishCompletion(os.Stdout, true)
		case "powershell":
			cmd.Root().GenPowerShellCompletionWithDesc(os.Stdout)
		}
	},
}
