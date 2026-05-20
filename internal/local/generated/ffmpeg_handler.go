package generated

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"mediakit-cli/internal/local/core"
)

// FFmpegPlan describes a generated local ffmpeg execution and its JSON result.
type FFmpegPlan struct {
	Args   []string
	Result map[string]any
}

// BuildPlanFunc lets generated handlers construct ffmpeg plans from params.
type BuildPlanFunc func(ctx *core.ExecContext) (*FFmpegPlan, error)

// NewFFmpegHandler creates a generated local handler backed by ffmpeg.
func NewFFmpegHandler(build BuildPlanFunc) core.Handler {
	return core.HandlerFunc(func(ctx *core.ExecContext) (map[string]any, error) {
		plan, err := build(ctx)
		if err != nil {
			return nil, err
		}
		if plan == nil {
			return nil, fmt.Errorf("ffmpeg plan is required")
		}
		if validateErr := core.DefaultFFmpegPolicy().ValidateArgs(plan.Args); validateErr != nil {
			return nil, validateErr
		}

		output, err := core.RunFFmpeg(plan.Args...)
		if err != nil {
			return nil, err
		}

		if plan.Result == nil {
			plan.Result = map[string]any{}
		}
		if text := strings.TrimSpace(string(output)); text != "" && len(plan.Result) == 0 {
			plan.Result["ffmpeg_output"] = text
		}
		enrichMediaResult(plan.Result)
		return plan.Result, nil
	})
}

type ffprobeMediaInfo struct {
	Format struct {
		Duration string `json:"duration"`
	} `json:"format"`
	Streams []struct {
		Width  int `json:"width"`
		Height int `json:"height"`
	} `json:"streams"`
}

func enrichMediaResult(result map[string]any) {
	if len(result) == 0 {
		return
	}
	output, isVideo := stringResultField(result, "video_url")
	if output == "" {
		output, _ = stringResultField(result, "audio_url")
	}
	if output == "" {
		return
	}

	info, err := probeMediaInfo(output)
	if err != nil {
		return
	}
	if duration, err := strconv.ParseFloat(strings.TrimSpace(info.Format.Duration), 64); err == nil && duration > 0 {
		result["duration"] = duration
	}
	if isVideo {
		for _, stream := range info.Streams {
			if stream.Width > 0 && stream.Height > 0 {
				// Use the shorter side to determine resolution class,
				// so portrait videos (e.g. 1080×1920) are not misclassified.
				shortSide := stream.Height
				if stream.Width < stream.Height {
					shortSide = stream.Width
				}
				if resolution := resolutionFromHeight(shortSide); resolution != "" {
					result["resolution"] = resolution
					break
				}
			}
		}
	}
}

func stringResultField(result map[string]any, key string) (string, bool) {
	value, ok := result[key].(string)
	if !ok {
		return "", false
	}
	return strings.TrimSpace(value), true
}

func probeMediaInfo(output string) (ffprobeMediaInfo, error) {
	var info ffprobeMediaInfo
	raw, err := core.RunFFprobe(
		"-v", "error",
		"-show_entries", "format=duration:stream=width,height",
		"-of", "json",
		output,
	)
	if err != nil {
		return info, err
	}
	if err := json.Unmarshal(raw, &info); err != nil {
		return info, err
	}
	return info, nil
}

func resolutionFromHeight(height int) string {
	switch {
	case height <= 0:
		return ""
	case height <= 240:
		return "240p"
	case height <= 360:
		return "360p"
	case height <= 480:
		return "480p"
	case height <= 540:
		return "540p"
	case height <= 720:
		return "720p"
	case height <= 1080:
		return "1080p"
	case height <= 1440:
		return "2k"
	default:
		return "4k"
	}
}

func preferredH264Encoder() string {
	if supportsFFmpegEncoder("libopenh264") {
		return "libopenh264"
	}
	if supportsFFmpegEncoder("h264_videotoolbox") {
		return "h264_videotoolbox"
	}
	return "libopenh264"
}

func supportsFFmpegEncoder(encoder string) bool {
	output, err := exec.Command("ffmpeg", "-hide_banner", "-encoders").CombinedOutput()
	if err != nil {
		return false
	}
	return strings.Contains(strings.ToLower(string(output)), strings.ToLower(encoder))
}

func h264OutputArgs(videoBitrate string) []string {
	encoder := preferredH264Encoder()
	args := []string{"-c:v", encoder, "-b:v", videoBitrate}
	if encoder == "h264_videotoolbox" {
		args = append(args, "-pix_fmt", "yuv420p")
	}
	return args
}

