package commands

import (
	"os"

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
	if err := validateModeArguments(os.Args[1:]); err != nil {
		return err
	}
	return newRootCmd().Execute()
}

func newRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:               "mediakit-cli",
		Short:             "MediaKit command line interface",
		Long:              rootLongDescription(),
		SilenceUsage:      true,
		SilenceErrors:     true,
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

	strictBoolVar(cmd.Flags(), &showDomains, "domains", false, "List all domains")
	strictBoolVar(cmd.Flags(), &showHelpFull, "help-full", false, "Show the full capability index")
	configureModeFlags(cmd)
	configureUpdate(cmd)

	cmd.AddCommand(newInitCmd())
	cmd.AddCommand(newDoctorCmd())
	cmd.AddCommand(newConfigCmd())
	cmd.AddCommand(newVersionCmd())

	for _, domainCmd := range newGeneratedDomainCommands() {
		cmd.AddCommand(domainCmd)
	}

	return cmd
}
