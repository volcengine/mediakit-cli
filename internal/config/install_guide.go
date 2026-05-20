package config

import (
	"fmt"
	"runtime"
	"strings"
)

// InstallGuide 返回当前平台的依赖安装指引
func InstallGuide(missingDeps []string) map[string]any {
	guide := map[string]any{
		"platform": runtime.GOOS,
	}

	switch runtime.GOOS {
	case "darwin":
		guide["commands"] = []string{"brew install ffmpeg --with-openh264"}
		guide["note"] = "--with-openh264 会触发源码编译，自动启用 libass/freetype/fontconfig/harfbuzz/libmp3lame 等默认依赖。如不需要 openh264 可直接: brew install ffmpeg"
	case "linux":
		guide["commands"] = []string{
			"sudo apt update && sudo apt install -y ffmpeg",
		}
		guide["note"] = "Ubuntu/Debian 系统包通常含全量编解码器。RHEL/Fedora 需先启用 RPM Fusion: sudo dnf install -y https://download1.rpmfusion.org/free/fedora/rpmfusion-free-release-$(rpm -E %fedora).noarch.rpm && sudo dnf install -y ffmpeg"
	case "windows":
		guide["commands"] = []string{"choco install ffmpeg-full"}
		guide["note"] = "或使用 Scoop: scoop install ffmpeg。安装后确认 ffmpeg 和 ffprobe 在 PATH 中。"
	default:
		guide["commands"] = []string{}
		guide["note"] = "请自行安装 FFmpeg/FFprobe >= 5.1，确认已加入 PATH。"
	}

	if len(missingDeps) > 0 {
		guide["missing"] = missingDeps
	}

	return guide
}

// InstallGuideText 返回纯文本格式的安装指引（用于 init 等交互场景）
func InstallGuideText(missingDeps []string) string {
	var lines []string
	lines = append(lines, fmt.Sprintf("缺失依赖: %s", strings.Join(missingDeps, ", ")))
	lines = append(lines, fmt.Sprintf("当前平台: %s", runtime.GOOS))
	lines = append(lines, "")

	switch runtime.GOOS {
	case "darwin":
		lines = append(lines,
			"安装命令（推荐全量）:",
			"  brew install ffmpeg --with-openh264",
			"",
			"说明: --with-openh264 触发源码编译，自动启用 libass/freetype/fontconfig/harfbuzz/libmp3lame 等默认依赖。",
			"如不需要 openh264: brew install ffmpeg",
		)
	case "linux":
		lines = append(lines,
			"安装命令:",
			"  sudo apt update && sudo apt install -y ffmpeg",
			"",
			"RHEL/Fedora:",
			"  sudo dnf install -y https://download1.rpmfusion.org/free/fedora/rpmfusion-free-release-$(rpm -E %fedora).noarch.rpm",
			"  sudo dnf install -y ffmpeg",
		)
	case "windows":
		lines = append(lines,
			"安装命令:",
			"  choco install ffmpeg-full",
			"",
			"或使用 Scoop: scoop install ffmpeg",
		)
	default:
		lines = append(lines, "请自行安装 FFmpeg/FFprobe >= 5.1，确认已加入 PATH。")
	}

	lines = append(lines, "", "安装后执行 `ffmpeg -version` 和 `ffprobe -version` 验证。")
	return strings.Join(lines, "\n")
}