func materializeRequiredInput(ctx *core.ExecContext, key string) (string, error) {
	value, err := requiredStringParam(ctx.Params, key)
	if err != nil {
		return "", err
	}
	return core.MaterializeInput(ctx, value)
}

func materializeRequiredInputList(ctx *core.ExecContext, key string) ([]string, error) {
	values, err := requiredStringListParam(ctx.Params, key)
	if err != nil {
		return nil, err
	}
	outputs := make([]string, 0, len(values))
	for _, value := range values {
		item, err := core.MaterializeInput(ctx, value)
		if err != nil {
			return nil, err
		}
		outputs = append(outputs, item)
	}
	return outputs, nil
}

func requiredStringParam(params map[string]any, key string) (string, error) {
	value, ok, err := optionalStringParam(params, key)
	if err != nil {
		return "", err
	}
	if !ok {
		return "", fmt.Errorf("%s 是必填参数", key)
	}
	return value, nil
}

func optionalStringParam(params map[string]any, key string) (string, bool, error) {
	value, ok := params[key]
	if !ok {
		return "", false, nil
	}
	text, ok := value.(string)
	if !ok {
		return "", false, fmt.Errorf("%s 必须是字符串", key)
	}
	text = strings.TrimSpace(text)
	if text == "" {
		return "", false, nil
	}
	if err := core.ValidateSafeText(text, key); err != nil {
		return "", false, err
	}
	return text, true, nil
}

func requiredStringListParam(params map[string]any, key string) ([]string, error) {
	value, ok := params[key]
	if !ok {
		return nil, fmt.Errorf("%s 是必填参数", key)
	}
	items, err := stringListValue(value, key)
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return nil, fmt.Errorf("%s 至少需要 1 个元素", key)
	}
	return items, nil
}

func stringListValue(value any, key string) ([]string, error) {
	switch typed := value.(type) {
	case []string:
		items := make([]string, 0, len(typed))
		for _, item := range typed {
			text := strings.TrimSpace(item)
			if text == "" {
				continue
			}
			if err := core.ValidateSafeText(text, key); err != nil {
				return nil, err
			}
			items = append(items, text)
		}
		return items, nil
	case []any:
		items := make([]string, 0, len(typed))
		for _, raw := range typed {
			text, ok := raw.(string)
			if !ok {
				return nil, fmt.Errorf("%s 必须是字符串数组", key)
			}
			text = strings.TrimSpace(text)
			if text == "" {
				continue
			}
			if err := core.ValidateSafeText(text, key); err != nil {
				return nil, err
			}
			items = append(items, text)
		}
		return items, nil
	default:
		return nil, fmt.Errorf("%s 必须是字符串数组", key)
	}
}

func optionalFloatParam(params map[string]any, key string) (float64, bool, error) {
	value, ok := params[key]
	if !ok {
		return 0, false, nil
	}
	switch typed := value.(type) {
	case float64:
		return typed, true, nil
	case float32:
		return float64(typed), true, nil
	case int:
		return float64(typed), true, nil
	case int64:
		return float64(typed), true, nil
	case string:
		text := strings.TrimSpace(typed)
		if text == "" {
			return 0, false, nil
		}
		parsed, err := strconv.ParseFloat(text, 64)
		if err != nil {
			return 0, false, fmt.Errorf("%s 必须是数字", key)
		}
		return parsed, true, nil
	default:
		return 0, false, fmt.Errorf("%s 必须是数字", key)
	}
}

func validateTrimWindow(start float64, hasStart bool, end float64, hasEnd bool) error {
	if hasStart && start < 0 {
		return fmt.Errorf("start_time 必须大于等于 0")
	}
	if hasEnd && end <= 0 {
		return fmt.Errorf("end_time 必须大于 0")
	}
	if hasStart && hasEnd && end <= start {
		return fmt.Errorf("end_time 必须大于 start_time")
	}
	return nil
}

func trimDuration(start float64, hasStart bool, end float64, hasEnd bool) (float64, bool) {
	if !hasEnd {
		return 0, false
	}
	if hasStart {
		return end - start, true
	}
	return end, true
}

func formatFloat(value float64) string {
	return strconv.FormatFloat(value, 'f', -1, 64)
}

func preferredExtFromPath(source string, fallback string) string {
	ext := strings.TrimSpace(filepath.Ext(source))
	if ext != "" {
		return ext
	}
	if strings.TrimSpace(fallback) == "" {
		return ".bin"
	}
	if strings.HasPrefix(fallback, ".") {
		return fallback
	}
	return "." + fallback
}

