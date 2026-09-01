//go:build mediakit_cloud_only

package commands

import (
	"fmt"

	"github.com/spf13/cobra"

	"mediakit-cli/internal/auth"
	"mediakit-cli/internal/buildenv"
	cliconfig "mediakit-cli/internal/config"
)

func newInitCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize non-secret Cloud configuration",
		Long: "Initialize the Cloud endpoint and runtime label. " +
			"Authentication values are accepted only from runtime environment variables.",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			home, err := cliconfig.ResolveHomeDir()
			if err != nil {
				return err
			}
			cfg, err := cliconfig.LoadConfig(home)
			if err != nil {
				return err
			}
			if value, _ := cmd.Flags().GetString("endpoint"); value != "" {
				cfg.Endpoint = value
			}
			if value, _ := cmd.Flags().GetString("runtime"); value != "" {
				cfg.Runtime = value
			}
			if err := cliconfig.SaveConfig(home, cfg); err != nil {
				return err
			}
			authContext, authErr := auth.Resolve()
			if authErr != nil {
				return authErr
			}
			_, err = fmt.Fprintf(
				cmd.OutOrStdout(),
				"Cloud 配置完成\n- auth: %s\n- API Key 环境变量: %s\n- LogID 环境变量: %s\n- 配置文件: %s\n",
				authContext.Kind(),
				buildenv.CloudAPIKey,
				buildenv.CloudLogID,
				cliconfig.ConfigFile(home),
			)
			return err
		},
	}
	cmd.Flags().String("endpoint", "", "Custom Cloud API endpoint")
	cmd.Flags().String("runtime", "", "Runtime label")
	return cmd
}
