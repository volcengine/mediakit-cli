package commands

import (
	"fmt"

	"github.com/spf13/cobra"

	cliconfig "mediakit-cli/internal/config"
)

func newConfigCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Manage mediakit-cli configuration",
	}

	cmd.AddCommand(newConfigSetCmd())
	cmd.AddCommand(newConfigShowCmd())
	return cmd
}

func newConfigSetCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "set",
		Short: "Set configuration values",
	}
	cmd.AddCommand(newConfigSetModeCmd())
	return cmd
}

func newConfigSetModeCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "mode [local-first|cloud-first]",
		Short: "Set the default execution mode",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			mode := args[0]
			if err := cliconfig.ValidateMode(mode); err != nil {
				return err
			}
			home, homeErr := cliconfig.ResolveHomeDir()
			if homeErr != nil {
				return homeErr
			}
			cfg, loadErr := cliconfig.LoadConfig(home)
			if loadErr != nil {
				return loadErr
			}
			cfg.Mode = mode
			if err := cliconfig.SaveConfig(home, cfg); err != nil {
				return err
			}
			_, err := fmt.Fprintf(cmd.OutOrStdout(), "默认执行模式已更新为 %s\n配置文件：%s\n", mode, cliconfig.ConfigFile(home))
			return err
		},
	}
}

func newConfigShowCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "show",
		Short: "Show current configuration",
		RunE: func(cmd *cobra.Command, args []string) error {
			home, resolveErr := cliconfig.ResolveHomeDir()
			if resolveErr != nil {
				return resolveErr
			}
			resolved, configErr := cliconfig.ResolveConfig(home)
			if configErr != nil {
				return configErr
			}
			cache, cacheErr := cliconfig.LoadEnvCache(home)
			if cacheErr != nil {
				return cacheErr
			}
			_, err := fmt.Fprintf(
				cmd.OutOrStdout(),
				"当前配置\n- mode: %s\n- api_key: %s (%s)\n- endpoint: %s (%s)\n- credential_store: %s\n- config_file: %s\n- env_cache: %s\n- last_env_check: %s\n",
				resolved.Mode,
				displaySecret(resolved.APIKey),
				resolved.APIKeySource,
				resolved.Endpoint,
				resolved.EndpointSource,
				resolved.CredentialStore,
				resolved.ConfigPath,
				resolved.EnvCachePath,
				displayValueOrFallback(cache.CheckedAt, "never"),
			)
			return err
		},
	}
}

func displaySecret(value string) string {
	if value == "" {
		return "<not configured>"
	}
	return cliconfig.MaskSecret(value)
}

func displayValueOrFallback(value string, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}
