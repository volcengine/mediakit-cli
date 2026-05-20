package commands

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	cliconfig "mediakit-cli/internal/config"
	"mediakit-cli/internal/local"
	"mediakit-cli/internal/modes"
)

type DomainMeta struct {
	Name        string
	Description string
}

type ParamMeta struct {
	Name         string
	FlagName     string
	Description  string
	Type         string
	ItemType     string
	Required     bool
	Enum         []string
	HasDefault   bool
	DefaultValue string
	JSONEncoded  bool
}

type CapabilityMeta struct {
	Name              string
	DisplayName       string
	Domain            string
	Description       string
	ModeLabel         string
	CloudOnly         bool
	LocalSupported    bool
	LocalSource       string
	LocalDeps         []string
	LocalLimitations  []string
	Async             bool
	AsyncQueryCommand string
	OutputType        string // "video", "audio", "file", "" (cloud-only 无 local 输出)
	Params            []ParamMeta
}

var generatedDomains = []DomainMeta{
	{Name: "video", Description: "视频处理，涵盖视频画质增强、视频理解、字幕擦除等能力"},
	{Name: "editing", Description: "音视频剪辑，涵盖音视频拼接、裁剪、合成等能力"},
	{Name: "shared", Description: "通用能力与任务查询"},
}

var generatedCapabilities = []CapabilityMeta{
	{
		Name:              "erase-video-subtitle-pro",
		DisplayName:       "精细化字幕擦除",
		Domain:            "video",
		Description:       "针对视频中的字幕，实现高质量的无痕擦除，最大程度的还原视频画面。\n支持格式：主流视频格式如mp4、flv、ts、avi、mov、wmv、mkv。",
		LocalSupported:    false,
		Async:             true,
		AsyncQueryCommand: "query-task",
		OutputType:        "video",
		Params: []ParamMeta{
			{
				Name:        "video_url",
				FlagName:    "video-url",
				Description: "输入视频。String 类型，支持http://xxx或https://xxx格式 URL",
				Type:        "string",
				Required:    true,
				HasDefault:  false,
				JSONEncoded: false,
			},
			{
				Name:         "mode",
				FlagName:     "mode",
				Description: "字幕擦除模式，取值如下：Subtitle：擦除OCR检测为字幕的文本。在此模式下，系统将启用 OCR 识别，并依据检测结果进行擦除操作，仅擦除下面50%画面的字幕。 Text：擦除OCR检测为字幕及其他的文本（如人物介绍等），不包含场景文字（如宫殿门牌匾等）。",
				Type:         "string",
				Required:     false,
				Enum:         []string{"Subtitle", "Text"},
				HasDefault:   true,
				DefaultValue: "Subtitle",
				JSONEncoded:  false,
			},
			{
				Name:         "output_encode_mode",
				FlagName:     "output-encode-mode",
				Description: "输出视频编码模式，支持以下两种取值：Quality（默认值）：画质优先模式。此模式下，系统会采用较高的目标码率进行编码，以确保高画质。这通常会导致输出文件的码率显著高于源文件，文件体积也相应增大。 Size：大小优先模式。在保证一定画质的前提下，使输出码率尽量向源视频码率对齐。",
				Type:         "string",
				Required:     false,
				Enum:         []string{"Quality", "Size"},
				HasDefault:   true,
				DefaultValue: "Quality",
				JSONEncoded:  false,
			},
			{
				Name:        "erase_ratio_location",
				FlagName:    "erase-ratio-location",
				Description: "擦除框数组。添加擦除框后，系统仅擦除框内文本。\n子项说明：擦除框位置信息\n子字段说明（JSON 数组每项）:\n- top_left_x: 框选区域左上角相对于视频左上角在X轴上的偏移比例，取值范围为[0,1]，其中 0 表示无偏移（与视频左边缘对齐），1 表示完全偏移（与视频右边缘对齐）。必填\n- top_left_y: 框选区域左上角相对于视频左上角在 Y 轴上的偏移比例，取值范围为 [0,1]，其中 0 表示无偏移（与视频上边缘对齐），1 表示完全偏移（与视频下边缘对齐）。必填\n- bottom_right_x: 框选区域右下角相对于视频左上角在 X 轴上的偏移比例，取值范围为 [0,1]，其中 0 表示无偏移（与视频左边缘对齐），1 表示完全偏移（与视频右边缘对齐）。必填\n- bottom_right_y: 框选区域右下角相对于视频左上角在 Y 轴上的偏移比例，取值范围为 [0,1]，其中 0 表示无偏移（与视频上边缘对齐），1 表示完全偏移（与视频下边缘对齐）。必填",
				Type:        "array",
				ItemType:    "object",
				Required:    false,
				HasDefault:  false,
				JSONEncoded: true,
			},
			{
				Name:        "callback_args",
				FlagName:    "callback-args",
				Description: "可选，回调参数",
				Type:        "string",
				Required:    false,
				HasDefault:  false,
				JSONEncoded: false,
			},
			{
				Name:        "client_token",
				FlagName:    "client-token",
				Description: "可选，用于幂等，默认幂等，用户可根据需求进行调整",
				Type:        "string",
				Required:    false,
				HasDefault:  false,
				JSONEncoded: false,
			},
		},
	},
	{
		Name:              "enhance-video",
		DisplayName:       "画质增强",
		Domain:            "video",
		Description:       "画质增强：针对 AIGC / UGC / 短剧 / 教育 / 游戏 / 老片修复等场景，提供画质提升 + 超分增强一站式解决方案。依托 AI MediaKit 智能媒体处理引擎，融合视频内容理解、画质指标智能决策、多维度增强原子算法，实现画质的全面优化。\n支持格式：主流视频格式如mp4、flv、ts、avi、mov、wmv、mkv。\n使用限制：单文件大小不超过100G。",
		LocalSupported:    false,
		Async:             true,
		AsyncQueryCommand: "query-task",
		OutputType:        "video",
		Params: []ParamMeta{
			{
				Name:        "video_url",
				FlagName:    "video-url",
				Description: "输入视频。String 类型，支持http://xxx或https://xxx格式 URL",
				Type:        "string",
				Required:    true,
				HasDefault:  false,
				JSONEncoded: false,
			},
			{
				Name:         "scene",
				FlagName:     "scene",
				Description: "场景化模板类型。用于选择一个针对特定业务场景的预设画质增强模板。支持的取值如下：common（默认值）: 通用模板；ugc: UGC 短视频；short_series: 短剧；aigc: AIGC 内容；old_film: 老片修复",
				Type:         "string",
				Required:     false,
				Enum:         []string{"common", "ugc", "short_series", "aigc", "old_film"},
				HasDefault:   true,
				DefaultValue: "common",
				JSONEncoded:  false,
			},
			{
				Name:         "tool_version",
				FlagName:     "tool-version",
				Description: "工具版本，标准版:standard，专业版：professional，默认为标准版",
				Type:         "string",
				Required:     false,
				Enum:         []string{"standard", "professional"},
				HasDefault:   true,
				DefaultValue: "standard",
				JSONEncoded:  false,
			},
			{
				Name:        "resolution",
				FlagName:    "resolution",
				Description: "目标分辨率。支持的取值如下所示。配置此参数后，不可同时配置resolution_limit字段",
				Type:        "string",
				Required:    false,
				Enum:        []string{"240p", "360p", "480p", "540p", "720p", "1080p", "2k", "4k"},
				HasDefault:  false,
				JSONEncoded: false,
			},
			{
				Name:        "resolution_limit",
				FlagName:    "resolution-limit",
				Description: "目标长宽限制，用于指定输出视频的长边或短边的最大像素值，取值范围为 [64, 2160]。配置此参数后，不可同时配置resolution字段",
				Type:        "integer",
				Required:    false,
				HasDefault:  false,
				JSONEncoded: false,
			},
			{
				Name:        "fps",
				FlagName:    "fps",
				Description: "目标帧率，单位为 fps。取值范围为 (0, 120]。",
				Type:        "number",
				Required:    false,
				HasDefault:  false,
				JSONEncoded: false,
			},
			{
				Name:        "callback_args",
				FlagName:    "callback-args",
				Description: "可选，回调参数",
				Type:        "string",
				Required:    false,
				HasDefault:  false,
				JSONEncoded: false,
			},
			{
				Name:        "client_token",
				FlagName:    "client-token",
				Description: "可选，用于幂等，默认幂等，用户可根据需求进行调整",
				Type:        "string",
				Required:    false,
				HasDefault:  false,
				JSONEncoded: false,
			},
		},
	},
	{
		Name:              "image-to-video",
		DisplayName:       "图片转视频",
		Domain:            "editing",
		Description:       "多张图片生成动画视频。",
		LocalSupported:    false,
		Async:             true,
		AsyncQueryCommand: "query-task",
		OutputType:        "video",
		Params: []ParamMeta{
			{
				Name:        "images",
				FlagName:    "images",
				Description: "待合成的图片列表，Array<Image>类型。最少传入1个，最多传入100个\n子项说明：待合成的图片。Image类型\n子字段说明（JSON 数组每项）:\n- image_url: 输入图片。String 类型，支持http://xxx或https://xxx格式 URL，必填\n- duration: 图片播放时长，选填，默认值：3，单位：秒，支持 2 位小数。可选\n- animation_type: 图片的动画类型，选填，不填时无动画效果。 move_up：向上移动 move_down：向下移动 move_left：向左移动 move_right：向右移动 zoom_in：缩小 zoom_out：放大，可选\n- animation_in: 动画结束时间，选填，支持2位小数。默认为图片展示时长，表示动画随图片播放同时结束，单位：秒，可选\n- animation_out: 动画结束时间，选填，支持2位小数。默认为图片展示时长，表示动画随图片播放同时结束，单位：秒，可选",
				Type:        "array",
				ItemType:    "object",
				Required:    true,
				HasDefault:  false,
				JSONEncoded: true,
			},
			{
				Name:        "transitions",
				FlagName:    "transitions",
				Description: "转场效果 ID，Array<string> 类型。如果不提供，则没有转场。当视频数量超过转场数量 2 个及以上时，系统将自动循环使用转场。例如有 10 个视频，2 种转场效果，那么在 9 处拼接点上，这 2 种转场效果将被依次循环使用。\n子项说明：转场效果 ID\n分类：交替出场，ID：1182359\n分类：旋转放大，ID：1182360\n分类：泛开，ID：1182358\n分类：六角形，ID：1182365\n分类：故障转换，ID：1182367\n分类：飞眼，ID：1182368\n分类：梦幻放大，ID：1182369\n分类：开门展现，ID：1182370\n分类：立方转换，ID：1182373\n分类：透镜变换，ID：1182374\n分类：晚霞转场，ID：1182375\n分类：圆形交替，ID：1182378",
				Type:        "array",
				ItemType:    "string",
				Required:    false,
				Enum:        []string{"1182359", "1182360", "1182358", "1182365", "1182367", "1182368", "1182369", "1182370", "1182373", "1182374", "1182375", "1182378"},
				HasDefault:  false,
				JSONEncoded: false,
			},
			{
				Name:        "callback_args",
				FlagName:    "callback-args",
				Description: "可选，回调参数",
				Type:        "string",
				Required:    false,
				HasDefault:  false,
				JSONEncoded: false,
			},
			{
				Name:        "client_token",
				FlagName:    "client-token",
				Description: "可选，用于幂等，默认幂等，用户可根据需求进行调整",
				Type:        "string",
				Required:    false,
				HasDefault:  false,
				JSONEncoded: false,
			},
		},
	},
	{
		Name:              "extract-audio",
		DisplayName:       "提取音频",
		Domain:            "editing",
		Description:       "将视频文件中的音频流分离并保存为独立的音频文件。",
		LocalSupported:    true,
		LocalSource:       "generated",
		LocalDeps:         []string{"ffmpeg", "libmp3lame"},
		LocalLimitations:  []string{"支持本地 FFmpeg 提取音频；callback/client_token 本地忽略。"},
		Async:             true,
		AsyncQueryCommand: "query-task",
		OutputType:        "audio",
		Params: []ParamMeta{
			{
				Name:        "video_url",
				FlagName:    "video-url",
				Description: "输入视频，String 类型，支持http://xxx或https://xxx格式 URL",
				Type:        "string",
				Required:    true,
				HasDefault:  false,
				JSONEncoded: false,
			},
			{
				Name:         "format",
				FlagName:     "format",
				Description: "输出音频的格式，支持 mp3、m4a 格式。 默认m4a",
				Type:         "string",
				Required:     false,
				Enum:         []string{"mp3", "m4a"},
				HasDefault:   true,
				DefaultValue: "m4a",
				JSONEncoded:  false,
			},
			{
				Name:        "callback_args",
				FlagName:    "callback-args",
				Description: "可选，回调参数",
				Type:        "string",
				Required:    false,
				HasDefault:  false,
				JSONEncoded: false,
			},
			{
				Name:        "client_token",
				FlagName:    "client-token",
				Description: "可选，用于幂等，默认幂等，用户可根据需求进行调整",
				Type:        "string",
				Required:    false,
				HasDefault:  false,
				JSONEncoded: false,
			},
		},
	},
	{
		Name:              "add-image-to-video",
		DisplayName:       "视频加图片",
		Domain:            "editing",
		Description:       "视频加图片，可用作加图片水印。",
		LocalSupported:    true,
		LocalSource:       "generated",
		LocalDeps:         []string{"ffmpeg", "openh264", "libpng"},
		LocalLimitations:  []string{"支持本地 FFmpeg 叠加图片；callback/client_token 本地忽略。"},
		Async:             true,
		AsyncQueryCommand: "query-task",
		OutputType:        "video",
		Params: []ParamMeta{
			{
				Name:        "video_url",
				FlagName:    "video-url",
				Description: "输入视频。String 类型，支持http://xxx或https://xxx格式 URL",
				Type:        "string",
				Required:    true,
				HasDefault:  false,
				JSONEncoded: false,
			},
			{
				Name:        "sub_image_url",
				FlagName:    "sub-image-url",
				Description: "图片URL。支持http://xxx或https://xxx格式 URL",
				Type:        "string",
				Required:    true,
				HasDefault:  false,
				JSONEncoded: false,
			},
			{
				Name:         "sub_image_height",
				FlagName:     "sub-image-height",
				Description: "图片的高度，字符串类型，支持具体像素值（如 '100'）或百分比（如 '20%'，相对于视频高度）。",
				Type:         "string",
				Required:     false,
				HasDefault:   true,
				DefaultValue: "5%",
				JSONEncoded:  false,
			},
			{
				Name:         "sub_image_width",
				FlagName:     "sub-image-width",
				Description: "图片的宽度，字符串类型，支持具体像素值（如 '100'）或百分比（如 '20%'，相对于视频高度）。",
				Type:         "string",
				Required:     false,
				HasDefault:   true,
				DefaultValue: "10%",
				JSONEncoded:  false,
			},
			{
				Name:         "sub_image_pos_x",
				FlagName:     "sub-image-pos-x",
				Description: "图片在水平方向（X 轴）的位置，以视频左上角为原点，字符串类型，支持具体像素值（如 '100'）或百分比（如 '20%'）。例如值为 '0' 时，表示处于最左侧。",
				Type:         "string",
				Required:     false,
				HasDefault:   true,
				DefaultValue: "85%",
				JSONEncoded:  false,
			},
			{
				Name:         "sub_image_pos_y",
				FlagName:     "sub-image-pos-y",
				Description: "图片在垂直方向（Y 轴）的位置，以视频左上角为原点，字符串类型，支持具体像素值（如 '100'）或百分比（如 '20%'）。例如值为 '0' 时，表示处于最上侧。",
				Type:         "string",
				Required:     false,
				HasDefault:   true,
				DefaultValue: "90%",
				JSONEncoded:  false,
			},
			{
				Name:        "start_time",
				FlagName:    "start-time",
				Description: "图片的开始时间，单位：秒。不传默认同视频开始时间",
				Type:        "number",
				Required:    false,
				HasDefault:  false,
				JSONEncoded: false,
			},
			{
				Name:        "end_time",
				FlagName:    "end-time",
				Description: "图片的结束时间，单位：秒。注意：如果设置的开始/结束时间超出原始视频时长，输出视频长度将以该结束时间为准，超出部分以黑屏形式延续。不传默认同视频结束时间",
				Type:        "number",
				Required:    false,
				HasDefault:  false,
				JSONEncoded: false,
			},
			{
				Name:        "callback_args",
				FlagName:    "callback-args",
				Description: "可选，回调参数",
				Type:        "string",
				Required:    false,
				HasDefault:  false,
				JSONEncoded: false,
			},
			{
				Name:        "client_token",
				FlagName:    "client-token",
				Description: "可选，用于幂等，默认幂等，用户可根据需求进行调整",
				Type:        "string",
				Required:    false,
				HasDefault:  false,
				JSONEncoded: false,
			},
		},
	},
	{
		Name:              "add-subtitle-to-video",
		DisplayName:       "视频加字幕",
		Domain:            "editing",
		Description:       "将字幕文件或文本内容，以指定样式压制到视频画面中，生成带内嵌字幕的新视频。",
		LocalSupported:    true,
		LocalSource:       "generated",
		LocalDeps:         []string{"ffmpeg", "openh264", "libass", "libfreetype", "libfontconfig", "libfribidi", "libharfbuzz"},
		LocalLimitations:  []string{"支持本地 FFmpeg 字幕烧录；callback/client_token 本地忽略。"},
		Async:             true,
		AsyncQueryCommand: "query-task",
		OutputType:        "video",
		Params: []ParamMeta{
			{
				Name:        "video_url",
				FlagName:    "video-url",
				Description: "输入视频。String 类型，支持http://xxx或https://xxx格式 URL",
				Type:        "string",
				Required:    true,
				HasDefault:  false,
				JSONEncoded: false,
			},
			{
				Name:        "subtitle_url",
				FlagName:    "subtitle-url",
				Description: "字幕文件 URL、filename。常见的字幕文件为 SRT、VTT、ASS 等格式。",
				Type:        "string",
				Required:    false,
				HasDefault:  false,
				JSONEncoded: false,
			},
			{
				Name:        "subtitles",
				FlagName:    "subtitles",
				Description: "字幕列表，Array<object>类型。\n子字段说明（JSON 数组每项）:\n- subtitle_text: 字幕文本，必填\n- start_time: 字幕开始时间。单位：秒。必填\n- end_time: 字幕结束时间。单位：秒。必填",
				Type:        "array",
				ItemType:    "object",
				Required:    false,
				HasDefault:  false,
				JSONEncoded: true,
			},
			{
				Name:         "subtitle_pos_preset",
				FlagName:     "subtitle-pos-preset",
				Description: "预设字幕位置。底部居中（默认常用） bottom_center；顶部居中 top_center；画面正中央 center；偏下三分之一处 lower_third",
				Type:         "string",
				Required:     false,
				Enum:         []string{"bottom_center", "top_center", "center", "lower_third"},
				HasDefault:   true,
				DefaultValue: "bottom_center",
				JSONEncoded:  false,
			},
			{
				Name:         "subtitle_font_size",
				FlagName:     "subtitle-font-size",
				Description: "字幕的字体大小，单位：像素。",
				Type:         "integer",
				Required:     false,
				HasDefault:   true,
				DefaultValue: "50",
				JSONEncoded:  false,
			},
			{
				Name:         "subtitle_font_color",
				FlagName:     "subtitle-font-color",
				Description: "字幕的字体颜色，RGBA 格式。默认#FFFFFFFF",
				Type:         "string",
				Required:     false,
				HasDefault:   true,
				DefaultValue: "#FFFFFFFF",
				JSONEncoded:  false,
			},
			{
				Name:         "subtitle_font_type",
				FlagName:     "subtitle-font-type",
				Description: "字幕的字体 ID。 思源黑体：sy_black （经典无衬线黑体，端正百搭，正文首选） 庞门正道标题体：pm_zhengdao （粗壮有力，硬汉气场，大标题/封面神器） 阿里巴巴普惠体：ali_puhui （现代感极强，结构饱满，屏幕阅读体验极佳） 站酷快乐体：zhanku_kuaile （圆润活泼，带手写感，适合轻松搞笑的 Vlog 氛围）",
				Type:         "string",
				Required:     false,
				Enum:         []string{"sy_black", "pm_zhengdao", "ali_puhui", "zhanku_kuaile"},
				HasDefault:   true,
				DefaultValue: "sy_black",
				JSONEncoded:  false,
			},
			{
				Name:        "callback_args",
				FlagName:    "callback-args",
				Description: "可选，回调参数",
				Type:        "string",
				Required:    false,
				HasDefault:  false,
				JSONEncoded: false,
			},
			{
				Name:        "client_token",
				FlagName:    "client-token",
				Description: "可选，用于幂等，默认幂等，用户可根据需求进行调整",
				Type:        "string",
				Required:    false,
				HasDefault:  false,
				JSONEncoded: false,
			},
		},
	},
	{
		Name:              "mux-audio-video",
		DisplayName:       "视频加音频",
		Domain:            "editing",
		Description:       "音视频合成。",
		LocalSupported:    true,
		LocalSource:       "generated",
		LocalDeps:         []string{"ffmpeg"},
		LocalLimitations:  []string{"支持本地 FFmpeg 音视频合并；callback/client_token 本地忽略。"},
		Async:             true,
		AsyncQueryCommand: "query-task",
		OutputType:        "video",
		Params: []ParamMeta{
			{
				Name:        "video_url",
				FlagName:    "video-url",
				Description: "输入视频。String 类型，支持http://xxx或https://xxx格式 URL",
				Type:        "string",
				Required:    true,
				HasDefault:  false,
				JSONEncoded: false,
			},
			{
				Name:        "audio_url",
				FlagName:    "audio-url",
				Description: "输入音频。String 类型，支持http://xxx或https://xxx格式 URL",
				Type:        "string",
				Required:    true,
				HasDefault:  false,
				JSONEncoded: false,
			},
			{
				Name:         "is_audio_reserve",
				FlagName:     "is-audio-reserve",
				Description: "Boolean 类型，是否保留原视频流中的音频。默认值 true：保留。false：不保留。",
				Type:         "boolean",
				Required:     false,
				HasDefault:   true,
				DefaultValue: "true",
				JSONEncoded:  false,
			},
			{
				Name:         "is_video_audio_sync",
				FlagName:     "is-video-audio-sync",
				Description: "Boolean 类型，是否对齐音频和视频时长。 true：通过 output_sync 配置，对齐音频和视频时长。 false（默认值）：保持原样输出，不做音视频对齐。最终合成的视频时长，以较长的流为准。",
				Type:         "boolean",
				Required:     false,
				HasDefault:   true,
				DefaultValue: "false",
				JSONEncoded:  false,
			},
			{
				Name:         "sync_mode",
				FlagName:     "sync-mode",
				Description: "String 类型，设置 is_video_audio_sync 为 true 时生效；当音频和视频时长不相等时，可指定对齐基准，可选项：video、audio。 video：【默认值】以视频的时长为准。 audio：以音频的时长为准。",
				Type:         "string",
				Required:     false,
				Enum:         []string{"video", "audio"},
				HasDefault:   true,
				DefaultValue: "video",
				JSONEncoded:  false,
			},
			{
				Name:         "sync_method",
				FlagName:     "sync-method",
				Description: "String 类型，设置 is_video_audio_sync 为 true 时生效；指定对齐方式，支持通过裁剪或加速的方式，对齐音频和视频的时长。可选项：speed、trim。 speed：通过加快音频或视频的速度，对齐音频和视频的时长。 trim：【默认值】通过裁剪音频或视频，对齐音频和视频的时长。从头开始计算并裁剪。",
				Type:         "string",
				Required:     false,
				Enum:         []string{"speed", "trim"},
				HasDefault:   true,
				DefaultValue: "trim",
				JSONEncoded:  false,
			},
			{
				Name:        "callback_args",
				FlagName:    "callback-args",
				Description: "可选，回调参数",
				Type:        "string",
				Required:    false,
				HasDefault:  false,
				JSONEncoded: false,
			},
			{
				Name:        "client_token",
				FlagName:    "client-token",
				Description: "可选，用于幂等，默认幂等，用户可根据需求进行调整",
				Type:        "string",
				Required:    false,
				HasDefault:  false,
				JSONEncoded: false,
			},
		},
	},
	{
		Name:              "concat-video",
		DisplayName:       "视频拼接",
		Domain:            "editing",
		Description:       "拼接多个视频片段，支持添加转场效果。",
		LocalSupported:    true,
		LocalSource:       "generated",
		LocalDeps:         []string{"ffmpeg", "demuxer"},
		LocalLimitations:  []string{"支持本地 FFmpeg 无转场拼接；transitions 需使用 cloud 模式。"},
		Async:             true,
		AsyncQueryCommand: "query-task",
		OutputType:        "video",
		Params: []ParamMeta{
			{
				Name:        "video_urls",
				FlagName:    "video-urls",
				Description: "待拼接的视频列表，Array<string>类型。最少传入1个，最多传入100个\n子项说明：待拼接的输入视频。String 类型，支持http://xxx或https://xxx格式 URL",
				Type:        "array",
				ItemType:    "string",
				Required:    true,
				HasDefault:  false,
				JSONEncoded: false,
			},
			{
				Name:        "transitions",
				FlagName:    "transitions",
				Description: "转场效果 ID，Array<string> 类型。如果不提供，则没有转场。当视频数量超过转场数量 2 个及以上时，系统将自动循环使用转场。例如有 10 个视频，2 种转场效果，那么在 9 处拼接点上，这 2 种转场效果将被依次循环使用。\n子项说明：转场效果 ID\n分类：交替出场，ID：1182359\n分类：旋转放大，ID：1182360\n分类：泛开，ID：1182358\n分类：六角形，ID：1182365\n分类：故障转换，ID：1182367\n分类：飞眼，ID：1182368\n分类：梦幻放大，ID：1182369\n分类：开门展现，ID：1182370\n分类：立方转换，ID：1182373\n分类：透镜变换，ID：1182374\n分类：晚霞转场，ID：1182375\n分类：圆形交替，ID：1182378",
				Type:        "array",
				ItemType:    "string",
				Required:    false,
				Enum:        []string{"1182359", "1182360", "1182358", "1182365", "1182367", "1182368", "1182369", "1182370", "1182373", "1182374", "1182375", "1182378"},
				HasDefault:  false,
				JSONEncoded: false,
			},
			{
				Name:        "callback_args",
				FlagName:    "callback-args",
				Description: "可选，回调参数",
				Type:        "string",
				Required:    false,
				HasDefault:  false,
				JSONEncoded: false,
			},
			{
				Name:        "client_token",
				FlagName:    "client-token",
				Description: "可选，用于幂等，默认幂等，用户可根据需求进行调整",
				Type:        "string",
				Required:    false,
				HasDefault:  false,
				JSONEncoded: false,
			},
		},
	},
	{
		Name:              "flip-video",
		DisplayName:       "视频画面翻转",
		Domain:            "editing",
		Description:       "对视频画面进行上下或左右镜像翻转。",
		LocalSupported:    true,
		LocalSource:       "generated",
		LocalDeps:         []string{"ffmpeg", "openh264"},
		LocalLimitations:  []string{"支持本地 FFmpeg 翻转；callback/client_token 本地忽略。"},
		Async:             true,
		AsyncQueryCommand: "query-task",
		OutputType:        "video",
		Params: []ParamMeta{
			{
				Name:        "video_url",
				FlagName:    "video-url",
				Description: "输入视频。String 类型，支持http://xxx或https://xxx格式 URL",
				Type:        "string",
				Required:    true,
				HasDefault:  false,
				JSONEncoded: false,
			},
			{
				Name:         "is_flip_vertical",
				FlagName:     "is-flip-vertical",
				Description: "是否进行垂直翻转。Boolean 类型，默认值为 false， 表示不翻转。",
				Type:         "boolean",
				Required:     false,
				HasDefault:   true,
				DefaultValue: "false",
				JSONEncoded:  false,
			},
			{
				Name:         "is_flip_horizontal",
				FlagName:     "is-flip-horizontal",
				Description: "是否进行水平翻转。Boolean 类型，默认值为 false， 表示不翻转。",
				Type:         "boolean",
				Required:     false,
				HasDefault:   true,
				DefaultValue: "false",
				JSONEncoded:  false,
			},
			{
				Name:        "callback_args",
				FlagName:    "callback-args",
				Description: "可选，回调参数",
				Type:        "string",
				Required:    false,
				HasDefault:  false,
				JSONEncoded: false,
			},
			{
				Name:        "client_token",
				FlagName:    "client-token",
				Description: "可选，用于幂等，默认幂等，用户可根据需求进行调整",
				Type:        "string",
				Required:    false,
				HasDefault:  false,
				JSONEncoded: false,
			},
		},
	},
	{
		Name:              "trim-video",
		DisplayName:       "视频裁剪",
		Domain:            "editing",
		Description:       "按起止时间点裁剪视频，生成新片段。",
		LocalSupported:    true,
		LocalSource:       "generated",
		LocalDeps:         []string{"ffmpeg"},
		LocalLimitations:  []string{"支持本地 FFmpeg 裁剪；callback/client_token 本地忽略。"},
		Async:             true,
		AsyncQueryCommand: "query-task",
		OutputType:        "video",
		Params: []ParamMeta{
			{
				Name:        "video_url",
				FlagName:    "video-url",
				Description: "输入视频。String 类型，支持http://xxx或https://xxx格式 URL",
				Type:        "string",
				Required:    true,
				HasDefault:  false,
				JSONEncoded: false,
			},
			{
				Name:         "start_time",
				FlagName:     "start-time",
				Description: "裁剪开始时间，默认为 0， 表示从头开始裁剪。支持设置为 2 位小数，单位：秒。",
				Type:         "number",
				Required:     false,
				HasDefault:   true,
				DefaultValue: "0",
				JSONEncoded:  false,
			},
			{
				Name:        "end_time",
				FlagName:    "end-time",
				Description: "裁剪结束时间，默认为片源结尾。支持设置为 2 位小数，单位：秒。",
				Type:        "number",
				Required:    false,
				HasDefault:  false,
				JSONEncoded: false,
			},
			{
				Name:        "callback_args",
				FlagName:    "callback-args",
				Description: "可选，回调参数",
				Type:        "string",
				Required:    false,
				HasDefault:  false,
				JSONEncoded: false,
			},
			{
				Name:        "client_token",
				FlagName:    "client-token",
				Description: "可选，用于幂等，默认幂等，用户可根据需求进行调整",
				Type:        "string",
				Required:    false,
				HasDefault:  false,
				JSONEncoded: false,
			},
		},
	},
	{
		Name:              "adjust-video-speed",
		DisplayName:       "视频调速",
		Domain:            "editing",
		Description:       "调整视频的播放倍速，实现快放或慢放效果。",
		LocalSupported:    true,
		LocalSource:       "generated",
		LocalDeps:         []string{"ffmpeg", "openh264"},
		LocalLimitations:  []string{"支持本地 FFmpeg 变速；callback/client_token 本地忽略。"},
		Async:             true,
		AsyncQueryCommand: "query-task",
		OutputType:        "video",
		Params: []ParamMeta{
			{
				Name:        "video_url",
				FlagName:    "video-url",
				Description: "输入视频。String 类型，支持http://xxx或https://xxx格式 URL",
				Type:        "string",
				Required:    true,
				HasDefault:  false,
				JSONEncoded: false,
			},
			{
				Name:         "speed",
				FlagName:     "speed",
				Description: "调整速度的倍数，Float类型，取值范围为0.1～4。",
				Type:         "number",
				Required:     false,
				HasDefault:   true,
				DefaultValue: "1",
				JSONEncoded:  false,
			},
			{
				Name:        "callback_args",
				FlagName:    "callback-args",
				Description: "可选，回调参数",
				Type:        "string",
				Required:    false,
				HasDefault:  false,
				JSONEncoded: false,
			},
			{
				Name:        "client_token",
				FlagName:    "client-token",
				Description: "可选，用于幂等，默认幂等，用户可根据需求进行调整",
				Type:        "string",
				Required:    false,
				HasDefault:  false,
				JSONEncoded: false,
			},
		},
	},
	{
		Name:              "concat-audio",
		DisplayName:       "音频拼接",
		Domain:            "editing",
		Description:       "拼接多个音频片段。",
		LocalSupported:    true,
		LocalSource:       "generated",
		LocalDeps:         []string{"ffmpeg", "demuxer"},
		LocalLimitations:  []string{"支持本地 FFmpeg 音频拼接；callback/client_token 本地忽略。"},
		Async:             true,
		AsyncQueryCommand: "query-task",
		OutputType:        "audio",
		Params: []ParamMeta{
			{
				Name:        "audio_urls",
				FlagName:    "audio-urls",
				Description: "待拼接的音频列表，Array<string>类型。最少传入1个，最多传入100个\n子项说明：待拼接的输入音频。String 类型，支持http://xxx或https://xxx格式 URL",
				Type:        "array",
				ItemType:    "string",
				Required:    true,
				HasDefault:  false,
				JSONEncoded: false,
			},
			{
				Name:        "callback_args",
				FlagName:    "callback-args",
				Description: "可选，回调参数",
				Type:        "string",
				Required:    false,
				HasDefault:  false,
				JSONEncoded: false,
			},
			{
				Name:        "client_token",
				FlagName:    "client-token",
				Description: "可选，用于幂等，默认幂等，用户可根据需求进行调整",
				Type:        "string",
				Required:    false,
				HasDefault:  false,
				JSONEncoded: false,
			},
		},
	},
	{
		Name:              "trim-audio",
		DisplayName:       "音频裁剪",
		Domain:            "editing",
		Description:       "按起止时间点（秒级）裁剪音频，生成新片段。",
		LocalSupported:    true,
		LocalSource:       "generated",
		LocalDeps:         []string{"ffmpeg"},
		LocalLimitations:  []string{"支持本地 FFmpeg 音频裁剪；callback/client_token 本地忽略。"},
		Async:             true,
		AsyncQueryCommand: "query-task",
		OutputType:        "audio",
		Params: []ParamMeta{
			{
				Name:        "audio_url",
				FlagName:    "audio-url",
				Description: "输入纯音频。String 类型，支持http://xxx或https://xxx格式 URL",
				Type:        "string",
				Required:    true,
				HasDefault:  false,
				JSONEncoded: false,
			},
			{
				Name:         "start_time",
				FlagName:     "start-time",
				Description: "裁剪开始时间，默认为 0， 表示从头开始裁剪。支持设置为 2 位小数，单位：秒。",
				Type:         "number",
				Required:     false,
				HasDefault:   true,
				DefaultValue: "0",
				JSONEncoded:  false,
			},
			{
				Name:        "end_time",
				FlagName:    "end-time",
				Description: "裁剪结束时间，默认为片源结尾。支持设置为 2 位小数，单位：秒。",
				Type:        "number",
				Required:    false,
				HasDefault:  false,
				JSONEncoded: false,
			},
			{
				Name:        "callback_args",
				FlagName:    "callback-args",
				Description: "可选，回调参数",
				Type:        "string",
				Required:    false,
				HasDefault:  false,
				JSONEncoded: false,
			},
			{
				Name:        "client_token",
				FlagName:    "client-token",
				Description: "可选，用于幂等，默认幂等，用户可根据需求进行调整",
				Type:        "string",
				Required:    false,
				HasDefault:  false,
				JSONEncoded: false,
			},
		},
	},
	{
		Name:           "fetch-file",
		DisplayName:    "拉取远程文件",
		Domain:         "shared",
		Description:    "将 HTTP/HTTPS 远程文件拉取到本地输出目录；本地路径会被识别并直接返回。",
		LocalSupported: true,
		LocalSource:    "generated",
		Async:          false,
		OutputType:     "file",
		Params: []ParamMeta{
			{
				Name:        "url",
				FlagName:    "url",
				Description: "远程 HTTP/HTTPS URL 或本地文件路径",
				Type:        "string",
				Required:    true,
				HasDefault:  false,
				JSONEncoded: false,
			},
		},
	},
	{
		Name:              "query-task",
		DisplayName:       "查询任务",
		Domain:            "shared",
		Description:       "异步任务结果查询通过task_id查询任务信息",
		LocalSupported:    false,
		Async:             false,
		AsyncQueryCommand: "",
		OutputType:        "query-task",
		Params: []ParamMeta{
			{
				Name:        "task_id",
				FlagName:    "task-id",
				Description: "Path parameter `task_id`.",
				Type:        "string",
				Required:    true,
				HasDefault:  false,
				JSONEncoded: false,
			},
			{
				Name:         "poll_interval_seconds",
				FlagName:     "poll-interval-seconds",
				Description:  "轮询间隔秒数；仅 query-task 使用。",
				Type:         "number",
				Required:     false,
				HasDefault:   true,
				DefaultValue: "10",
				JSONEncoded:  false,
			},
			{
				Name:        "max_poll_attempts",
				FlagName:    "max-poll-attempts",
				Description: "最大轮询次数；0 表示不自动轮询。",
				Type:        "integer",
				Required:    false,
				HasDefault:  false,
				JSONEncoded: false,
			},
			{
				Name:        "poll_complete",
				FlagName:    "poll-complete",
				Description: "是否持续轮询直到任务完成。",
				Type:        "boolean",
				Required:    false,
				HasDefault:  false,
				JSONEncoded: false,
			},
		},
	},
}

