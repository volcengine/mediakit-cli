package commands

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"

	cliconfig "mediakit-cli/internal/config"
)

func newInitCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize mediakit-cli configuration",
		Long: `Initialize mediakit-cli configuration.

Supports both interactive and non-interactive modes.
For non-interactive (agent-friendly) usage, pass all required flags:

  mediakit-cli init --mode cloud-first --api-key <key> --runtime <runtime> --surface cli --yes
  mediakit-cli init --mode local-first --api-key <key> --endpoint <url> --output-path ~/mediakit-output --runtime <runtime> --surface cli --credential-store config --yes`,
		RunE: func(cmd *cobra.Command, args []string) error {
			home, homeErr := cliconfig.ResolveHomeDir()
			if homeErr != nil {
				return homeErr
			}
			if err := cliconfig.EnsureConfigDir(home); err != nil {
				return err
			}

			fileCfg, loadErr := cliconfig.LoadConfig(home)
			if loadErr != nil {
				return loadErr
			}

			// Check if non-interactive mode
			yes, _ := cmd.Flags().GetBool("yes")
			flagMode, _ := cmd.Flags().GetString("mode")
			flagAPIKey, _ := cmd.Flags().GetString("api-key")
			flagEndpoint, _ := cmd.Flags().GetString("endpoint")
			flagStore, _ := cmd.Flags().GetString("credential-store")
			flagOutputPath, _ := cmd.Flags().GetString("output-path")
			flagRuntime, _ := cmd.Flags().GetString("runtime")
			flagSurface, _ := cmd.Flags().GetString("surface")

			if yes {
				return runNonInteractive(cmd, home, fileCfg, flagMode, flagAPIKey, flagEndpoint, flagStore, flagOutputPath, flagSurface, flagRuntime)
			}

			// Interactive mode (original behavior)
			reader := bufio.NewReader(cmd.InOrStdin())
			out := cmd.OutOrStdout()

			// Use flag value if provided, otherwise prompt
			var mode string
			if flagMode != "" {
				if err := cliconfig.ValidateMode(flagMode); err != nil {
					return err
				}
				mode = flagMode
			} else {
				var err error
				mode, err = promptMode(reader, out)
				if err != nil {
					return err
				}
			}
			fileCfg.Mode = mode

			cache, refreshErr := cliconfig.RefreshEnvCache(home)
			if refreshErr != nil {
				return refreshErr
			}
			if _, err := fmt.Fprintf(out, "本地依赖检查完成：\n%s\n", renderLocalDependencyStatus(cache)); err != nil {
				return err
			}
			if _, err := fmt.Fprintf(out, "缺失本地依赖清单：\n%s\n", renderMissingLocalDependencyInstallList(cache)); err != nil {
				return err
			}
			if mode == cliconfig.ModeLocalFirst {
				missing := cliconfig.MissingRequiredLocalDependencies(cache)
				if len(missing) > 0 {
					_, _ = fmt.Fprintln(out, "")
					_, _ = fmt.Fprintln(out, cliconfig.InstallGuideText(missing))
					return fmt.Errorf("local-first 模式依赖缺失，请按上述指引安装后重试")
				}
			}

			if _, err := fmt.Fprintln(out, "MediaKit API Key 获取地址："); err != nil {
				return err
			}
			if _, err := fmt.Fprintln(out, "https://console.volcengine.com/imp/ai-mediakit/settings"); err != nil {
				return err
			}

			var apiKey string
			if flagAPIKey != "" {
				apiKey = flagAPIKey
			} else {
				var err error
				apiKey, err = promptRequired(reader, out, "请输入 apikey：")
				if err != nil {
					return err
				}
			}

			var endpoint string
			if flagEndpoint != "" {
				endpoint = flagEndpoint
			} else {
				var err error
				endpoint, err = promptLine(reader, out, "是否自定义 MEDIAKIT_ENDPOINT？可直接回车跳过：")
				if err != nil {
					return err
				}
			}

			var outputPath, outputSource string
			if flagOutputPath != "" {
				outputPath = flagOutputPath
				outputSource = "flag"
				fileCfg.OutputPath = flagOutputPath
			} else {
				var err error
				outputPath, outputSource, err = cliconfig.ResolveOutputPath(home)
				if err != nil {
					return err
				}
			}
			if err := cliconfig.EnsureOutputDir(outputPath); err != nil {
				return err
			}
			if _, err := fmt.Fprintf(out, "本地文件输出目录：%s（来源：%s，可通过 %s 或 --output-path 覆盖；默认 %s）\n", outputPath, outputSource, cliconfig.EnvOutputPath, cliconfig.DefaultOutputPath(home)); err != nil {
				return err
			}

			var store string
			if flagStore != "" {
				if err := validateCredentialStore(flagStore); err != nil {
					return err
				}
				store = flagStore
			} else {
				var err error
				store, err = promptCredentialStore(reader, out)
				if err != nil {
					return err
				}
			}
			fileCfg.CredentialStore = store
			if flagSurface != "" {
				fileCfg.Surface = flagSurface
			}
			if flagRuntime != "" {
				fileCfg.Runtime = flagRuntime
			}

			if err := applyCredentialStore(cmd.OutOrStdout(), home, fileCfg, store, apiKey, endpoint); err != nil {
				return err
			}

			return printInitSummary(cmd.OutOrStdout(), home, fileCfg, endpoint, outputPath)
		},
	}

	cmd.Flags().String("mode", "", "执行模式：local-first 或 cloud-first")
	cmd.Flags().String("api-key", "", "MediaKit API Key")
	cmd.Flags().String("endpoint", "", "自定义 MEDIAKIT_ENDPOINT")
	cmd.Flags().String("output-path", "", "本地文件输出目录（默认 ~/.mediakit/temp，可通过 MEDIAKIT_OUTPUT_PATH 环境变量覆盖）")
	cmd.Flags().String("surface", "", "请求来源环境变量 MEDIAKIT_SURFACE（默认 cli；如 skill、plugin）")
	cmd.Flags().String("runtime", "", "运行时环境变量 MEDIAKIT_RUNTIME（如 arkclaw、claude、cursor）")
	cmd.Flags().String("credential-store", "", "凭据保存方式：config、shell 或 export")
	cmd.Flags().Bool("yes", false, "非交互模式，跳过所有确认提示（Agent 友好）")

	return cmd
}

