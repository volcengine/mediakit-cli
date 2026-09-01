//go:build mediakit_cloud_only

package commands

import (
	"encoding/json"

	"github.com/spf13/cobra"

	"mediakit-cli/internal/auth"
	cliconfig "mediakit-cli/internal/config"
)

func newDoctorCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "doctor",
		Short: "Inspect Cloud runtime readiness",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			home, err := cliconfig.ResolveHomeDir()
			if err != nil {
				return err
			}
			resolved, err := cliconfig.ResolveConfig(home)
			if err != nil {
				return err
			}
			authContext, authErr := auth.Resolve()
			payload := map[string]any{
				"cloud_ready": authErr == nil,
				"endpoint":    resolved.Endpoint,
				"config_file": resolved.ConfigPath,
			}
			if authErr == nil {
				payload["auth_kind"] = authContext.Kind()
			} else {
				payload["auth_error"] = authErr.Error()
			}
			encoder := json.NewEncoder(cmd.OutOrStdout())
			encoder.SetEscapeHTML(false)
			encoder.SetIndent("", "  ")
			if err := encoder.Encode(payload); err != nil {
				return err
			}
			return authErr
		},
	}
}