func printDomains(cmd *cobra.Command) error {
	_, err := fmt.Fprintln(cmd.OutOrStdout(), renderDomainsIndex())
	return err
}

func printHelpFull(cmd *cobra.Command) error {
	_, err := fmt.Fprintln(cmd.OutOrStdout(), renderHelpFullIndex())
	return err
}

func renderDomainsIndex() string {
	domains := domainList()
	if len(domains) == 0 {
		return "Available domains:\n<generated by CLI generator>"
	}

	lines := []string{"Available domains:"}
	for _, domain := range domains {
		lines = append(lines, fmt.Sprintf("- %s   %s", domain.Name, domain.Description))
	}
	return strings.Join(lines, "\n")
}

func renderHelpFullIndex() string {
	domains := domainList()
	if len(domains) == 0 {
		return "MediaKit CLI Full Help\n\n<generated by CLI generator>"
	}

	grouped := capabilitiesByDomain()
	lines := []string{"MediaKit CLI Full Help", ""}
	for _, domain := range domains {
		lines = append(lines, fmt.Sprintf("[%s]", domain.Name))
		lines = append(lines, domain.Description)
		capabilities := grouped[domain.Name]
		if len(capabilities) == 0 {
			lines = append(lines, "- <no generated capabilities>")
			lines = append(lines, "")
			continue
		}
		for _, capability := range capabilities {
			lines = append(lines, fmt.Sprintf("- %s    %s", capability.Name, capability.Description))
			lines = append(lines, fmt.Sprintf("  查看详情: %s --help", capabilityInvocation(capability)))
		}
		lines = append(lines, "")
	}

	return strings.TrimRight(strings.Join(lines, "\n"), "\n")
}

