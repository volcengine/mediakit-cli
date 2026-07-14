package commands

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"mediakit-cli/internal/cloud"
	cliconfig "mediakit-cli/internal/config"
)

func newConfigCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Manage mediakit-cli configuration",
	}

	cmd.AddCommand(newConfigSetCmd())
	cmd.AddCommand(newConfigShowCmd())
	cmd.AddCommand(newConfigHeadersCmd())
	return cmd
}

func newConfigSetCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "set",
		Short: "Set configuration values",
	}
	cmd.AddCommand(newConfigSetModeCmd())
	cmd.AddCommand(newConfigSetOutputPathCmd())
	cmd.AddCommand(newConfigSetUpdateCheckCmd())
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

func newConfigSetOutputPathCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "output-path [path]",
		Short: "Set the local file output directory",
		Long:  "Set the local file output directory. Priority: --output-path flag > MEDIAKIT_OUTPUT_PATH env > config > ~/.mediakit/temp",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			outputPath := args[0]
			home, homeErr := cliconfig.ResolveHomeDir()
			if homeErr != nil {
				return homeErr
			}
			cfg, loadErr := cliconfig.LoadConfig(home)
			if loadErr != nil {
				return loadErr
			}
			cfg.OutputPath = outputPath
			if err := cliconfig.SaveConfig(home, cfg); err != nil {
				return err
			}
			_, err := fmt.Fprintf(cmd.OutOrStdout(), "本地文件输出目录已更新为 %s\n配置文件：%s\n", outputPath, cliconfig.ConfigFile(home))
			return err
		},
	}
}

func newConfigSetUpdateCheckCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "update-check [on|off]",
		Short: "Enable or disable the automatic version update check",
		Long:  "Enable or disable the automatic version update check. Default: on. The MEDIAKIT_DISABLE_UPDATE_CHECK env var overrides this setting.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			disabled, err := parseUpdateCheckToggle(args[0])
			if err != nil {
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
			cfg.DisableUpdateCheck = disabled
			if err = cliconfig.SaveConfig(home, cfg); err != nil {
				return err
			}
			_, err = fmt.Fprintf(cmd.OutOrStdout(), "自动更新检查已%s\n配置文件：%s\n", updateCheckStateLabel(!disabled), cliconfig.ConfigFile(home))
			return err
		},
	}
}

func parseUpdateCheckToggle(value string) (bool, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "on", "true", "enable", "enabled", "1":
		return false, nil
	case "off", "false", "disable", "disabled", "0":
		return true, nil
	default:
		return false, fmt.Errorf("invalid value %q: expected on or off", value)
	}
}

func updateCheckStateLabel(enabled bool) string {
	if enabled {
		return "开启"
	}
	return "关闭"
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
				"当前配置\n- mode: %s\n- api_key: %s (%s)\n- endpoint: %s (%s)\n- output_path: %s (%s)\n- credential_store: %s\n- update_check: %s\n- config_file: %s\n- env_cache: %s\n- last_env_check: %s\n",
				resolved.Mode,
				displaySecret(resolved.APIKey),
				resolved.APIKeySource,
				resolved.Endpoint,
				resolved.EndpointSource,
				resolved.OutputPath,
				resolved.OutputPathSource,
				resolved.CredentialStore,
				updateCheckStateLabel(!resolved.DisableUpdateCheck),
				resolved.ConfigPath,
				resolved.EnvCachePath,
				displayValueOrFallback(cache.CheckedAt, "never"),
			)
			return err
		},
	}
}

func newConfigHeadersCmd() *cobra.Command {
	var asJSON bool
	cmd := &cobra.Command{
		Use:   "headers",
		Short: "Show locally resolved cloud request headers",
		Long:  "Show locally resolved non-sensitive cloud request headers. This command does not send a request and never prints Authorization.",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			home, resolveErr := cliconfig.ResolveHomeDir()
			if resolveErr != nil {
				return resolveErr
			}
			resolved, configErr := cliconfig.ResolveConfig(home)
			if configErr != nil {
				return configErr
			}
			headers := cloud.PreviewHeaders(resolved.Surface, resolved.Runtime)
			if asJSON {
				encoder := json.NewEncoder(cmd.OutOrStdout())
				encoder.SetEscapeHTML(false)
				encoder.SetIndent("", "  ")
				return encoder.Encode(map[string]any{
					"headers":     headers,
					"config_file": resolved.ConfigPath,
				})
			}
			if _, err := fmt.Fprintf(cmd.OutOrStdout(), "本地请求 Header（未发送请求，不包含 Authorization）\n"); err != nil {
				return err
			}
			for _, key := range []string{"Accept", "Content-Type", "x-surface", "X-Amk-Cli-Runtime", "X-Amk-Task-Source", "X-Amk-Cli-Version"} {
				if _, err := fmt.Fprintf(cmd.OutOrStdout(), "- %s: %s\n", key, headers[key]); err != nil {
					return err
				}
			}
			_, err := fmt.Fprintf(cmd.OutOrStdout(), "配置文件：%s\n", resolved.ConfigPath)
			return err
		},
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "Output structured JSON")
	return cmd
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
