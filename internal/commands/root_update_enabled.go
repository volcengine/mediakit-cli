//go:build !mediakit_no_update

package commands

import (
	"mediakit-cli/internal/updatecheck"

	"github.com/spf13/cobra"
)

func rootLongDescription() string {
	return `MediaKit CLI provides system commands, domain navigation, and generated capability commands.

One-click install (CLI + AI agent Skills):
  npx @volcengine/mediakit-cli install -y

Update:
  mediakit-cli update          # update CLI and AI Agent Skills via npm
  mediakit-cli update --check  # only report status
  mediakit-cli version --check # show current vs latest

AI Agent Skills:
  mediakit-cli pairs with AI agent skills (Claude Code, etc.) that
  teach the agent MediaKit CLI patterns, best practices, and workflows.

  Reinstall all skills:
    npx @volcengine/mediakit-cli install --skills-only -y`
}

func configureUpdate(cmd *cobra.Command) {
	cmd.PersistentPreRun = func(cmd *cobra.Command, args []string) {
		updatecheck.StartAsync()
	}
	cmd.PersistentPostRun = func(cmd *cobra.Command, args []string) {
		updatecheck.PrintStderrNag(cmd.ErrOrStderr())
	}
	cmd.AddCommand(newUpdateCmd())
	cmd.AddCommand(newUpdateRefreshCmd())
}