func newGeneratedDomainCommands() []*cobra.Command {
	domains := domainList()
	grouped := capabilitiesByDomain()
	cmds := make([]*cobra.Command, 0, len(domains))

	for _, meta := range domains {
		domainMeta := meta
		domainCmd := &cobra.Command{
			Use:               domainMeta.Name,
			Short:             domainMeta.Description,
			Long:              renderDomainHelp(domainMeta, grouped[domainMeta.Name]),
			Args:              cobra.NoArgs,
			DisableAutoGenTag: true,
			RunE: func(cmd *cobra.Command, args []string) error {
				return cmd.Help()
			},
		}
		for _, capability := range grouped[domainMeta.Name] {
			domainCmd.AddCommand(newCapabilityCommand(capability))
		}
		cmds = append(cmds, domainCmd)
	}

	return cmds
}

func newCapabilityCommand(meta CapabilityMeta) *cobra.Command {
	capabilityMeta := meta
	cmd := &cobra.Command{
		Use:               capabilityMeta.Name,
		Short:             capabilityMeta.Description,
		Long:              renderCapabilityHelp(capabilityMeta),
		Args:              cobra.NoArgs,
		DisableAutoGenTag: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			// --schema: 输出工具 schema 后退出
			if schemaFlag, _ := cmd.Flags().GetBool("schema"); schemaFlag {
				resolvedMode := resolveSchemaMode(cmd)
				return writeJSON(cmd.OutOrStdout(), buildCapabilitySchema(capabilityMeta, resolvedMode))
			}
			params, err := collectCapabilityParams(cmd, capabilityMeta)
			if err != nil {
				return err
			}
			if err := modes.Dispatch(cmd, capabilityMeta.runtimeMeta(), params); err != nil {
				return writeCapabilityError(cmd.OutOrStdout(), err)
			}
			return nil
		},
	}
	bindCapabilityFlags(cmd, capabilityMeta)
	cmd.Flags().String("output-path", "", "本地文件输出目录（覆盖 config/env 设置）")
	cmd.Flags().Bool("schema", false, "输出该工具的 JSON Schema 描述（供 Agent 使用）")

	return cmd
}

