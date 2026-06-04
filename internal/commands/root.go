package commands

import (
	buildinfo "mediakit-cli/internal/build"

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
		Use:               "mediakit-cli",
		Short:             "MediaKit command line interface",
		Long: `MediaKit CLI provides system commands, domain navigation, and generated capability commands.

AI Agent Skills:
  mediakit-cli pairs with AI agent skills (Claude Code, etc.) that
  teach the agent MediaKit CLI patterns, best practices, and workflows.

  Install all skills:
    npx skills add volcengine/mediakit-cli -g -y

  Or pick specific domains:
    npx skills add volcengine/mediakit-cli -s byted-mediakit-editing -y
    npx skills add volcengine/mediakit-cli -s byted-mediakit-video -y
    npx skills add volcengine/mediakit-cli -s byted-mediakit-shared -y`,
		SilenceUsage:      true,
		DisableAutoGenTag: true,
		Version:           buildinfo.Version,
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

	for _, domainCmd := range newGeneratedDomainCommands() {
		cmd.AddCommand(domainCmd)
	}

	return cmd
}
