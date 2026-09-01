//go:build mediakit_cloud_only

package commands

import (
	"fmt"

	"github.com/spf13/cobra"

	"mediakit-cli/internal/auth"
	cliconfig "mediakit-cli/internal/config"
)

func newConfigCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Manage Cloud configuration",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Help()
		},
	}
	cmd.AddCommand(newCloudConfigSetCmd())
	cmd.AddCommand(newCloudConfigShowCmd())
	return cmd
}

func newCloudConfigSetCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "set",
		Short: "Set non-secret Cloud configuration values",
	}
	for _, item := range []struct {
		name  string
		short string
		apply func(*cliconfig.Config, string)
	}{
		{
			name:  "endpoint",
			short: "Set the Cloud API endpoint",
			apply: func(cfg *cliconfig.Config, value string) { cfg.Endpoint = value },
		},
		{
			name:  "runtime",
			short: "Set the runtime label",
			apply: func(cfg *cliconfig.Config, value string) { cfg.Runtime = value },
		},
	} {
		entry := item
		cmd.AddCommand(&cobra.Command{
			Use:   entry.name + " [value]",
			Short: entry.short,
			Args:  cobra.ExactArgs(1),
			RunE: func(cmd *cobra.Command, args []string) error {
				home, err := cliconfig.ResolveHomeDir()
				if err != nil {
					return err
				}
				cfg, err := cliconfig.LoadConfig(home)
				if err != nil {
					return err
				}
				entry.apply(&cfg, args[0])
				if err := cliconfig.SaveConfig(home, cfg); err != nil {
					return err
				}
				_, err = fmt.Fprintf(
					cmd.OutOrStdout(),
					"%s 已更新\n配置文件：%s\n",
					entry.name,
					cliconfig.ConfigFile(home),
				)
				return err
			},
		})
	}
	return cmd
}

func newCloudConfigShowCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "show",
		Short: "Show current Cloud configuration",
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
			authKind := "not_configured"
			if authErr == nil {
				authKind = string(authContext.Kind())
			}
			_, err = fmt.Fprintf(
				cmd.OutOrStdout(),
				"当前 Cloud 配置\n- auth: %s\n- endpoint: %s (%s)\n- runtime: %s\n- config_file: %s\n- cache_directory: %s\n",
				authKind,
				resolved.Endpoint,
				resolved.EndpointSource,
				resolved.Runtime,
				resolved.ConfigPath,
				resolved.CacheDir,
			)
			return err
		},
	}
}