func renderDomainHelp(domain DomainMeta, capabilities []CapabilityMeta) string {
	lines := []string{fmt.Sprintf("%s — %s", domain.Name, domain.Description), "", "Available commands:"}
	if len(capabilities) == 0 {
		lines = append(lines, "- <no generated capabilities>")
	} else {
		for _, capability := range capabilities {
			lines = append(lines, fmt.Sprintf("- %s    %s    [%s]", capability.Name, capability.Description, capability.ModeLabel))
			lines = append(lines, fmt.Sprintf("  查看详情: %s --help", capabilityInvocation(capability)))
		}
	}
	return strings.Join(lines, "\n")
}

func renderCapabilityHelp(meta CapabilityMeta) string {
	lines := []string{
		fmt.Sprintf("命令: %s", capabilityInvocation(meta)),
		fmt.Sprintf("名称: %s", capabilityDisplayName(meta)),
		fmt.Sprintf("分组: %s", meta.Domain),
		fmt.Sprintf("描述: %s", meta.Description),
	}
	if meta.ModeLabel != "" {
		lines = append(lines, fmt.Sprintf("模式: %s", meta.ModeLabel))
	}
	lines = append(lines, "单次调用覆盖:")
	if meta.Name == "fetch-file" {
		lines = append(lines, "  - `fetch-file` 是 local-only 工具，不支持 `--cloud`。")
		lines = append(lines, "  - 可使用 `mediakit-cli --local shared fetch-file --url <url-or-path>` 显式本地执行。")
	} else {
		lines = append(lines, "  - 可使用 `mediakit-cli --local {domain} {tool}`，仅强制当前命令按本地优先策略执行，不会修改全局 config.mode。")
		lines = append(lines, "  - 可使用 `mediakit-cli --cloud {domain} {tool}`，仅强制当前命令按云端优先策略执行，不会修改全局 config.mode。")
	}
	if len(meta.LocalDeps) > 0 {
		lines = append(lines, fmt.Sprintf("本地依赖: %s", strings.Join(meta.LocalDeps, ", ")))
	}
	if len(meta.LocalLimitations) > 0 {
		lines = append(lines, fmt.Sprintf("本地限制: %s", strings.Join(meta.LocalLimitations, "；")))
	}
	if len(meta.Params) > 0 {
		lines = append(lines, "", "参数:")
		for _, param := range meta.Params {
			lines = append(lines, "  "+renderParamDetail(param))
		}
	}
	if hasJSONEncodedParams(meta.Params) {
		lines = append(lines, "", "复杂对象或复杂数组参数请使用 JSON 字符串传入。")
	}
	if hasIdempotencyParams(meta.Params) {
		lines = append(lines, "")
		lines = append(lines, "幂等参数维护:")
		lines = append(lines, "  - 默认开启幂等：即使不传 `--client-token`，相同账户 + 核心请求参数在 2 天内重复提交时，也会直接返回首次任务结果，不会重复创建任务。")
		if hasParam(meta.Params, "client_token") {
			lines = append(lines, "  - `--client-token` 是可选参数；如需主动控制幂等，请传该值，请求重试时复用同一值，强制重新执行时必须传新的唯一值。")
			lines = append(lines, "  - `--client-token` 由客户端生成，长度不超过 64 个字符。")
		}
		if hasParam(meta.Params, "callback_args") {
			lines = append(lines, "  - `--callback-args` 用于透传回调参数，建议与 `--client-token` 一起维护，便于回调对账与重试追踪。")
		}
	}
	if meta.Async {
		queryCommand := meta.AsyncQueryCommand
		if strings.TrimSpace(queryCommand) == "" {
			queryCommand = "query-task"
		}
		lines = append(lines, "")
		lines = append(lines, "ASYNC TASK:")
		lines = append(lines, fmt.Sprintf("  提交后会返回 task_id，可使用 `%s --task-id <task_id>` 查询结果。", capabilityInvocationByName(queryCommand)))
	}
	return strings.Join(lines, "\n")
}

