package commands

import (
	buildinfo "mediakit-cli/internal/build"
	"mediakit-cli/internal/updatecheck"

	"github.com/spf13/cobra"
)

var (
	showDomains  bool
	showHelpFull bool
	forceLocal   bool
	forceCloud   bool
)

func Execute() error {
	return newRootCmd().Execute()
}

func newRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "mediakit-cli",
		Short: "MediaKit command line interface",
		Long: `MediaKit CLI provides system commands, domain navigation, and generated capability commands.

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
    npx @volcengine/mediakit-cli install --skills-only -y`,
		SilenceUsage:      true,
		SilenceErrors:     true,
		DisableAutoGenTag: true,
		Version:           buildinfo.Version,
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			updatecheck.StartAsync()
		},
		PersistentPostRun: func(cmd *cobra.Command, args []string) {
			updatecheck.PrintStderrNag(cmd.ErrOrStderr())
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			switch {
			case showDomains:
				return printDomains(cmd)
			case showHelpFull:
				return printHelpFull(cmd)
			default:
				return cmd.Help()
			}
		},
	}

	cmd.Flags().BoolVar(&showDomains, "domains", false, "List all domains")
	cmd.Flags().BoolVar(&showHelpFull, "help-full", false, "Show the full capability index")
	cmd.PersistentFlags().BoolVar(&forceLocal, "local", false, "仅对当前命令生效：强制按 local-first 策略执行，不修改全局配置")
	cmd.PersistentFlags().BoolVar(&forceCloud, "cloud", false, "仅对当前命令生效：强制按 cloud-first 策略执行，不修改全局配置")

	cmd.AddCommand(newInitCmd())
	cmd.AddCommand(newDoctorCmd())
	cmd.AddCommand(newConfigCmd())
	cmd.AddCommand(newVersionCmd())
	cmd.AddCommand(newUpdateCmd())
	cmd.AddCommand(newUpdateRefreshCmd())

	for _, domainCmd := range newGeneratedDomainCommands() {
		cmd.AddCommand(domainCmd)
	}

	return cmd
}
