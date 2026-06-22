package modes

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"mediakit-cli/internal/cloud"
	cliconfig "mediakit-cli/internal/config"
	"mediakit-cli/internal/local"
	"mediakit-cli/internal/local/admission"
)

type CapabilityRuntimeMeta struct {
	Name           string
	Domain         string
	Description    string
	CloudOnly      bool
	LocalSupported bool
	LocalSource    string
	LocalDeps      []string
}

type Decision struct {
	Mode    string
	Warning string
}

type Resolver struct{}

const (
	queryTaskCapabilityName = "query-task"
	fetchFileCapabilityName = "fetch-file"
)

func Dispatch(cmd *cobra.Command, meta CapabilityRuntimeMeta, params map[string]any) error {
	meta = ApplyRuntimeConstraints(meta)
	home, err := cliconfig.ResolveHomeDir()
	if err != nil {
		return err
	}

	resolved, err := cliconfig.ResolveConfig(home)
	if err != nil {
		return err
	}
	resolved.Mode, err = resolveCommandModeOverride(cmd, resolved.Mode)
	if err != nil {
		return err
	}
	if normalizeCapabilityName(meta.Name) == fetchFileCapabilityName {
		cloudMode, err := cmd.Flags().GetBool("cloud")
		if err != nil {
			return err
		}
		if cloudMode {
			return fmt.Errorf("fetch-file 是本地文件拉取工具，不支持 --cloud")
		}
	}
	cache, err := cliconfig.LoadEnvCache(home)
	if err != nil {
		return err
	}
	if cache.CheckedAt == "" {
		cache, err = cliconfig.RefreshEnvCache(home)
		if err != nil {
			return err
		}
	}

	decision, err := resolveDecision(meta, resolved, cache)
	if err != nil {
		return err
	}
	if decision.Warning != "" {
		if _, err := fmt.Fprintf(cmd.ErrOrStderr(), "Warning: %s\n", decision.Warning); err != nil {
			return err
		}
	}

	switch decision.Mode {
	case "local":
		return local.Execute(cmd, meta.Name, params)
	case "cloud":
		return cloud.Execute(cmd, meta.Name, params, resolved.APIKey, resolved.Endpoint, resolved.Surface, resolved.Runtime)
	default:
		return fmt.Errorf("unsupported execution mode: %s", decision.Mode)
	}
}

func ModeLabel(meta CapabilityRuntimeMeta) string {
	meta = ApplyRuntimeConstraints(meta)
	if normalizeCapabilityName(meta.Name) == fetchFileCapabilityName {
		return "local only"
	}
	switch {
	case meta.CloudOnly:
		return "cloud only"
	case meta.LocalSupported:
		return "cloud + local"
	default:
		return "cloud"
	}
}

func resolveDecision(meta CapabilityRuntimeMeta, resolved cliconfig.ResolvedConfig, cache cliconfig.EnvCache) (Decision, error) {
	meta = ApplyRuntimeConstraints(meta)
	cloudReady := resolved.APIKey != ""
	localReady, localReason := evaluateLocalReadiness(meta, cache)
	if normalizeCapabilityName(meta.Name) == fetchFileCapabilityName {
		if localReady {
			return Decision{Mode: "local"}, nil
		}
		if localReason == "" {
			localReason = "本地执行条件不满足"
		}
		return Decision{}, fmt.Errorf("%s 是本地文件拉取工具；%s", meta.Name, localReason)
	}

	switch resolved.Mode {
	case cliconfig.ModeLocalFirst:
		if meta.CloudOnly {
			if cloudReady {
				return Decision{Mode: "cloud"}, nil
			}
			return Decision{}, fmt.Errorf("%s 仅支持 cloud 执行，但当前未配置 MEDIAKIT_API_KEY", meta.Name)
		}
		if localReady {
			return Decision{Mode: "local"}, nil
		}
		if cloudReady {
			warning := "本地依赖缺失，降级到云端执行"
			if localReason != "" {
				warning = localReason + "，降级到云端执行"
			}
			return Decision{Mode: "cloud", Warning: warning}, nil
		}
		if localReason == "" {
			localReason = "本地执行条件不满足"
		}
		return Decision{}, fmt.Errorf("%s；且云端凭据未配置，无法降级", localReason)

	case cliconfig.ModeCloudFirst:
		if cloudReady {
			return Decision{Mode: "cloud"}, nil
		}
		if meta.CloudOnly {
			return Decision{}, fmt.Errorf("云端凭据未配置，%s 不支持本地执行", meta.Name)
		}
		if localReady {
			return Decision{Mode: "local", Warning: "云端凭据未配置，降级到本地执行"}, nil
		}
		if localReason == "" {
			localReason = "本地依赖不满足"
		}
		return Decision{}, fmt.Errorf("云端凭据未配置；%s", localReason)

	default:
		return Decision{}, fmt.Errorf("unsupported config mode: %s", resolved.Mode)
	}
}