func (m CapabilityMeta) runtimeMeta() modes.CapabilityRuntimeMeta {
	return modes.CapabilityRuntimeMeta{
		Name:           m.Name,
		Domain:         m.Domain,
		Description:    m.Description,
		CloudOnly:      m.CloudOnly,
		LocalSupported: m.LocalSupported,
		LocalSource:    m.LocalSource,
		LocalDeps:      append([]string(nil), m.LocalDeps...),
	}
}

func domainList() []DomainMeta {
	domains := append([]DomainMeta(nil), generatedDomains...)
	sort.Slice(domains, func(i, j int) bool {
		return domains[i].Name < domains[j].Name
	})
	return domains
}

func capabilityList() []CapabilityMeta {
	capabilities := append([]CapabilityMeta(nil), generatedCapabilities...)
	for i := range capabilities {
		capabilities[i] = applyCapabilityConstraints(capabilities[i])
		if capabilities[i].ModeLabel == "" {
			capabilities[i].ModeLabel = modes.ModeLabel(capabilities[i].runtimeMeta())
		}
	}
	sort.Slice(capabilities, func(i, j int) bool {
		return capabilities[i].Name < capabilities[j].Name
	})
	return capabilities
}

func capabilitiesByDomain() map[string][]CapabilityMeta {
	grouped := map[string][]CapabilityMeta{}
	for _, capability := range capabilityList() {
		grouped[capability.Domain] = append(grouped[capability.Domain], capability)
	}
	return grouped
}

