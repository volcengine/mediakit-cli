package commands

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	cliconfig "mediakit-cli/internal/config"
)

func newInitCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "Initialize mediakit-cli configuration",
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

			reader := bufio.NewReader(cmd.InOrStdin())
			out := cmd.OutOrStdout()

			mode, err := promptMode(reader, out)
			if err != nil {
				return err
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
					return fmt.Errorf("local-first 依赖缺失: %s", strings.Join(missing, ", "))
				}
			}

			if _, err := fmt.Fprintln(out, "MediaKit API Key 获取地址："); err != nil {
				return err
			}
			if _, err := fmt.Fprintln(out, "https://console.volcengine.com/imp/ai-mediakit/settings"); err != nil {
				return err
			}
			apiKey, apiKeyErr := promptRequired(reader, out, "请输入 apikey：")
			if apiKeyErr != nil {
				return apiKeyErr
			}

			endpoint, endpointErr := promptLine(reader, out, "是否自定义 MEDIAKIT_ENDPOINT？可直接回车跳过：")
			if endpointErr != nil {
				return endpointErr
			}
			outputPath, outputSource, outputPathErr := cliconfig.ResolveOutputPath(home)
			if outputPathErr != nil {
				return outputPathErr
			}
			if err := cliconfig.EnsureOutputDir(outputPath); err != nil {
				return err
			}
			if _, err := fmt.Fprintf(out, "本地文件输出目录：%s（来源：%s，可通过 %s 覆盖；默认 %s）\n", outputPath, outputSource, cliconfig.EnvOutputPath, cliconfig.DefaultOutputPath(home)); err != nil {
				return err
			}

			store, storeErr := promptCredentialStore(reader, out)
			if storeErr != nil {
				return storeErr
			}
			fileCfg.CredentialStore = store

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
				content := "\n# mediakit-cli credentials\n" + formatExportLines(apiKey, endpoint) + "\n"
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
				if _, err := fmt.Fprintln(out, formatExportLines(apiKey, endpoint)); err != nil {
					return err
				}
			}

			if _, err := fmt.Fprintln(out, "初始化完成"); err != nil {
				return err
			}
			if _, err := fmt.Fprintf(out, "- 默认模式：%s\n", fileCfg.Mode); err != nil {
				return err
			}
			if _, err := fmt.Fprintf(out, "- API Key：已配置（%s）\n", store); err != nil {
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
			return nil
		},
	}
}