func evaluateLocalReadiness(meta CapabilityRuntimeMeta, cache cliconfig.EnvCache) (bool, string) {
	meta = ApplyRuntimeConstraints(meta)
	if !meta.LocalSupported {
		return false, "该命令不支持本地执行"
	}
	if !local.Has(meta.Name) {
		return false, "本地处理器未实现"
	}

	missing := make([]string, 0, len(meta.LocalDeps))
	missingDetails := make([]string, 0, len(meta.LocalDeps))
	for _, dep := range meta.LocalDeps {
		dep = strings.TrimSpace(dep)
		if dep == "" {
			continue
		}
		if !admission.IsAllowedDependency(dep) {
			return false, "本地依赖不在白名单内: " + dep
		}
		status, ok := cache.Tools[dep]
		if !ok || !status.Available {
			missing = append(missing, dep)
			detail := dep
			if ok && strings.TrimSpace(status.Reason) != "" {
				detail += "(" + status.Reason + ")"
			}
			if hint := localDependencyInstallHint(dep); hint != "" {
				detail += "，" + hint
			}
			missingDetails = append(missingDetails, detail)
		}
	}

	if len(missing) > 0 {
		return false, "本地依赖缺失: " + strings.Join(missing, ", ") + "；详情: " + strings.Join(missingDetails, "；")
	}
	return true, ""
}

func localDependencyInstallHint(dep string) string {
	switch dep {
	case "ffmpeg":
		return "请安装 ffmpeg >= 5.1"
	case "ffprobe":
		return "请安装 ffprobe >= 5.1"
	case "openh264":
		return "请安装或切换到包含 libopenh264 或 h264_videotoolbox 编码器的 FFmpeg"
	case "h264_videotoolbox":
		return "请安装或切换到包含 h264_videotoolbox 编码器的 FFmpeg（macOS 可用）"
	case "demuxer":
		return "请安装或切换到包含 concat demuxer 的 FFmpeg"
	case "libmp3lame":
		return "请安装或切换到包含 libmp3lame 编码器的 FFmpeg"
	case "prores_ks":
		return "请安装或切换到包含 prores_ks 编码器的 FFmpeg"
	case "libass":
		return "请安装或切换到包含 subtitles/ass 字幕滤镜的 FFmpeg"
	case "libfreetype", "libfontconfig", "libfribidi", "libharfbuzz":
		return "请安装或切换到包含字幕渲染相关库的 FFmpeg"
	case "libpng":
		return "请安装或切换到包含 PNG 解码能力的 FFmpeg"
	case "libjpeg-turbo":
		return "请安装或切换到包含 JPEG 解码能力的 FFmpeg"
	default:
		return ""
	}
}

func ApplyRuntimeConstraints(meta CapabilityRuntimeMeta) CapabilityRuntimeMeta {
	if normalizeCapabilityName(meta.Name) == fetchFileCapabilityName {
		meta.CloudOnly = false
		meta.LocalSupported = true
		meta.LocalSource = "generated"
		meta.LocalDeps = nil
		return meta
	}
	if normalizeCapabilityName(meta.Name) == queryTaskCapabilityName {
		meta.CloudOnly = true
		meta.LocalSupported = false
		meta.LocalSource = ""
		meta.LocalDeps = nil
	}
	if meta.CloudOnly {
		meta.LocalSupported = false
		meta.LocalSource = ""
		meta.LocalDeps = nil
	}
	return meta
}

func normalizeCapabilityName(name string) string {
	name = strings.TrimSpace(name)
	name = strings.ReplaceAll(name, "_", "-")
	return strings.ToLower(name)
}

func resolveCommandModeOverride(cmd *cobra.Command, currentMode string) (string, error) {
	localMode, err := cmd.Flags().GetBool("local")
	if err != nil {
		return "", err
	}
	cloudMode, err := cmd.Flags().GetBool("cloud")
	if err != nil {
		return "", err
	}
	if localMode && cloudMode {
		return "", fmt.Errorf("`--local` 与 `--cloud` 不能同时使用")
	}
	if localMode {
		return cliconfig.ModeLocalFirst, nil
	}
	if cloudMode {
		return cliconfig.ModeCloudFirst, nil
	}
	return currentMode, nil
}