func applyCapabilityConstraints(meta CapabilityMeta) CapabilityMeta {
	runtimeMeta := modes.ApplyRuntimeConstraints(meta.runtimeMeta())
	meta.CloudOnly = runtimeMeta.CloudOnly
	meta.LocalSupported = runtimeMeta.LocalSupported
	meta.LocalSource = runtimeMeta.LocalSource
	meta.LocalDeps = append([]string(nil), runtimeMeta.LocalDeps...)
	return meta
}

func capabilityDisplayName(meta CapabilityMeta) string {
	displayName := strings.TrimSpace(meta.DisplayName)
	if displayName == "" {
		return meta.Name
	}
	return displayName
}

func capabilityInvocation(meta CapabilityMeta) string {
	return fmt.Sprintf("mediakit-cli %s %s", meta.Domain, meta.Name)
}

func capabilityInvocationByName(name string) string {
	for _, capability := range capabilityList() {
		if capability.Name == name {
			return capabilityInvocation(capability)
		}
	}
	return "mediakit-cli " + name
}

func bindCapabilityFlags(cmd *cobra.Command, meta CapabilityMeta) {
	for _, param := range meta.Params {
		helpText := renderParamHelp(param)
		switch {
		case param.Type == "string" && !param.JSONEncoded:
			cmd.Flags().String(param.FlagName, defaultString(param), helpText)
		case param.Type == "number":
			cmd.Flags().Float64(param.FlagName, defaultFloat64(param), helpText)
		case param.Type == "integer":
			cmd.Flags().Int64(param.FlagName, defaultInt64(param), helpText)
		case param.Type == "boolean":
			cmd.Flags().Bool(param.FlagName, defaultBool(param), helpText)
		case param.Type == "array" && param.ItemType == "string" && !param.JSONEncoded:
			cmd.Flags().StringSlice(param.FlagName, defaultStringSlice(param), helpText)
		default:
			cmd.Flags().String(param.FlagName, defaultString(param), helpText)
		}
		if param.Required {
			// required 校验移至 collectCapabilityParams，使 --schema 可无参执行
		}
	}
}