func runNonInteractive(cmd *cobra.Command, home string, fileCfg cliconfig.Config, flagMode, flagAPIKey, flagEndpoint, flagStore, flagOutputPath, flagSurface, flagRuntime string) error {
	out := cmd.OutOrStdout()

	// Validate mode
	if flagMode == "" {
		flagMode = cliconfig.DefaultMode
	}
	if err := cliconfig.ValidateMode(flagMode); err != nil {
		return err
	}
	fileCfg.Mode = flagMode

	// Refresh env cache
	cache, refreshErr := cliconfig.RefreshEnvCache(home)
	if refreshErr != nil {
		return refreshErr
	}
	if flagMode == cliconfig.ModeLocalFirst {
		missing := cliconfig.MissingRequiredLocalDependencies(cache)
		if len(missing) > 0 {
			return fmt.Errorf("local-first 依赖缺失: %s", strings.Join(missing, ", "))
		}
	}

	// API Key: flag > env > error
	apiKey := flagAPIKey
	if apiKey == "" {
		apiKey = strings.TrimSpace(os.Getenv(cliconfig.EnvAPIKey))
	}
	if apiKey == "" {
		return fmt.Errorf("--api-key 是必填参数（非交互模式下）")
	}

	endpoint := flagEndpoint

	// Output path: flag > env > default
	var outputPath string
	if flagOutputPath != "" {
		outputPath = flagOutputPath
		fileCfg.OutputPath = flagOutputPath
	} else {
		resolved, _, err := cliconfig.ResolveOutputPath(home)
		if err != nil {
			return err
		}
		outputPath = resolved
	}
	if err := cliconfig.EnsureOutputDir(outputPath); err != nil {
		return err
	}

	// Credential store: default to "config" in non-interactive mode
	store := flagStore
	if store == "" {
		store = "config"
	}
	if err := validateCredentialStore(store); err != nil {
		return err
	}
	fileCfg.CredentialStore = store
	if flagSurface != "" {
		fileCfg.Surface = flagSurface
	}
	if flagRuntime != "" {
		fileCfg.Runtime = flagRuntime
	}

	if err := applyCredentialStore(out, home, fileCfg, store, apiKey, endpoint); err != nil {
		return err
	}

	_ = cache // suppress unused warning
	return printInitSummary(out, home, fileCfg, endpoint, outputPath)
}

