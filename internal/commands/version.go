package commands

import (
	"encoding/json"
	"fmt"
	"time"

	buildinfo "mediakit-cli/internal/build"
	"mediakit-cli/internal/updatecheck"

	"github.com/spf13/cobra"
)

func newVersionCmd() *cobra.Command {
	var checkUpdate bool
	cmd := &cobra.Command{
		Use:               "version",
		Short:             "Print CLI version information",
		Args:              cobra.NoArgs,
		DisableAutoGenTag: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if checkUpdate {
				r := updatecheck.CheckNow(2 * time.Second)
				payload := map[string]any{
					"current": buildinfo.Version,
					"date":    buildinfo.Date,
				}
				if r != nil {
					payload["latest"] = r.Latest
					payload["has_update"] = r.HasUpdate
					if r.Err != nil {
						payload["error"] = r.Err.Error()
					}
					if r.HasUpdate {
						payload["upgrade_command"] = "mediakit-cli update"
					}
				}
				encoder := json.NewEncoder(cmd.OutOrStdout())
				encoder.SetEscapeHTML(false)
				encoder.SetIndent("", "  ")
				return encoder.Encode(payload)
			}
			_, err := fmt.Fprintf(
				cmd.OutOrStdout(),
				"mediakit-cli %s\nbuild date: %s\n",
				buildinfo.Version,
				buildinfo.Date,
			)
			return err
		},
	}
	cmd.Flags().BoolVar(&checkUpdate, "check", false, "Check the npm registry for a newer release and report status")
	return cmd
}