func collectCapabilityParams(cmd *cobra.Command, meta CapabilityMeta) (map[string]any, error) {
	// 手动校验必填参数
	var missing []string
	for _, param := range meta.Params {
		if param.Required && !cmd.Flags().Changed(param.FlagName) {
			missing = append(missing, "--"+param.FlagName)
		}
	}
	if len(missing) > 0 {
		return nil, fmt.Errorf("required flag(s) %s not set", strings.Join(missing, ", "))
	}

	params := map[string]any{}
	for _, param := range meta.Params {
		if !cmd.Flags().Changed(param.FlagName) {
			continue
		}
		value, err := readCapabilityParamValue(cmd, param)
		if err != nil {
			return nil, fmt.Errorf("%s 参数无效: %w", param.Name, err)
		}
		params[param.Name] = value
	}
	return params, nil
}

func readCapabilityParamValue(cmd *cobra.Command, param ParamMeta) (any, error) {
	switch {
	case param.Type == "string" && !param.JSONEncoded:
		return cmd.Flags().GetString(param.FlagName)
	case param.Type == "number":
		return cmd.Flags().GetFloat64(param.FlagName)
	case param.Type == "integer":
		return cmd.Flags().GetInt64(param.FlagName)
	case param.Type == "boolean":
		return cmd.Flags().GetBool(param.FlagName)
	case param.Type == "array" && param.ItemType == "string" && !param.JSONEncoded:
		return cmd.Flags().GetStringSlice(param.FlagName)
	default:
		raw, err := cmd.Flags().GetString(param.FlagName)
		if err != nil {
			return nil, err
		}
		raw = strings.TrimSpace(raw)
		if raw == "" {
			return nil, fmt.Errorf("不能为空")
		}
		var decoded any
		if err := json.Unmarshal([]byte(raw), &decoded); err != nil {
			return nil, fmt.Errorf("需要合法 JSON: %w", err)
		}
		return decoded, nil
	}
}

func renderParamHelp(param ParamMeta) string {
	parts := []string{}
	if desc := strings.TrimSpace(param.Description); desc != "" {
		parts = append(parts, desc)
	}
	if extra := idempotencyParamHint(param.Name); extra != "" {
		parts = append(parts, extra)
	}
	if len(param.Enum) > 0 {
		parts = append(parts, "可选值: "+strings.Join(param.Enum, ", "))
	}
	if param.JSONEncoded {
		parts = append(parts, "使用 JSON 字符串传入")
	}
	if param.HasDefault && strings.TrimSpace(param.DefaultValue) != "" {
		parts = append(parts, "默认值: "+param.DefaultValue)
	}
	return strings.Join(parts, " ")
}

func renderParamDetail(param ParamMeta) string {
	qualifiers := []string{renderParamType(param)}
	if param.Required {
		qualifiers = append(qualifiers, "required")
	}
	if param.HasDefault && strings.TrimSpace(param.DefaultValue) != "" {
		qualifiers = append(qualifiers, "default="+param.DefaultValue)
	}
	detail := fmt.Sprintf("- --%s [%s]", param.FlagName, strings.Join(qualifiers, ", "))
	if desc := strings.TrimSpace(param.Description); desc != "" {
		detail += " " + desc
	}
	if extra := idempotencyParamHint(param.Name); extra != "" {
		detail += " " + extra
	}
	if len(param.Enum) > 0 {
		detail += " 可选值: " + strings.Join(param.Enum, ", ")
	}
	if param.JSONEncoded {
		detail += " 复杂值请使用 JSON 字符串。"
	}
	return detail
}

func renderParamType(param ParamMeta) string {
	switch {
	case param.Type == "array" && param.ItemType != "":
		return param.Type + "<" + param.ItemType + ">"
	case param.Type != "":
		return param.Type
	default:
		return "string"
	}
}

func hasJSONEncodedParams(params []ParamMeta) bool {
	for _, param := range params {
		if param.JSONEncoded {
			return true
		}
	}
	return false
}

func hasIdempotencyParams(params []ParamMeta) bool {
	return hasParam(params, "client_token") || hasParam(params, "callback_args")
}

func hasParam(params []ParamMeta, name string) bool {
	for _, param := range params {
		if param.Name == name {
			return true
		}
	}
	return false
}

func idempotencyParamHint(name string) string {
	switch name {
	case "client_token":
		return "可选，用于主动控制幂等；不传也默认幂等。重试复用同一值，强制重新执行请更换新的唯一值；长度不超过 64 个字符。"
	case "callback_args":
		return "用于透传回调参数，建议与 client_token 一起维护。"
	default:
		return ""
	}
}

func defaultString(param ParamMeta) string {
	if !param.HasDefault {
		return ""
	}
	return param.DefaultValue
}

func defaultFloat64(param ParamMeta) float64 {
	if !param.HasDefault {
		return 0
	}
	value, err := strconv.ParseFloat(strings.TrimSpace(param.DefaultValue), 64)
	if err != nil {
		return 0
	}
	return value
}

func defaultInt64(param ParamMeta) int64 {
	if !param.HasDefault {
		return 0
	}
	value, err := strconv.ParseInt(strings.TrimSpace(param.DefaultValue), 10, 64)
	if err != nil {
		return 0
	}
	return value
}

func defaultBool(param ParamMeta) bool {
	if !param.HasDefault {
		return false
	}
	value, err := strconv.ParseBool(strings.TrimSpace(param.DefaultValue))
	if err != nil {
		return false
	}
	return value
}

func defaultStringSlice(param ParamMeta) []string {
	if !param.HasDefault || strings.TrimSpace(param.DefaultValue) == "" {
		return nil
	}
	var values []string
	if err := json.Unmarshal([]byte(param.DefaultValue), &values); err != nil {
		return nil
	}
	return values
}

func writeCapabilityError(writer io.Writer, err error) error {
	// 如果是 DependencyError，使用其自带的结构化输出（含 install_guide）
	var depErr *local.DependencyError
	if errors.As(err, &depErr) {
		encoder := json.NewEncoder(writer)
		encoder.SetEscapeHTML(false)
		encoder.SetIndent("", "  ")
		return encoder.Encode(depErr.StructuredError())
	}

	encoder := json.NewEncoder(writer)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")
	return encoder.Encode(map[string]any{
		"error": map[string]any{
			"type":    classifyErrorType(err),
			"code":    classifyErrorCode(err),
			"message": err.Error(),
		},
	})
}

