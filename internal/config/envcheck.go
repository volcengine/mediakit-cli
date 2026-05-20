package config

import (
	"fmt"
	"os/exec"
	"strings"

	"mediakit-cli/internal/local/admission"
)

func RefreshEnvCache(home string) (EnvCache, error) {
	resolved, err := ResolveConfig(home)
	if err != nil {
		return EnvCache{}, err
	}

	cache := NewEnvCache(resolved.Mode)
	cache.CloudReady = resolved.APIKey != ""

	ffmpegStatus := probeFFmpegVersion()
	ffprobeStatus := probeFFprobeVersion()
	cache.Tools["ffmpeg"] = ffmpegStatus
	cache.Tools["ffprobe"] = ffprobeStatus
	for _, dep := range admission.AllowedDependencyNames()[2:] {
		cache.Tools[dep] = probeAllowedFFmpegDependency(ffmpegStatus.Available, dep)
	}

	cache.LocalReady = ffmpegStatus.Available && ffprobeStatus.Available

	if err := SaveEnvCache(home, cache); err != nil {
		return EnvCache{}, err
	}
	return cache, nil
}

func MissingRequiredLocalDependencies(cache EnvCache) []string {
	missing := []string{}
	for _, policy := range admission.CurrentPolicies() {
		if !policy.Required {
			continue
		}
		status, ok := cache.Tools[policy.Binary]
		if !ok || !status.Available {
			missing = append(missing, policy.Binary)
		}
	}
	return missing
}

func probeFFmpegVersion() ToolStatus {
	return probeFFmpegSuiteVersion("ffmpeg")
}

func probeFFprobeVersion() ToolStatus {
	return probeFFmpegSuiteVersion("ffprobe")
}

func probeFFmpegSuiteVersion(binary string) ToolStatus {
	path, err := exec.LookPath(binary)
	if err != nil {
		return ToolStatus{Available: false, Reason: "not found"}
	}

	cmd := exec.Command(path, "-version")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return ToolStatus{Available: false, Reason: strings.TrimSpace(string(output))}
	}

	firstLine := firstNonEmptyLine(string(output))
	if !isFFmpegSuiteAtLeast51(firstLine, binary) {
		return ToolStatus{
			Available: false,
			Version:   firstLine,
			Reason:    fmt.Sprintf("requires %s >= 5.1, got %s", binary, firstLine),
		}
	}
	return ToolStatus{Available: true, Version: firstLine}
}

func probeAllowedFFmpegDependency(ffmpegAvailable bool, dep string) ToolStatus {
	if !ffmpegAvailable {
		return ToolStatus{Available: false, Reason: "ffmpeg >= 5.1 not available"}
	}
	switch dep {
	case "openh264":
		openh264Status := probeFFmpegOutput(dep, []string{"-hide_banner", "-encoders"}, "libopenh264")
		if openh264Status.Available {
			return openh264Status
		}
		videotoolboxStatus := probeFFmpegOutput("h264_videotoolbox", []string{"-hide_banner", "-encoders"}, "h264_videotoolbox")
		if videotoolboxStatus.Available {
			return ToolStatus{Available: true, Version: "detected via h264_videotoolbox"}
		}
		return ToolStatus{Available: false, Reason: "neither libopenh264 nor h264_videotoolbox detected"}
	case "h264_videotoolbox":
		return probeFFmpegOutput(dep, []string{"-hide_banner", "-encoders"}, "h264_videotoolbox")
	case "demuxer":
		return probeFFmpegOutput(dep, []string{"-hide_banner", "-demuxers"}, " concat")
	case "libmp3lame":
		return probeFFmpegOutput(dep, []string{"-hide_banner", "-encoders"}, "libmp3lame")
	case "libass":
		return probeFFmpegOutput(dep, []string{"-hide_banner", "-filters"}, " ass")
	case "libpng":
		return probeFFmpegOutput(dep, []string{"-hide_banner", "-decoders"}, " png")
	case "libjpeg-turbo":
		return probeFFmpegOutput(dep, []string{"-hide_banner", "-decoders"}, " mjpeg")
	case "libfreetype", "libfontconfig", "libfribidi", "libharfbuzz", "zlib":
		return probeFFmpegBuildConfig(dep)
	default:
		return ToolStatus{Available: false, Reason: "dependency is not in local allowlist"}
	}
}

func probeFFmpegOutput(dep string, args []string, needle string) ToolStatus {
	cmd := exec.Command("ffmpeg", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return ToolStatus{Available: false, Reason: strings.TrimSpace(string(output))}
	}

	text := strings.ToLower(string(output))
	if strings.Contains(text, strings.ToLower(needle)) {
		return ToolStatus{Available: true, Version: "detected"}
	}
	return ToolStatus{Available: false, Reason: dep + " not detected"}
}

func probeFFmpegBuildConfig(dep string) ToolStatus {
	cmd := exec.Command("ffmpeg", "-version")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return ToolStatus{Available: false, Reason: strings.TrimSpace(string(output))}
	}
	text := strings.ToLower(string(output))
	needles := []string{dep}
	if dep == "libjpeg-turbo" {
		needles = []string{"libjpeg", "jpeg"}
	}
	for _, needle := range needles {
		if strings.Contains(text, strings.ToLower(needle)) {
			return ToolStatus{Available: true, Version: "detected"}
		}
	}
	if status, ok := probePkgConfigDependency(dep); ok {
		return status
	}
	return ToolStatus{Available: false, Reason: dep + " not detected"}
}

func probePkgConfigDependency(dep string) (ToolStatus, bool) {
	packages := map[string]string{
		"libass":        "libass",
		"libfreetype":   "freetype2",
		"libfontconfig": "fontconfig",
		"libfribidi":    "fribidi",
		"libharfbuzz":   "harfbuzz",
		"zlib":          "zlib",
	}
	pkg, ok := packages[dep]
	if !ok {
		return ToolStatus{}, false
	}
	path, err := exec.LookPath("pkg-config")
	if err != nil {
		return ToolStatus{}, false
	}
	cmd := exec.Command(path, "--modversion", pkg)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return ToolStatus{}, false
	}
	version := strings.TrimSpace(string(output))
	if version == "" {
		version = "detected"
	}
	return ToolStatus{Available: true, Version: version}, true
}

func isFFmpegSuiteAtLeast51(line string, binary string) bool {
	line = strings.ToLower(strings.TrimSpace(line))
	binary = strings.ToLower(strings.TrimSpace(binary))
	prefix := binary + " version "
	index := strings.Index(line, prefix)
	if index < 0 {
		return false
	}
	versionText := strings.TrimPrefix(line[index:], prefix)
	versionText = strings.TrimPrefix(versionText, "n")
	fields := strings.Fields(versionText)
	if len(fields) == 0 {
		return false
	}
	parts := strings.Split(fields[0], ".")
	if len(parts) < 2 {
		return false
	}
	major := parseLeadingInt(parts[0])
	minor := parseLeadingInt(parts[1])
	return major > 5 || (major == 5 && minor >= 1)
}

func parseLeadingInt(value string) int {
	result := 0
	for _, r := range value {
		if r < '0' || r > '9' {
			break
		}
		result = result*10 + int(r-'0')
	}
	return result
}

func firstNonEmptyLine(text string) string {
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			return line
		}
	}
	return ""
}