func outputPathFor(ctx *core.ExecContext, command string, ext string) (string, error) {
	// 如果用户指定了完整输出文件路径，直接使用
	if ctx.OutputFile != "" {
		if err := os.MkdirAll(filepath.Dir(ctx.OutputFile), 0o755); err != nil {
			return "", err
		}
		return ctx.OutputFile, nil
	}

	command = strings.TrimSpace(strings.ReplaceAll(command, "_", "-"))
	if command == "" {
		command = "output"
	}
	ext = preferredExtFromPath("", ext)

	// 尝试从输入参数提取源文件名
	inputName := extractInputBaseName(ctx.Params)
	var filename string
	if inputName != "" {
		// 原文件名_工具名.ext
		filename = fmt.Sprintf("%s_%s%s", inputName, command, ext)
	} else {
		// 无文件名时保持原逻辑
		filename = fmt.Sprintf("%s-%d%s", command, time.Now().UnixNano(), ext)
	}

	outputPath := filepath.Join(ctx.OutputDir, filename)

	// 文件已存在则加 6 位随机串
	if _, err := os.Stat(outputPath); err == nil {
		randSuffix := fmt.Sprintf("%06d", rand.Intn(1000000))
		if inputName != "" {
			filename = fmt.Sprintf("%s_%s_%s%s", inputName, command, randSuffix, ext)
		} else {
			filename = fmt.Sprintf("%s-%s%s", command, randSuffix, ext)
		}
		outputPath = filepath.Join(ctx.OutputDir, filename)
	}

	if err := os.MkdirAll(filepath.Dir(outputPath), 0o755); err != nil {
		return "", err
	}
	return outputPath, nil
}

// extractInputBaseName 从参数中提取输入文件的基础名称（不含扩展名）
func extractInputBaseName(params map[string]any) string {
	// 优先取 video_url，其次 audio_url
	for _, key := range []string{"video_url", "audio_url"} {
		if val, ok := params[key]; ok {
			if s, ok := val.(string); ok && s != "" {
				return fileBaseName(s)
			}
		}
	}
	// concat 场景取第一个
	for _, key := range []string{"video_urls", "audio_urls"} {
		if val, ok := params[key]; ok {
			if arr, ok := val.([]any); ok && len(arr) > 0 {
				if s, ok := arr[0].(string); ok && s != "" {
					return fileBaseName(s)
				}
			}
		}
	}
	return ""
}

// fileBaseName 提取路径或 URL 的文件名（不含扩展名）
func fileBaseName(path string) string {
	// 尝试解析为 URL
	if u, err := url.Parse(path); err == nil && u.Path != "" {
		path = u.Path
	}
	base := filepath.Base(path)
	ext := filepath.Ext(base)
	name := strings.TrimSuffix(base, ext)
	// URL 解码（如 %E5%8F%A3%E6%92%AD -> 口播）
	if decoded, err := url.PathUnescape(name); err == nil {
		name = decoded
	}
	if name == "" || name == "." {
		return ""
	}
	return name
}

func concatListFile(ctx *core.ExecContext, inputs []string) (string, error) {
	if len(inputs) == 0 {
		return "", fmt.Errorf("至少需要 1 个输入文件")
	}
	lines := make([]string, 0, len(inputs))
	for _, input := range inputs {
		lines = append(lines, fmt.Sprintf("file '%s'", escapeConcatPath(input)))
	}
	target := filepath.Join(ctx.TempDir, fmt.Sprintf("concat-%d.txt", time.Now().UnixNano()))
	if err := os.WriteFile(target, []byte(strings.Join(lines, "\n")+"\n"), 0o600); err != nil {
		return "", err
	}
	return target, nil
}

func escapeConcatPath(value string) string {
	value = strings.ReplaceAll(value, "\\", "\\\\")
	value = strings.ReplaceAll(value, "'", "'\\''")
	return value
}

func extractAudioSpec(params map[string]any) (string, string, error) {
	formatValue, ok, err := optionalStringParam(params, "format")
	if err != nil {
		return "", "", err
	}
	if !ok {
		return ".m4a", "aac", nil
	}
	switch strings.ToLower(formatValue) {
	case "m4a":
		return ".m4a", "aac", nil
	case "mp3":
		return ".mp3", "libmp3lame", nil
	default:
		return "", "", fmt.Errorf("format 仅支持 mp3 或 m4a")
	}
}
