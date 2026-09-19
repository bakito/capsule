// Copyright 2020-2026 Project Capsule Authors
// SPDX-License-Identifier: Apache-2.0

package completion

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"
)

// NewCmdCompletion returns the completion subcommand.
func NewCmdCompletion(out io.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "completion [bash|zsh|fish|powershell]",
		Short: "Generate completion script for shell",
		Long: `To load completions:

Bash:

  $ source <(kubectl-capsule completion bash)

  # To load completions for each session, execute once:
  # Linux:
  $ kubectl-capsule completion bash > /etc/bash_completion.d/kubectl-capsule
  # macOS:
  $ kubectl-capsule completion bash > $(brew --prefix)/etc/bash_completion.d/kubectl-capsule

Zsh:

  # If shell completion is not already enabled in your environment,
  # you will need to enable it.  You can execute the following once:

  $ echo "autoload -U compinit; compinit" >> ~/.zshrc

  # To load completions for each session, execute once:
  $ kubectl-capsule completion zsh > "${fpath[1]}/_kubectl-capsule"

  # You will need to start a new shell for this setup to take effect.

Fish:

  $ kubectl-capsule completion fish | source

  # To load completions for each session, execute once:
  $ kubectl-capsule completion fish > ~/.config/fish/completions/kubectl-capsule.fish

PowerShell:

  PS> kubectl-capsule completion powershell | Out-String | Invoke-Expression

  # To load completions for every new session, run:
  PS> kubectl-capsule completion powershell > kubectl-capsule.ps1
  # and source this file from your PowerShell profile.
`,
		DisableFlagsInUseLine: true,
		ValidArgs:             []string{"bash", "zsh", "fish", "powershell"},
		Args:                  cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
		RunE: func(cmd *cobra.Command, args []string) error {
			switch args[0] {
			case "bash":
				return cmd.Root().GenBashCompletion(out)
			case "zsh":
				return cmd.Root().GenZshCompletion(out)
			case "fish":
				return cmd.Root().GenFishCompletion(out, true)
			case "powershell":
				return cmd.Root().GenPowerShellCompletionWithDesc(out)
			default:
				return fmt.Errorf("unsupported shell type %q", args[0])
			}
		},
	}

	return cmd
}