func validateCredentialStore(store string) error {
	switch store {
	case "config", "shell", "export":
		return nil
	default:
		return fmt.Errorf("credential-store 仅支持 config、shell 或 export，当前值：%s", store)
	}
}

func applyCredentialStore(out io.Writer, home string, fileCfg cliconfig.Config, store, apiKey, endpoint string) error {

	switch store {
	case "config":
		fileCfg.APIKey = apiKey
		fileCfg.Endpoint = endpoint
		if err := cliconfig.SaveConfig(home, fileCfg); err != nil {
			return err
		}
	case "shell":
		fileCfg.APIKey = ""
		fileCfg.Endpoint = ""
		if err := cliconfig.SaveConfig(home, fileCfg); err != nil {
			return err
		}
		profile, profileErr := cliconfig.DetectShellProfile(home)
		if profileErr != nil {
			return profileErr
		}
		content := "\n# mediakit-cli credentials\n" + formatExportLines(apiKey, endpoint, fileCfg.Surface, fileCfg.Runtime) + "\n"
		f, openErr := os.OpenFile(profile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
		if openErr != nil {
			return openErr
		}
		if _, err := f.WriteString(content); err != nil {
			f.Close()
			return err
		}
		if err := f.Close(); err != nil {
			return err
		}
		if _, err := fmt.Fprintf(out, "已写入 shell 配置文件：%s\n", profile); err != nil {
			return err
		}
	case "export":
		fileCfg.APIKey = ""
		fileCfg.Endpoint = ""
		if err := cliconfig.SaveConfig(home, fileCfg); err != nil {
			return err
		}
		if _, err := fmt.Fprintln(out, "请手动执行以下命令："); err != nil {
			return err
		}
		if _, err := fmt.Fprintln(out, formatExportLines(apiKey, endpoint, fileCfg.Surface, fileCfg.Runtime)); err != nil {
			return err
		}
	}
	return nil
}

func printInitSummary(out io.Writer, home string, fileCfg cliconfig.Config, endpoint, outputPath string) error {
	if _, err := fmt.Fprintln(out, "初始化完成"); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(out, "- 默认模式：%s\n", fileCfg.Mode); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(out, "- API Key：已配置（%s）\n", fileCfg.CredentialStore); err != nil {
		return err
	}
	if endpoint != "" {
		if _, err := fmt.Fprintf(out, "- MEDIAKIT_ENDPOINT：%s\n", endpoint); err != nil {
			return err
		}
	} else if _, err := fmt.Fprintf(out, "- MEDIAKIT_ENDPOINT：默认值（%s）\n", cliconfig.DefaultEndpoint); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(out, "- 配置文件：%s\n", cliconfig.ConfigFile(home)); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(out, "- 环境缓存：%s\n", cliconfig.EnvCacheFile(home)); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(out, "- 本地文件输出目录：%s（可通过 %s 覆盖）\n", outputPath, cliconfig.EnvOutputPath); err != nil {
		return err
	}
	if fileCfg.Surface != "" {
		if _, err := fmt.Fprintf(out, "- MEDIAKIT_SURFACE：%s\n", fileCfg.Surface); err != nil {
			return err
		}
	}
	if fileCfg.Runtime != "" {
		if _, err := fmt.Fprintf(out, "- MEDIAKIT_RUNTIME：%s\n", fileCfg.Runtime); err != nil {
			return err
		}
	}
	return nil
}
