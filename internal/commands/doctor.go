package commands

import (
	"fmt"
	"runtime"
	"strings"

	"github.com/spf13/cobra"

	cliconfig "mediakit-cli/internal/config"
	"mediakit-cli/internal/local/admission"
)

func newDoctorCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "doctor",
		Short: "Inspect local and cloud runtime readiness",
		RunE: func(cmd *cobra.Command, args []string) error {
			home, err := cliconfig.ResolveHomeDir()
			if err != nil {
				return err
			}
			cache, err := cliconfig.RefreshEnvCache(home)
			if err != nil {
				return err
			}
			resolved, err := cliconfig.ResolveConfig(home)
			if err != nil {
				return err
			}
			_, err = fmt.Fprintf(
				cmd.OutOrStdout(),
				"doctor 检查完成\n- mode: %s\n- cloud_ready: %t\n- local_ready: %t\n- cache_file: %s\n\n=== Local Dependencies ===\n%s\n\n=== Missing Local Dependencies ===\n%s\n\n=== Local Tool Admission ===\n%s\n\n审批要求:\n- %s\n- 标准清单: %s\n",
				resolved.Mode,
				cache.CloudReady,
				cache.LocalReady,
				cliconfig.EnvCacheFile(home),
				renderLocalDependencyStatus(cache),
				renderMissingLocalDependencyInstallList(cache),
				renderAdmissionPolicies(),
				admission.ApprovalPrompt(),
				strings.Join(admission.ApprovalTemplate(), " / "),
			)
			if err != nil {
				return err
			}
			if resolved.Mode == cliconfig.ModeLocalFirst {
				missing := cliconfig.MissingRequiredLocalDependencies(cache)
				if len(missing) > 0 {
					return fmt.Errorf("local-first 依赖缺失: %s", strings.Join(missing, ", "))
				}
			}
			return nil
		},
	}
}

func summarizeTool(status cliconfig.ToolStatus) string {
	if status.Available {
		if status.Version != "" {
			return "ok (" + status.Version + ")"
		}
		return "ok"
	}
	if status.Reason != "" {
		return "missing (" + status.Reason + ")"
	}
	return "missing"
}

func renderAdmissionPolicies() string {
	lines := make([]string, 0, len(admission.CurrentPolicies()))
	for _, policy := range admission.CurrentPolicies() {
		lines = append(lines, admission.PolicyLine(policy))
		lines = append(lines, fmt.Sprintf("  role: %s", policy.Role))
		lines = append(lines, fmt.Sprintf("  boundary: %s", policy.Boundary))
		lines = append(lines, fmt.Sprintf("  approval: %s", policy.ApprovalRequiredFor))
	}
	return strings.Join(lines, "\n")
}

func renderLocalDependencyStatus(cache cliconfig.EnvCache) string {
	lines := make([]string, 0, len(admission.AllowedDependencyNames()))
	for _, dep := range admission.AllowedDependencyNames() {
		lines = append(lines, fmt.Sprintf("- %s: %s", dep, summarizeTool(cache.Tools[dep])))
	}
	return strings.Join(lines, "\n")
}

func renderMissingLocalDependencyInstallList(cache cliconfig.EnvCache) string {
	requiredMissing := cliconfig.MissingRequiredLocalDependencies(cache)
	optionalMissing := []string{}
	requiredSet := stringSet(requiredMissing)
	for _, dep := range admission.AllowedDependencyNames()[2:] {
		status := cache.Tools[dep]
		if status.Available {
			continue
		}
		if _, required := requiredSet[dep]; required {
			continue
		}
		optionalMissing = append(optionalMissing, dep)
	}
	if len(requiredMissing) == 0 && len(optionalMissing) == 0 {
		return "无缺失依赖。"
	}
	lines := []string{}
	if len(requiredMissing) > 0 {
		lines = append(lines, "请用户自行安装以下必需本地工具后重试：")
		for _, dep := range requiredMissing {
			lines = append(lines, fmt.Sprintf("- %s", dep))
		}
	}
	if len(optionalMissing) > 0 && cache.Tools["ffmpeg"].Available {
		if len(lines) > 0 {
			lines = append(lines, "")
		}
		lines = append(lines, "以下为可选 FFmpeg 能力未检测到；如需相关本地能力，请安装包含对应能力的 FFmpeg 构建：")
		for _, dep := range optionalMissing {
			lines = append(lines, fmt.Sprintf("- %s", dep))
		}
	}
	lines = append(lines, renderLocalInstallGuide(requiredMissing, optionalMissing)...)
	return strings.Join(lines, "\n")
}

func renderLocalInstallGuide(requiredMissing []string, optionalMissing []string) []string {
	lines := []string{
		"",
		fmt.Sprintf("当前平台: %s", runtime.GOOS),
		"安装要求: 请安装 FFmpeg/FFprobe >= 5.1；如需可选能力，请确保 FFmpeg 构建包含对应白名单能力。",
	}
	switch runtime.GOOS {
	case "darwin":
		lines = append(lines,
			"macOS: 请安装 FFmpeg/FFprobe >= 5.1，并确认 `ffmpeg` 与 `ffprobe` 已加入 PATH。",
			"注意: `brew install openh264 libass` 只安装库；当前 PATH 中的 `ffmpeg` 仍必须在编译时启用对应能力。",
			"验证: `ffmpeg -hide_banner -encoders | grep libopenh264` 与 `ffmpeg -hide_banner -filters | grep subtitles` 都应有输出。",
		)
		if containsDependency(optionalMissing, "openh264") {
			lines = append(lines,
				"openh264: 请安装或切换到包含 `--enable-libopenh264` 且暴露 `libopenh264` 编码器的 FFmpeg 构建。",
			)
		}
		if containsDependency(optionalMissing, "libass") {
			lines = append(lines,
				"libass: 字幕烧录需要 `subtitles`/`ass` 滤镜，请安装或切换到包含 `--enable-libass` 的 FFmpeg 构建。",
			)
		}
	case "windows":
		lines = append(lines,
			"Windows: 请安装 FFmpeg 5.1 系列完整构建，并把 `bin` 目录加入 PATH。",
			"示例: 安装后执行 `ffmpeg -version` 和 `ffprobe -version` 确认版本与 PATH 生效。",
		)
	case "linux":
		lines = append(lines,
			"Linux: Ubuntu 22.04 / Debian 12 可使用系统包安装 FFmpeg 5.1 LTS。",
			"示例: sudo apt update && sudo apt install -y ffmpeg",
		)
	default:
		lines = append(lines, "其他平台: 请自行安装 FFmpeg/FFprobe >= 5.1，并确认 `ffmpeg` 与 `ffprobe` 已加入 PATH。")
	}
	if len(requiredMissing) == 0 && len(optionalMissing) > 0 {
		lines = append(lines, "当前必需工具已满足；可选能力是否安装取决于你要运行的本地方法。")
	}
	return lines
}

func stringSet(values []string) map[string]struct{} {
	result := make(map[string]struct{}, len(values))
	for _, value := range values {
		result[value] = struct{}{}
	}
	return result
}

func containsDependency(deps []string, target string) bool {
	for _, dep := range deps {
		if dep == target {
			return true
		}
	}
	return false
}
