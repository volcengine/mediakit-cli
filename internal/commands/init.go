//go:build !mediakit_cloud_only

package commands

import (
	"bufio"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"mediakit-cli/internal/auth"
	"mediakit-cli/internal/buildenv"
	cliconfig "mediakit-cli/internal/config"
)

func newInitCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize non-secret mediakit-cli configuration",
		Long: `Initialize execution mode, Cloud endpoint, Local output path and runtime label.

Authentication is never accepted as a flag or persisted in config. Provide
MEDIAKIT_API_KEY at runtime for Cloud execution.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			home, err := cliconfig.ResolveHomeDir()
			if err != nil {
				return err
			}
			if err := cliconfig.EnsureConfigDir(home); err != nil {
				return err
			}
			cfg, err := cliconfig.LoadConfig(home)
			if err != nil {
				return err
			}

			yes, _ := cmd.Flags().GetBool("yes")
			mode, _ := cmd.Flags().GetString("mode")
			endpoint, _ := cmd.Flags().GetString("endpoint")
			outputPath, _ := cmd.Flags().GetString("output-path")
			runtimeName, _ := cmd.Flags().GetString("runtime")

			if mode == "" {
				if yes {
					mode = cfg.Mode
					if mode == "" {
						mode = cliconfig.DefaultMode
					}
				} else {
					reader := bufio.NewReader(cmd.InOrStdin())
					mode, err = promptMode(reader, cmd.OutOrStdout())
					if err != nil {
						return err
					}
				}
			}
			if err := cliconfig.ValidateMode(mode); err != nil {
				return err
			}
			cfg.Mode = mode

			if endpoint != "" {
				cfg.Endpoint = strings.TrimSpace(endpoint)
			}
			if runtimeName != "" {
				cfg.Runtime = strings.TrimSpace(runtimeName)
			}
			if outputPath != "" {
				cfg.OutputPath = outputPath
			}

			resolvedOutput, outputSource, err := cliconfig.ResolveOutputPathFromConfig(
				home,
				cfg,
			)
			if err != nil {
				return err
			}
			if mode == cliconfig.ModeLocalFirst {
				if err := cliconfig.EnsureOutputDir(resolvedOutput); err != nil {
					return err
				}
				cache, refreshErr := cliconfig.RefreshEnvCacheForConfig(home, cfg)
				if refreshErr != nil {
					return refreshErr
				}
				if missing := cliconfig.MissingRequiredLocalDependencies(cache); len(missing) > 0 {
					return fmt.Errorf(
						"local-first 依赖缺失: %s",
						strings.Join(missing, ", "),
					)
				}
			}

			if err := cliconfig.SaveConfig(home, cfg); err != nil {
				return err
			}
			authKind := "not_configured"
			if context, authErr := auth.Resolve(); authErr == nil {
				authKind = string(context.Kind())
			} else if mode == cliconfig.ModeCloudFirst {
				return authErr
			}

			_, err = fmt.Fprintf(
				cmd.OutOrStdout(),
				"初始化完成\n- 默认模式: %s\n- Cloud 鉴权: %s\n- API Key 环境变量: %s\n- LogID 环境变量: %s\n- Endpoint: %s\n- Runtime: %s\n- 本地输出目录: %s (%s)\n- 配置文件: %s\n",
				cfg.Mode,
				authKind,
				buildenv.CloudAPIKey,
				buildenv.CloudLogID,
				displayValueOrFallback(cfg.Endpoint, cliconfig.DefaultEndpoint),
				displayValueOrFallback(cfg.Runtime, "auto-detect"),
				resolvedOutput,
				outputSource,
				cliconfig.ConfigFile(home),
			)
			return err
		},
	}
	cmd.Flags().String("mode", "", "执行模式：local-first 或 cloud-first")
	cmd.Flags().String("endpoint", "", "自定义 Cloud API endpoint")
	cmd.Flags().String("output-path", "", "本地文件输出目录")
	cmd.Flags().String("runtime", "", "运行环境标识；Skill 应显式传入当前 Agent 宿主")
	strictBool(cmd.Flags(), "yes", false, "非交互模式，跳过模式选择")
	return cmd
}
