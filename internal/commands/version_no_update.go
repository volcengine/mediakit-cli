//go:build mediakit_no_update

package commands

import (
	"fmt"

	buildinfo "mediakit-cli/internal/build"

	"github.com/spf13/cobra"
)

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:               "version",
		Short:             "Print CLI version information",
		Args:              cobra.NoArgs,
		DisableAutoGenTag: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			_, err := fmt.Fprintf(
				cmd.OutOrStdout(),
				"mediakit-cli %s\nbuild date: %s\n",
				buildinfo.Version,
				buildinfo.Date,
			)
			return err
		},
	}
}
