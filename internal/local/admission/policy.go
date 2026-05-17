package admission

import "fmt"

type ToolPolicy struct {
	Name                string
	Binary              string
	Required            bool
	Version             string
	License             string
	CommercialUse       bool
	LicenseURL          string
	Role                string
	Boundary            string
	ApprovalRequiredFor string
}

func CurrentPolicies() []ToolPolicy {
	versionAnchors := map[string]string{
		"libass":        "0.17.3",
		"libfreetype":   "2.13.3",
		"libfontconfig": "2.15.0",
		"libfribidi":    "1.0.16",
		"libharfbuzz":   "10.2.0",
	}
	policies := []ToolPolicy{
		{
			Name:                "FFmpeg >= 5.1",
			Binary:              "ffmpeg",
			Required:            true,
			Version:             ">= 5.1",
			License:             "LGPL v2.1 or later",
			CommercialUse:       true,
			LicenseURL:          "https://ffmpeg.org/legal.html",
			Role:                "本地音视频处理执行引擎",
			Boundary:            "仅通过 os/exec 调用，不静态链接、不动态链接、不修改源码；默认不启用 GPL 模式且不引入 non-free 组件",
			ApprovalRequiredFor: "新增编译依赖、启用 GPL、引入 non-free 组件",
		},
		{
			Name:                "FFprobe >= 5.1",
			Binary:              "ffprobe",
			Required:            true,
			Version:             ">= 5.1",
			License:             "LGPL v2.1 or later",
			CommercialUse:       true,
			LicenseURL:          "https://ffmpeg.org/legal.html",
			Role:                "本地音视频信息查询工具",
			Boundary:            "仅通过 os/exec 调用，不静态链接、不动态链接、不修改源码",
			ApprovalRequiredFor: "替换媒体探测工具或新增外部探测依赖",
		},
	}
	for _, dep := range AllowedDependencyNames()[2:] {
		version := "FFmpeg >= 5.1 build feature"
		if anchor, ok := versionAnchors[dep]; ok {
			version = fmt.Sprintf("FFmpeg >= 5.1 build feature; approved anchor %s", anchor)
		}
		policies = append(policies, ToolPolicy{
			Name:                dep,
			Binary:              dep,
			Required:            false,
			Version:             version,
			License:             "system package / upstream license",
			CommercialUse:       true,
			Role:                "FFmpeg 本地能力白名单项",
			Boundary:            "仅作为 FFmpeg 能力检查项，不链接进 Go 二进制",
			ApprovalRequiredFor: "新增或替换本地能力依赖",
		})
	}
	return policies
}

func AllowedDependencyNames() []string {
	return []string{
		"ffmpeg",
		"ffprobe",
		"openh264",
		"demuxer",
		"libmp3lame",
		"libass",
		"libfreetype",
		"libfontconfig",
		"libfribidi",
		"libharfbuzz",
		"zlib",
		"libpng",
		"libjpeg-turbo",
	}
}

func IsAllowedDependency(name string) bool {
	for _, allowed := range AllowedDependencyNames() {
		if name == allowed {
			return true
		}
	}
	return false
}

func FindPolicy(binary string) (ToolPolicy, bool) {
	for _, policy := range CurrentPolicies() {
		if policy.Binary == binary {
			return policy, true
		}
	}
	return ToolPolicy{}, false
}

func ApprovalTemplate() []string {
	return []string{
		"组件",
		"版本",
		"License",
		"是否可商用",
		"License 链接",
		"用途",
		"接入边界",
	}
}

func ApprovalPrompt() string {
	return "新增 local 工具或新增 FFmpeg 编译依赖前，必须先向用户提交组件清单并获得确认。"
}

func PolicyLine(policy ToolPolicy) string {
	return fmt.Sprintf(
		"- %s (%s): required=%t, version=%s, license=%s, commercial=%t",
		policy.Name,
		policy.Binary,
		policy.Required,
		policy.Version,
		policy.License,
		policy.CommercialUse,
	)
}