func classifyErrorType(err error) string {
	msg := err.Error()
	switch {
	case strings.Contains(msg, "禁止") ||
		strings.Contains(msg, "不在白名单") ||
		strings.Contains(msg, "不安全字符"):
		return "security_violation"
	case strings.Contains(msg, "必填参数") ||
		strings.Contains(msg, "必须是") ||
		strings.Contains(msg, "取值范围") ||
		strings.Contains(msg, "至少需要") ||
		strings.Contains(msg, "仅支持") ||
		strings.Contains(msg, "必须大于") ||
		strings.Contains(msg, "必须大于等于"):
		return "invalid_parameter"
	case strings.Contains(msg, "本地处理器未实现") ||
		strings.Contains(msg, "不支持本地执行") ||
		strings.Contains(msg, "本地依赖"):
		return "environment_error"
	default:
		return "execution_error"
	}
}

func classifyErrorCode(err error) string {
	msg := err.Error()
	switch {
	case strings.Contains(msg, "禁止"):
		return "forbidden_operation"
	case strings.Contains(msg, "不在白名单"):
		return "not_whitelisted"
	case strings.Contains(msg, "不安全字符"):
		return "unsafe_characters"
	case strings.Contains(msg, "必填参数"):
		return "missing_required_param"
	case strings.Contains(msg, "必须大于等于"):
		return "param_out_of_range"
	case strings.Contains(msg, "必须大于"):
		return "param_out_of_range"
	case strings.Contains(msg, "取值范围"):
		return "param_out_of_range"
	case strings.Contains(msg, "必须是"):
		return "invalid_param_type"
	case strings.Contains(msg, "至少需要"):
		return "param_insufficient"
	case strings.Contains(msg, "仅支持"):
		return "unsupported_value"
	case strings.Contains(msg, "本地处理器未实现"):
		return "handler_not_implemented"
	case strings.Contains(msg, "不支持本地执行"):
		return "local_unsupported"
	case strings.Contains(msg, "本地依赖"):
		return "dependency_missing"
	case strings.Contains(msg, "download failed"):
		return "download_failed"
	case strings.Contains(msg, "执行失败"):
		return "execution_failed"
	default:
		return "unknown"
	}
}

func writeJSON(out io.Writer, value any) error {
	encoder := json.NewEncoder(out)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")
	return encoder.Encode(value)
}

func buildCapabilitySchema(meta CapabilityMeta, resolvedMode string) map[string]any {
	// 构建 input_schema properties
	properties := map[string]any{}
	required := []string{}

	for _, param := range meta.Params {
		prop := map[string]any{}

		// 类型映射
		switch param.Type {
		case "number":
			prop["type"] = "number"
		case "integer":
			prop["type"] = "integer"
		case "boolean":
			prop["type"] = "boolean"
		case "array":
			prop["type"] = "array"
			if param.ItemType != "" {
				prop["items"] = map[string]any{"type": param.ItemType}
			}
		default:
			prop["type"] = "string"
		}

		if param.Description != "" {
			prop["description"] = param.Description
		}
		if len(param.Enum) > 0 {
			prop["enum"] = param.Enum
		}
		if param.HasDefault && param.DefaultValue != "" {
			prop["default"] = param.DefaultValue
		}
		if param.Required {
			required = append(required, param.Name)
		}

		properties[param.Name] = prop
	}

	inputSchema := map[string]any{
		"type":       "object",
		"properties": properties,
	}
	if len(required) > 0 {
		inputSchema["required"] = required
	}

	// 构建描述
	description := meta.Description
	if meta.Async {
		description += "\n\n- Async: 是,用 mediakit_shared_query_task 查询结果"
	}
	modeLabel := modes.ModeLabel(meta.runtimeMeta())
	description += "\n- Mode: " + modeLabel

	// 构建工具名（domain_tool 格式）
	toolName := strings.ReplaceAll(meta.Name, "-", "_")

	schema := map[string]any{
		"name":          toolName,
		"description":   description,
		"input_schema":  inputSchema,
		"output_schema": buildOutputSchema(meta, resolvedMode),
	}

	return schema
}

// resolveSchemaMode 根据 --local/--cloud flag 和全局配置决定 schema 输出的模式
func resolveSchemaMode(cmd *cobra.Command) string {
	localFlag, _ := cmd.Flags().GetBool("local")
	cloudFlag, _ := cmd.Flags().GetBool("cloud")
	if localFlag {
		return cliconfig.ModeLocalFirst
	}
	if cloudFlag {
		return cliconfig.ModeCloudFirst
	}
	// 读取全局配置
	home, err := cliconfig.ResolveHomeDir()
	if err != nil {
		return cliconfig.ModeCloudFirst
	}
	resolved, err := cliconfig.ResolveConfig(home)
	if err != nil {
		return cliconfig.ModeCloudFirst
	}
	return resolved.Mode
}

// buildOutputSchema 根据当前 mode 输出对应的 output schema
func buildOutputSchema(meta CapabilityMeta, resolvedMode string) map[string]any {
	// query-task 特殊处理：始终返回 cloud schema
	if meta.OutputType == "query-task" {
		return buildFinalResultSchema("query-task", "查询异步任务状态与结果")
	}

	// 根据 mode 决定输出哪种 schema
	switch resolvedMode {
	case cliconfig.ModeLocalFirst:
		if meta.LocalSupported {
			return buildFinalResultSchema(meta.OutputType, "本地模式直接返回处理结果")
		}
		// local-first 但该命令不支持本地，降级展示 cloud
		if meta.Async {
			return buildAsyncCloudSchema(meta)
		}
		return buildFinalResultSchema(meta.OutputType, "云端同步返回结果")

	default: // cloud-first
		if meta.Async {
			return buildAsyncCloudSchema(meta)
		}
		if meta.LocalSupported && !meta.CloudOnly {
			// 同步且支持 local 的 fetch-file 等
			return buildFinalResultSchema(meta.OutputType, "本地模式直接返回处理结果")
		}
		return buildFinalResultSchema(meta.OutputType, "云端同步返回结果")
	}
}

func buildAsyncCloudSchema(meta CapabilityMeta) map[string]any {
	return map[string]any{
		"description": "云端异步模式返回 task_id，需通过 query-task 轮询获取最终结果",
		"type":        "object",
		"properties": map[string]any{
			"task_id":    map[string]any{"type": "string", "description": "异步任务 ID，用于 query-task 查询"},
			"request_id": map[string]any{"type": "string", "description": "请求 ID"},
		},
		"final_result": buildFinalResultSchema(meta.OutputType, "query-task 完成后的最终结果"),
	}
}

func buildFinalResultSchema(outputType string, description string) map[string]any {
	schema := map[string]any{
		"description": description,
		"type":        "object",
	}

	switch outputType {
	case "video":
		schema["properties"] = map[string]any{
			"video_url":  map[string]any{"type": "string", "description": "输出视频文件路径或 URL"},
			"duration":   map[string]any{"type": "number", "description": "视频时长，单位：秒"},
			"resolution": map[string]any{"type": "string", "description": "视频分辨率档位（如 360p, 480p, 720p, 1080p, 2k, 4k）"},
		}
	case "audio":
		schema["properties"] = map[string]any{
			"audio_url": map[string]any{"type": "string", "description": "输出音频文件路径或 URL"},
			"duration":  map[string]any{"type": "number", "description": "音频时长，单位：秒"},
		}
	case "file":
		schema["properties"] = map[string]any{
			"local_path": map[string]any{"type": "string", "description": "本地文件路径"},
		}
	case "query-task":
		schema["properties"] = map[string]any{
			"task_id":    map[string]any{"type": "string", "description": "任务 ID"},
			"status":     map[string]any{"type": "string", "description": "任务状态（processing, success, failed）"},
			"request_id": map[string]any{"type": "string", "description": "请求 ID"},
			"video_url":  map[string]any{"type": "string", "description": "任务完成后的输出文件 URL（视频类任务）"},
			"audio_url":  map[string]any{"type": "string", "description": "任务完成后的输出文件 URL（音频类任务）"},
		}
	default:
		schema["properties"] = map[string]any{}
	}

	return schema
}
