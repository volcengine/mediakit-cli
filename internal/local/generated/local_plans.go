//go:build !mediakit_cloud_only

package generated

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"mediakit-cli/internal/local/core"
)

type LocalInputRef struct {
	Original  string `json:"original"`
	LocalPath string `json:"local_path"`
}

type ffprobeAudioMetadata struct {
	Format struct {
		FormatName string `json:"format_name"`
		BitRate    string `json:"bit_rate"`
		Duration   string `json:"duration"`
		Size       string `json:"size"`
	} `json:"format"`
	Streams []struct {
		CodecName  string `json:"codec_name"`
		Duration   string `json:"duration"`
		SampleRate string `json:"sample_rate"`
		BitRate    string `json:"bit_rate"`
		Channels   int    `json:"channels"`
	} `json:"streams"`
}

type ffprobeMediaMetadata struct {
	Format struct {
		FormatName string `json:"format_name"`
		BitRate    string `json:"bit_rate"`
		Duration   string `json:"duration"`
		Size       string `json:"size"`
	} `json:"format"`
	Streams []struct {
		CodecType      string `json:"codec_type"`
		CodecName      string `json:"codec_name"`
		Width          int    `json:"width"`
		Height         int    `json:"height"`
		Duration       string `json:"duration"`
		BitRate        string `json:"bit_rate"`
		AvgFrameRate   string `json:"avg_frame_rate"`
		RFrameRate     string `json:"r_frame_rate"`
		ColorTransfer  string `json:"color_transfer"`
		ColorPrimaries string `json:"color_primaries"`
		ColorSpace     string `json:"color_space"`
		SampleRate     string `json:"sample_rate"`
		Channels       int    `json:"channels"`
	} `json:"streams"`
}

func materializeLocalInput(ctx *core.ExecContext, value string) (LocalInputRef, error) {
	localPath, err := core.MaterializeInput(ctx, value)
	if err != nil {
		return LocalInputRef{}, err
	}
	return LocalInputRef{Original: value, LocalPath: localPath}, nil
}

func outputPath(ctx *core.ExecContext, command string, ext string) (string, error) {
	return outputPathFor(ctx, command, ext)
}

func concatFile(ctx *core.ExecContext, inputs []LocalInputRef) (string, error) {
	paths := make([]string, 0, len(inputs))
	for _, input := range inputs {
		paths = append(paths, input.LocalPath)
	}
	return concatListFile(ctx, paths)
}

func unsupportedIfPresent(params map[string]any, key string) error {
	value, ok := params[key]
	if !ok || value == nil {
		return nil
	}
	switch typed := value.(type) {
	case string:
		if strings.TrimSpace(typed) == "" {
			return nil
		}
	case []any:
		if len(typed) == 0 {
			return nil
		}
	case []string:
		if len(typed) == 0 {
			return nil
		}
	}
	return fmt.Errorf("%s 当前不支持本地执行，请使用 cloud 模式", key)
}

func videoResponse(command string, output string, inputs []LocalInputRef, extra map[string]any) map[string]any {
	return localMediaResponse(command, "video", output, inputs, extra)
}

func audioResponse(command string, output string, inputs []LocalInputRef, extra map[string]any) map[string]any {
	return localMediaResponse(command, "audio", output, inputs, extra)
}

func localMediaResponse(command string, mediaType string, output string, inputs []LocalInputRef, extra map[string]any) map[string]any {
	result := map[string]any{}
	switch mediaType {
	case "audio":
		result["audio_url"] = output
	default:
		result["video_url"] = output
	}
	return result
}

func buildConcatVideoPlan(ctx *core.ExecContext) (*FFmpegPlan, error) {
	if err := unsupportedIfPresent(ctx.Params, "transitions"); err != nil {
		return nil, err
	}

	urls, err := requiredStringListParam(ctx.Params, "video_urls")
	if err != nil {
		return nil, err
	}
	inputs := make([]LocalInputRef, 0, len(urls))
	for _, url := range urls {
		input, err := materializeLocalInput(ctx, url)
		if err != nil {
			return nil, err
		}
		inputs = append(inputs, input)
	}
	fileList, err := concatFile(ctx, inputs)
	if err != nil {
		return nil, err
	}
	out, err := outputPath(ctx, "concat-video", ".mp4")
	if err != nil {
		return nil, err
	}
	args := []string{"-y", "-hide_banner", "-f", "concat", "-safe", "0", "-i", fileList, "-map", "0:v:0", "-map", "0:a:0?", "-c", "copy", out}
	return &FFmpegPlan{
		Args: args,
		Result: videoResponse("concat-video", out, inputs, map[string]any{
			"input_count": len(inputs),
		}),
	}, nil
}

func buildConcatAudioPlan(ctx *core.ExecContext) (*FFmpegPlan, error) {
	urls, err := requiredStringListParam(ctx.Params, "audio_urls")
	if err != nil {
		return nil, err
	}
	inputs := make([]LocalInputRef, 0, len(urls))
	for _, url := range urls {
		input, err := materializeLocalInput(ctx, url)
		if err != nil {
			return nil, err
		}
		inputs = append(inputs, input)
	}
	fileList, err := concatFile(ctx, inputs)
	if err != nil {
		return nil, err
	}
	out, err := outputPath(ctx, "concat-audio", ".m4a")
	if err != nil {
		return nil, err
	}
	args := []string{"-y", "-hide_banner", "-f", "concat", "-safe", "0", "-i", fileList, "-vn", "-c:a", "aac", "-b:a", "128k", out}
	return &FFmpegPlan{
		Args: args,
		Result: audioResponse("concat-audio", out, inputs, map[string]any{
			"input_count": len(inputs),
		}),
	}, nil
}

func buildTrimVideoPlan(ctx *core.ExecContext) (*FFmpegPlan, error) {
	videoURL, err := requiredStringParam(ctx.Params, "video_url")
	if err != nil {
		return nil, err
	}
	input, err := materializeLocalInput(ctx, videoURL)
	if err != nil {
		return nil, err
	}
	start, hasStart, err := optionalFloatParam(ctx.Params, "start_time")
	if err != nil {
		return nil, err
	}
	end, hasEnd, err := optionalFloatParam(ctx.Params, "end_time")
	if err != nil {
		return nil, err
	}
	if err := validateTrimWindow(start, hasStart, end, hasEnd); err != nil {
		return nil, err
	}
	out, err := outputPath(ctx, "trim-video", ".mp4")
	if err != nil {
		return nil, err
	}
	args := []string{"-y", "-hide_banner"}
	if hasStart {
		args = append(args, "-ss", formatFloat(start))
	}
	args = append(args, "-i", input.LocalPath)
	if duration, ok := trimDuration(start, hasStart, end, hasEnd); ok {
		args = append(args, "-t", formatFloat(duration))
	}
	args = append(args, "-map", "0:v:0", "-map", "0:a:0?", "-c", "copy", out)
	return &FFmpegPlan{Args: args, Result: videoResponse("trim-video", out, []LocalInputRef{input}, map[string]any{"start_time": start, "end_time": end})}, nil
}

func buildTrimAudioPlan(ctx *core.ExecContext) (*FFmpegPlan, error) {
	audioURL, err := requiredStringParam(ctx.Params, "audio_url")
	if err != nil {
		return nil, err
	}
	input, err := materializeLocalInput(ctx, audioURL)
	if err != nil {
		return nil, err
	}
	start, hasStart, err := optionalFloatParam(ctx.Params, "start_time")
	if err != nil {
		return nil, err
	}
	end, hasEnd, err := optionalFloatParam(ctx.Params, "end_time")
	if err != nil {
		return nil, err
	}
	if err := validateTrimWindow(start, hasStart, end, hasEnd); err != nil {
		return nil, err
	}
	out, err := outputPath(ctx, "trim-audio", ".m4a")
	if err != nil {
		return nil, err
	}
	args := []string{"-y", "-hide_banner"}
	if hasStart {
		args = append(args, "-ss", formatFloat(start))
	}
	args = append(args, "-i", input.LocalPath)
	if duration, ok := trimDuration(start, hasStart, end, hasEnd); ok {
		args = append(args, "-t", formatFloat(duration))
	}
	args = append(args, "-vn", "-c:a", "aac", "-b:a", "128k", out)
	return &FFmpegPlan{Args: args, Result: audioResponse("trim-audio", out, []LocalInputRef{input}, map[string]any{"start_time": start, "end_time": end})}, nil
}

func buildMuxAudioVideoPlan(ctx *core.ExecContext) (*FFmpegPlan, error) {
	videoURL, err := requiredStringParam(ctx.Params, "video_url")
	if err != nil {
		return nil, err
	}
	audioURL, err := requiredStringParam(ctx.Params, "audio_url")
	if err != nil {
		return nil, err
	}
	video, err := materializeLocalInput(ctx, videoURL)
	if err != nil {
		return nil, err
	}
	audio, err := materializeLocalInput(ctx, audioURL)
	if err != nil {
		return nil, err
	}
	out, err := outputPath(ctx, "mux-audio-video", ".mp4")
	if err != nil {
		return nil, err
	}
	args := []string{"-y", "-hide_banner", "-i", video.LocalPath, "-i", audio.LocalPath, "-map", "0:v:0", "-map", "1:a:0", "-c:v", "copy", "-c:a", "aac", "-b:a", "128k", "-shortest", out}
	return &FFmpegPlan{Args: args, Result: videoResponse("mux-audio-video", out, []LocalInputRef{video, audio}, map[string]any{})}, nil
}

func buildFadeVideoAudioPlan(ctx *core.ExecContext) (*FFmpegPlan, error) {
	videoURL, err := requiredStringParam(ctx.Params, "video_url")
	if err != nil {
		return nil, err
	}
	input, err := materializeLocalInput(ctx, videoURL)
	if err != nil {
		return nil, err
	}
	if !hasAudioStream(input.LocalPath) {
		return nil, fmt.Errorf("fade-video-audio 需要输入视频包含音频流")
	}
	fadeIn, hasFadeIn, err := optionalFloatParam(ctx.Params, "fade_in_duration")
	if err != nil {
		return nil, err
	}
	if !hasFadeIn {
		fadeIn = 1
	}
	fadeOut, hasFadeOut, err := optionalFloatParam(ctx.Params, "fade_out_duration")
	if err != nil {
		return nil, err
	}
	if !hasFadeOut {
		fadeOut = 1
	}
	if fadeIn < 0 {
		return nil, fmt.Errorf("fade_in_duration 必须大于等于 0")
	}
	if fadeOut < 0 {
		return nil, fmt.Errorf("fade_out_duration 必须大于等于 0")
	}
	out, err := outputPath(ctx, "fade-video-audio", ".mp4")
	if err != nil {
		return nil, err
	}
	if fadeIn == 0 && fadeOut == 0 {
		args := []string{"-y", "-hide_banner", "-i", input.LocalPath, "-map", "0:v:0", "-map", "0:a:0", "-c", "copy", out}
		return &FFmpegPlan{Args: args, Result: videoResponse("fade-video-audio", out, []LocalInputRef{input}, map[string]any{"fade_in_duration": fadeIn, "fade_out_duration": fadeOut})}, nil
	}
	duration, err := mediaDuration(input.LocalPath)
	if err != nil {
		return nil, err
	}
	filter := audioFadeFilter(fadeIn, fadeOut, duration)
	args := []string{"-y", "-hide_banner", "-i", input.LocalPath, "-filter_complex", "[0:a]" + filter + "[outa]", "-map", "0:v:0", "-map", "[outa]", "-c:v", "copy", "-c:a", "aac", "-b:a", "128k", out}
	return &FFmpegPlan{Args: args, Result: videoResponse("fade-video-audio", out, []LocalInputRef{input}, map[string]any{"fade_in_duration": fadeIn, "fade_out_duration": fadeOut})}, nil
}

func buildAdjustVideoVolumePlan(ctx *core.ExecContext) (*FFmpegPlan, error) {
	videoURL, err := requiredStringParam(ctx.Params, "video_url")
	if err != nil {
		return nil, err
	}
	input, err := materializeLocalInput(ctx, videoURL)
	if err != nil {
		return nil, err
	}
	if !hasAudioStream(input.LocalPath) {
		return nil, fmt.Errorf("adjust-video-volume 需要输入视频包含音频流")
	}
	volume, hasVolume, err := optionalFloatParam(ctx.Params, "volume")
	if err != nil {
		return nil, err
	}
	if !hasVolume {
		volume = 1
	}
	if volume < 0 || volume > 4 {
		return nil, fmt.Errorf("volume 取值范围为 0 到 4")
	}
	out, err := outputPath(ctx, "adjust-video-volume", ".mp4")
	if err != nil {
		return nil, err
	}
	if volume == 1 {
		args := []string{"-y", "-hide_banner", "-i", input.LocalPath, "-map", "0:v:0", "-map", "0:a:0", "-c", "copy", out}
		return &FFmpegPlan{Args: args, Result: videoResponse("adjust-video-volume", out, []LocalInputRef{input}, map[string]any{"volume": volume})}, nil
	}
	args := []string{"-y", "-hide_banner", "-i", input.LocalPath, "-filter_complex", "[0:a]volume=" + formatFloat(volume) + "[outa]", "-map", "0:v:0", "-map", "[outa]", "-c:v", "copy", "-c:a", "aac", "-b:a", "128k", out}
	return &FFmpegPlan{Args: args, Result: videoResponse("adjust-video-volume", out, []LocalInputRef{input}, map[string]any{"volume": volume})}, nil
}

func buildFadeAudioPlan(ctx *core.ExecContext) (*FFmpegPlan, error) {
	audioURL, err := requiredStringParam(ctx.Params, "audio_url")
	if err != nil {
		return nil, err
	}
	input, err := materializeLocalInput(ctx, audioURL)
	if err != nil {
		return nil, err
	}
	fadeIn, hasFadeIn, err := optionalFloatParam(ctx.Params, "fade_in_duration")
	if err != nil {
		return nil, err
	}
	if !hasFadeIn {
		fadeIn = 1
	}
	fadeOut, hasFadeOut, err := optionalFloatParam(ctx.Params, "fade_out_duration")
	if err != nil {
		return nil, err
	}
	if !hasFadeOut {
		fadeOut = 1
	}
	if fadeIn < 0 {
		return nil, fmt.Errorf("fade_in_duration 必须大于等于 0")
	}
	if fadeOut < 0 {
		return nil, fmt.Errorf("fade_out_duration 必须大于等于 0")
	}
	out, err := outputPath(ctx, "fade-audio", ".mp3")
	if err != nil {
		return nil, err
	}
	if fadeIn == 0 && fadeOut == 0 {
		args := []string{"-y", "-hide_banner", "-i", input.LocalPath, "-vn", "-c:a", "libmp3lame", "-b:a", "128k", out}
		return &FFmpegPlan{Args: args, Result: audioResponse("fade-audio", out, []LocalInputRef{input}, map[string]any{"fade_in_duration": fadeIn, "fade_out_duration": fadeOut})}, nil
	}
	duration, err := mediaDuration(input.LocalPath)
	if err != nil {
		return nil, err
	}
	filter := audioFadeFilter(fadeIn, fadeOut, duration)
	args := []string{"-y", "-hide_banner", "-i", input.LocalPath, "-filter_complex", "[0:a]" + filter + "[outa]", "-map", "[outa]", "-vn", "-c:a", "libmp3lame", "-b:a", "128k", out}
	return &FFmpegPlan{Args: args, Result: audioResponse("fade-audio", out, []LocalInputRef{input}, map[string]any{"fade_in_duration": fadeIn, "fade_out_duration": fadeOut})}, nil
}

func buildAdjustAudioSpeedPlan(ctx *core.ExecContext) (*FFmpegPlan, error) {
	audioURL, err := requiredStringParam(ctx.Params, "audio_url")
	if err != nil {
		return nil, err
	}
	input, err := materializeLocalInput(ctx, audioURL)
	if err != nil {
		return nil, err
	}
	speed, hasSpeed, err := optionalFloatParam(ctx.Params, "speed")
	if err != nil {
		return nil, err
	}
	if !hasSpeed {
		speed = 1
	}
	if speed < 0.1 || speed > 4 {
		return nil, fmt.Errorf("speed 取值范围为 0.1 到 4")
	}
	out, err := outputPath(ctx, "adjust-audio-speed", ".m4a")
	if err != nil {
		return nil, err
	}
	if speed == 1 {
		args := []string{"-y", "-hide_banner", "-i", input.LocalPath, "-vn", "-c:a", "aac", "-b:a", "128k", out}
		return &FFmpegPlan{Args: args, Result: audioResponse("adjust-audio-speed", out, []LocalInputRef{input}, map[string]any{"speed": speed})}, nil
	}
	args := []string{"-y", "-hide_banner", "-i", input.LocalPath, "-filter_complex", "[0:a]" + atempoChain(speed) + "[outa]", "-map", "[outa]", "-vn", "-c:a", "aac", "-b:a", "128k", out}
	return &FFmpegPlan{Args: args, Result: audioResponse("adjust-audio-speed", out, []LocalInputRef{input}, map[string]any{"speed": speed})}, nil
}

func buildMixAudioPlan(ctx *core.ExecContext) (*FFmpegPlan, error) {
	urls, err := requiredStringListParam(ctx.Params, "audio_urls")
	if err != nil {
		return nil, err
	}
	if len(urls) > 100 {
		return nil, fmt.Errorf("audio_urls 最多支持 100 个元素")
	}
	inputs := make([]LocalInputRef, 0, len(urls))
	for _, url := range urls {
		input, err := materializeLocalInput(ctx, url)
		if err != nil {
			return nil, err
		}
		inputs = append(inputs, input)
	}
	out, err := outputPath(ctx, "mix-audio", ".mp3")
	if err != nil {
		return nil, err
	}
	args := []string{"-y", "-hide_banner"}
	filterInputs := make([]string, 0, len(inputs))
	for index, input := range inputs {
		args = append(args, "-i", input.LocalPath)
		filterInputs = append(filterInputs, fmt.Sprintf("[%d:a]", index))
	}
	filter := strings.Join(filterInputs, "") + fmt.Sprintf("amix=inputs=%d:duration=longest:dropout_transition=0[outa]", len(inputs))
	args = append(args, "-filter_complex", filter, "-map", "[outa]", "-vn", "-c:a", "libmp3lame", "-b:a", "128k", out)
	return &FFmpegPlan{Args: args, Result: audioResponse("mix-audio", out, inputs, map[string]any{"input_count": len(inputs)})}, nil
}

func buildProbeAudioMetadataResult(ctx *core.ExecContext) (map[string]any, error) {
	audioURL, err := requiredStringParam(ctx.Params, "audio_url")
	if err != nil {
		return nil, err
	}
	input, err := materializeLocalInput(ctx, audioURL)
	if err != nil {
		return nil, err
	}
	metadata, err := probeAudioMetadata(input.LocalPath)
	if err != nil {
		return nil, err
	}
	md5Value, _ := fileMD5(input.LocalPath)
	formatMeta := map[string]any{
		"md5":       nullableString(md5Value),
		"container": nullableString(metadata.Format.FormatName),
		"bitrate":   nullableFloatString(metadata.Format.BitRate),
		"duration":  nullableFloatString(metadata.Format.Duration),
		"size":      nullableFloatString(metadata.Format.Size),
	}
	var audioStreamMeta any
	if len(metadata.Streams) > 0 {
		stream := metadata.Streams[0]
		audioStreamMeta = map[string]any{
			"codec":       nullableString(stream.CodecName),
			"duration":    nullableFloatString(stream.Duration),
			"sample_rate": nullableFloatString(stream.SampleRate),
			"bitrate":     nullableFloatString(stream.BitRate),
			"channels":    nullableInt(stream.Channels),
		}
	}
	return map[string]any{
		"format_meta":       formatMeta,
		"audio_stream_meta": audioStreamMeta,
	}, nil
}

func buildProbeVideoMetadataResult(ctx *core.ExecContext) (map[string]any, error) {
	videoURL, err := requiredStringParam(ctx.Params, "video_url")
	if err != nil {
		return nil, err
	}
	input, err := materializeLocalInput(ctx, videoURL)
	if err != nil {
		return nil, err
	}
	metadata, err := probeMediaMetadata(input.LocalPath)
	if err != nil {
		return nil, err
	}
	md5Value, _ := fileMD5(input.LocalPath)
	formatMeta := map[string]any{
		"md5":       nullableString(md5Value),
		"container": nullableString(metadata.Format.FormatName),
		"bitrate":   nullableFloatString(metadata.Format.BitRate),
		"duration":  nullableFloatString(metadata.Format.Duration),
		"size":      nullableFloatString(metadata.Format.Size),
	}
	var videoStreamMeta any
	var audioStreamMeta any
	for _, stream := range metadata.Streams {
		switch stream.CodecType {
		case "video":
			if videoStreamMeta == nil {
				videoStreamMeta = map[string]any{
					"codec":         nullableString(stream.CodecName),
					"width":         nullableInt(stream.Width),
					"height":        nullableInt(stream.Height),
					"duration":      nullableFloatString(stream.Duration),
					"bitrate":       nullableFloatString(stream.BitRate),
					"fps":           nullableFrameRate(stream.AvgFrameRate, stream.RFrameRate),
					"dynamic_range": dynamicRange(stream.ColorTransfer, stream.ColorPrimaries, stream.ColorSpace),
				}
			}
		case "audio":
			if audioStreamMeta == nil {
				audioStreamMeta = map[string]any{
					"codec":       nullableString(stream.CodecName),
					"duration":    nullableFloatString(stream.Duration),
					"sample_rate": nullableFloatString(stream.SampleRate),
					"bitrate":     nullableFloatString(stream.BitRate),
					"channels":    nullableInt(stream.Channels),
				}
			}
		}
	}
	return map[string]any{
		"format_meta":       formatMeta,
		"video_stream_meta": videoStreamMeta,
		"audio_stream_meta": audioStreamMeta,
	}, nil
}

func buildMatteGreenscreenVideoResult(ctx *core.ExecContext) (map[string]any, error) {
	videoURL, err := requiredStringParam(ctx.Params, "video_url")
	if err != nil {
		return nil, err
	}
	input, err := materializeLocalInput(ctx, videoURL)
	if err != nil {
		return nil, err
	}
	format := strings.ToUpper(valueOrDefault(ctx.Params, "format", "MOV"))
	if format != "MOV" {
		return nil, fmt.Errorf("matte-greenscreen-video 本地模式仅支持 --format MOV；WEBM 透明输出请使用 cloud 模式")
	}
	out, err := outputPath(ctx, "matte-greenscreen-video", ".mov")
	if err != nil {
		return nil, err
	}
	args := []string{
		"-y",
		"-hide_banner",
		"-i", input.LocalPath,
		"-vf", "chromakey=0x00ff00:0.18:0.08,format=yuva444p10le",
		"-map", "0:v:0",
		"-an",
		"-c:v", "prores_ks",
		"-profile:v", "4",
		"-pix_fmt", "yuva444p10le",
		out,
	}
	if _, err := core.RunFFmpeg(args...); err != nil {
		return nil, err
	}
	result := map[string]any{"video_url": out}
	if duration, err := mediaDuration(out); err == nil {
		result["duration"] = duration
	}
	return result, nil
}

func buildExtractAudioPlan(ctx *core.ExecContext) (*FFmpegPlan, error) {
	videoURL, err := requiredStringParam(ctx.Params, "video_url")
	if err != nil {
		return nil, err
	}
	input, err := materializeLocalInput(ctx, videoURL)
	if err != nil {
		return nil, err
	}
	ext, codec, err := extractAudioSpec(ctx.Params)
	if err != nil {
		return nil, err
	}
	out, err := outputPath(ctx, "extract-audio", ext)
	if err != nil {
		return nil, err
	}
	format := strings.TrimPrefix(ext, ".")
	args := []string{"-y", "-hide_banner", "-i", input.LocalPath, "-map", "0:a:0", "-c:a", codec, out}
	return &FFmpegPlan{Args: args, Result: audioResponse("extract-audio", out, []LocalInputRef{input}, map[string]any{"format": format})}, nil
}

func buildFlipVideoPlan(ctx *core.ExecContext) (*FFmpegPlan, error) {
	videoURL, err := requiredStringParam(ctx.Params, "video_url")
	if err != nil {
		return nil, err
	}
	input, err := materializeLocalInput(ctx, videoURL)
	if err != nil {
		return nil, err
	}
	vertical, err := core.ParamBool(ctx.Params, "is_flip_vertical", false)
	if err != nil {
		return nil, err
	}
	horizontal, err := core.ParamBool(ctx.Params, "is_flip_horizontal", false)
	if err != nil {
		return nil, err
	}
	filters := []string{}
	if horizontal {
		filters = append(filters, "hflip")
	}
	if vertical {
		filters = append(filters, "vflip")
	}
	if len(filters) == 0 {
		filters = append(filters, "null")
	}
	out, err := outputPath(ctx, "flip-video", ".mp4")
	if err != nil {
		return nil, err
	}
	args := append(
		[]string{"-y", "-hide_banner", "-i", input.LocalPath, "-filter_complex", "[0:v]" + strings.Join(filters, ",") + "[outv]", "-map", "[outv]", "-map", "0:a:0?", "-c:a", "aac", "-b:a", "128k"},
		append(h264OutputArgs("1200k"), out)...,
	)
	return &FFmpegPlan{Args: args, Result: videoResponse("flip-video", out, []LocalInputRef{input}, map[string]any{"is_flip_vertical": vertical, "is_flip_horizontal": horizontal})}, nil
}

func buildAdjustVideoSpeedPlan(ctx *core.ExecContext) (*FFmpegPlan, error) {
	videoURL, err := requiredStringParam(ctx.Params, "video_url")
	if err != nil {
		return nil, err
	}
	input, err := materializeLocalInput(ctx, videoURL)
	if err != nil {
		return nil, err
	}
	speed, hasSpeed, err := optionalFloatParam(ctx.Params, "speed")
	if err != nil {
		return nil, err
	}
	if !hasSpeed {
		speed = 1
	}
	if speed < 0.1 || speed > 4 {
		return nil, fmt.Errorf("speed 取值范围为 0.1 到 4")
	}
	out, err := outputPath(ctx, "adjust-video-speed", ".mp4")
	if err != nil {
		return nil, err
	}
	videoExpr := "setpts=" + formatFloat(1/speed) + "*PTS"
	hasAudio := hasAudioStream(input.LocalPath)
	if hasAudio {
		audioExpr := atempoChain(speed)
		args := append(
			[]string{"-y", "-hide_banner", "-i", input.LocalPath, "-filter_complex", "[0:v]" + videoExpr + "[outv];[0:a]" + audioExpr + "[outa]", "-map", "[outv]", "-map", "[outa]", "-c:a", "aac", "-b:a", "128k"},
			append(h264OutputArgs("1200k"), out)...,
		)
		return &FFmpegPlan{Args: args, Result: videoResponse("adjust-video-speed", out, []LocalInputRef{input}, map[string]any{"speed": speed})}, nil
	}
	args := append(
		[]string{"-y", "-hide_banner", "-i", input.LocalPath, "-filter_complex", "[0:v]" + videoExpr + "[outv]", "-map", "[outv]"},
		append(h264OutputArgs("1200k"), "-an", out)...,
	)
	return &FFmpegPlan{Args: args, Result: videoResponse("adjust-video-speed", out, []LocalInputRef{input}, map[string]any{"speed": speed})}, nil
}

func hasAudioStream(filePath string) bool {
	raw, err := core.RunFFprobe(
		"-v", "error",
		"-select_streams", "a",
		"-show_entries", "stream=codec_type",
		"-of", "csv=p=0",
		filePath,
	)
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(raw)) == "audio"
}

func mediaDuration(filePath string) (float64, error) {
	info, err := probeMediaInfo(filePath)
	if err != nil {
		return 0, err
	}
	duration, err := strconv.ParseFloat(strings.TrimSpace(info.Format.Duration), 64)
	if err != nil || duration <= 0 {
		return 0, fmt.Errorf("无法读取输入媒体时长")
	}
	return duration, nil
}

func audioFadeFilter(fadeIn float64, fadeOut float64, duration float64) string {
	filters := []string{}
	if fadeIn > 0 {
		filters = append(filters, "afade=t=in:st=0:d="+formatFloat(fadeIn))
	}
	if fadeOut > 0 {
		start := duration - fadeOut
		if start < 0 {
			start = 0
		}
		filters = append(filters, "afade=t=out:st="+formatFloat(start)+":d="+formatFloat(fadeOut))
	}
	return strings.Join(filters, ",")
}

func probeAudioMetadata(filePath string) (ffprobeAudioMetadata, error) {
	var metadata ffprobeAudioMetadata
	raw, err := core.RunFFprobe(
		"-v", "error",
		"-select_streams", "a:0",
		"-show_entries", "format=format_name,bit_rate,duration,size:stream=codec_name,duration,sample_rate,bit_rate,channels",
		"-of", "json",
		filePath,
	)
	if err != nil {
		return metadata, err
	}
	if err := json.Unmarshal(raw, &metadata); err != nil {
		return metadata, err
	}
	return metadata, nil
}

func probeMediaMetadata(filePath string) (ffprobeMediaMetadata, error) {
	var metadata ffprobeMediaMetadata
	raw, err := core.RunFFprobe(
		"-v", "error",
		"-show_entries", "format=format_name,bit_rate,duration,size:stream=codec_type,codec_name,width,height,duration,bit_rate,avg_frame_rate,r_frame_rate,color_transfer,color_primaries,color_space,sample_rate,channels",
		"-of", "json",
		filePath,
	)
	if err != nil {
		return metadata, err
	}
	if err := json.Unmarshal(raw, &metadata); err != nil {
		return metadata, err
	}
	return metadata, nil
}

func fileMD5(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()
	hash := md5.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

func nullableString(value string) any {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	return value
}

func nullableFloatString(value string) any {
	value = strings.TrimSpace(value)
	if value == "" || strings.EqualFold(value, "N/A") {
		return nil
	}
	parsed, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return nil
	}
	return parsed
}

func nullableInt(value int) any {
	if value <= 0 {
		return nil
	}
	return value
}

func nullableFrameRate(primary string, fallback string) any {
	if value := parseFrameRate(primary); value != nil {
		return value
	}
	return parseFrameRate(fallback)
}

func parseFrameRate(value string) any {
	value = strings.TrimSpace(value)
	if value == "" || value == "0/0" || strings.EqualFold(value, "N/A") {
		return nil
	}
	parts := strings.Split(value, "/")
	if len(parts) == 2 {
		numerator, numErr := strconv.ParseFloat(parts[0], 64)
		denominator, denErr := strconv.ParseFloat(parts[1], 64)
		if numErr != nil || denErr != nil || denominator == 0 {
			return nil
		}
		return numerator / denominator
	}
	return nullableFloatString(value)
}

func dynamicRange(colorTransfer string, colorPrimaries string, colorSpace string) any {
	value := strings.ToLower(strings.Join([]string{colorTransfer, colorPrimaries, colorSpace}, " "))
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	if strings.Contains(value, "smpte2084") || strings.Contains(value, "arib-std-b67") || strings.Contains(value, "bt2020") {
		return "HDR"
	}
	return "SDR"
}

func atempoChain(speed float64) string {
	parts := []string{}
	for speed > 2.0 {
		parts = append(parts, "atempo=2.0")
		speed /= 2.0
	}
	for speed < 0.5 {
		parts = append(parts, "atempo=0.5")
		speed /= 0.5
	}
	parts = append(parts, "atempo="+formatFloat(speed))
	return strings.Join(parts, ",")
}

func buildAddSubtitleToVideoPlan(ctx *core.ExecContext) (*FFmpegPlan, error) {
	videoURL, err := requiredStringParam(ctx.Params, "video_url")
	if err != nil {
		return nil, err
	}
	input, err := materializeLocalInput(ctx, videoURL)
	if err != nil {
		return nil, err
	}
	var subtitleInput LocalInputRef
	var subtitleSource string
	if rawURL, ok, err := optionalStringParam(ctx.Params, "subtitle_url"); err != nil {
		return nil, err
	} else if ok {
		subtitleInput, err = materializeLocalInput(ctx, rawURL)
		if err != nil {
			return nil, err
		}
		subtitleSource = "subtitle_url"
	} else {
		subtitlePath, err := writeSubtitleObjectsAsSRT(ctx)
		if err != nil {
			return nil, err
		}
		subtitleInput = LocalInputRef{Original: "subtitles", LocalPath: subtitlePath}
		subtitleSource = "subtitles"
	}
	style := buildSubtitleForceStyle(ctx.Params)
	out, err := outputPath(ctx, "add-subtitle-to-video", ".mp4")
	if err != nil {
		return nil, err
	}
	args := append(
		[]string{"-y", "-hide_banner", "-i", input.LocalPath, "-vf", "subtitles=" + escapeFilterPath(subtitleInput.LocalPath) + ":force_style='" + style + "'", "-map", "0:v:0", "-map", "0:a:0?", "-c:a", "aac", "-b:a", "128k"},
		append(h264OutputArgs("1300k"), out)...,
	)
	return &FFmpegPlan{Args: args, Result: videoResponse("add-subtitle-to-video", out, []LocalInputRef{input, subtitleInput}, map[string]any{"subtitle_source": subtitleSource, "subtitle_pos_preset": valueOrDefault(ctx.Params, "subtitle_pos_preset", "bottom_center")})}, nil
}

func writeSubtitleObjectsAsSRT(ctx *core.ExecContext) (string, error) {
	raw, ok := ctx.Params["subtitles"]
	if !ok {
		return "", fmt.Errorf("subtitle_url 和 subtitles 至少需要提供一个")
	}
	items, ok := raw.([]any)
	if !ok || len(items) == 0 {
		return "", fmt.Errorf("subtitles 至少需要 1 个元素")
	}
	lines := []string{}
	for i, rawItem := range items {
		item, ok := rawItem.(map[string]any)
		if !ok {
			return "", fmt.Errorf("subtitles[%d] 必须是对象", i)
		}
		text, _ := item["subtitle_text"].(string)
		start, startOK := numberField(item, "start_time")
		end, endOK := numberField(item, "end_time")
		if strings.TrimSpace(text) == "" || !startOK || !endOK || end <= start {
			return "", fmt.Errorf("subtitles[%d] 需要合法 subtitle_text/start_time/end_time", i)
		}
		lines = append(lines, strconv.Itoa(i+1), formatSRTTime(start)+" --> "+formatSRTTime(end), text, "")
	}
	path := filepath.Join(ctx.TempDir, fmt.Sprintf("subtitles-%d.srt", time.Now().UnixNano()))
	if err := os.WriteFile(path, []byte(strings.Join(lines, "\n")), 0o600); err != nil {
		return "", err
	}
	return path, nil
}

func buildAddImageToVideoPlan(ctx *core.ExecContext) (*FFmpegPlan, error) {
	videoURL, err := requiredStringParam(ctx.Params, "video_url")
	if err != nil {
		return nil, err
	}
	imageURL, err := requiredStringParam(ctx.Params, "sub_image_url")
	if err != nil {
		return nil, err
	}
	video, err := materializeLocalInput(ctx, videoURL)
	if err != nil {
		return nil, err
	}
	image, err := materializeLocalInput(ctx, imageURL)
	if err != nil {
		return nil, err
	}
	out, err := outputPath(ctx, "add-image-to-video", ".mp4")
	if err != nil {
		return nil, err
	}
	overlayExpr := overlayPosition(ctx.Params)
	enableExpr := overlayEnable(ctx.Params)

	// 只有用户明确指定了 width/height 时才缩放图片，否则保持原始尺寸（与云端行为一致）
	var filterComplex string
	if hasOverlayScale(ctx.Params) {
		scaleExpr := overlayScale2Ref(ctx.Params)
		filterComplex = "[1:v][0:v]scale2ref=" + scaleExpr + "[wm][base];[base][wm]overlay=" + overlayExpr + enableExpr + "[outv]"
	} else {
		filterComplex = "[0:v][1:v]overlay=" + overlayExpr + enableExpr + "[outv]"
	}
	args := append(
		[]string{"-y", "-hide_banner", "-i", video.LocalPath, "-i", image.LocalPath, "-filter_complex", filterComplex, "-map", "[outv]", "-map", "0:a:0?", "-c:a", "aac", "-b:a", "128k"},
		append(h264OutputArgs("1300k"), out)...,
	)
	return &FFmpegPlan{Args: args, Result: videoResponse("add-image-to-video", out, []LocalInputRef{video, image}, map[string]any{"overlay_start_time": ctx.Params["start_time"], "overlay_end_time": ctx.Params["end_time"]})}, nil
}

func valueOrDefault(params map[string]any, key string, defaultValue string) string {
	value, ok := params[key].(string)
	if !ok || strings.TrimSpace(value) == "" {
		return defaultValue
	}
	return value
}

func numberField(params map[string]any, key string) (float64, bool) {
	switch typed := params[key].(type) {
	case float64:
		return typed, true
	case int:
		return float64(typed), true
	case json.Number:
		value, err := typed.Float64()
		return value, err == nil
	default:
		return 0, false
	}
}

func formatSRTTime(seconds float64) string {
	totalMilliseconds := int64(seconds * 1000)
	milliseconds := totalMilliseconds % 1000
	totalSeconds := totalMilliseconds / 1000
	secs := totalSeconds % 60
	totalMinutes := totalSeconds / 60
	mins := totalMinutes % 60
	hours := totalMinutes / 60
	return fmt.Sprintf("%02d:%02d:%02d,%03d", hours, mins, secs, milliseconds)
}

func buildSubtitleForceStyle(params map[string]any) string {
	fontSize := valueOrDefault(params, "subtitle_font_size", "50")
	fontColor := rgbaToASS(valueOrDefault(params, "subtitle_font_color", "#FFFFFFFF"))
	fontName := fontNameFor(valueOrDefault(params, "subtitle_font_type", "sy_black"))
	alignment := alignmentFor(valueOrDefault(params, "subtitle_pos_preset", "bottom_center"))
	return strings.Join([]string{"FontName=" + fontName, "FontSize=" + fontSize, "PrimaryColour=" + fontColor, "Alignment=" + alignment}, ",")
}

func rgbaToASS(rgba string) string {
	value := strings.TrimPrefix(strings.TrimSpace(rgba), "#")
	if len(value) != 8 {
		return "&H00FFFFFF"
	}
	rr := value[0:2]
	gg := value[2:4]
	bb := value[4:6]
	aa := value[6:8]
	return "&H" + aa + bb + gg + rr
}

func fontNameFor(fontType string) string {
	switch fontType {
	case "sy_black":
		return "Source Han Sans"
	case "pm_zhengdao":
		return "PangMenZhengDao"
	case "zhanku_kuaile":
		return "ZhanKu KuaiLe"
	default:
		return "Source Han Sans"
	}
}

func alignmentFor(preset string) string {
	switch preset {
	case "top_center":
		return "8"
	case "center":
		return "5"
	case "lower_third":
		return "2"
	default:
		return "2"
	}
}

func escapeFilterPath(path string) string {
	path = strings.ReplaceAll(path, "\\", "\\\\")
	path = strings.ReplaceAll(path, ":", "\\:")
	path = strings.ReplaceAll(path, "'", "\\'")
	return path
}

// hasOverlayScale 判断用户是否明确指定了图片缩放尺寸
func hasOverlayScale(params map[string]any) bool {
	w, wOk := params["sub_image_width"].(string)
	h, hOk := params["sub_image_height"].(string)
	return (wOk && strings.TrimSpace(w) != "") || (hOk && strings.TrimSpace(h) != "")
}

func overlayScale2Ref(params map[string]any) string {
	width := valueOrDefault(params, "sub_image_width", "10%")
	height := valueOrDefault(params, "sub_image_height", "5%")
	return "w=" + overlayDimension(width, "main_w") + ":h=" + overlayDimension(height, "main_h")
}

func overlayDimension(value string, base string) string {
	value = strings.TrimSpace(value)
	if strings.HasSuffix(value, "%") {
		ratio, err := strconv.ParseFloat(strings.TrimSuffix(value, "%"), 64)
		if err == nil {
			return base + "*" + formatFloat(ratio/100)
		}
	}
	if value == "" {
		return "-1"
	}
	return value
}

func overlayPosition(params map[string]any) string {
	x := overlayCoordinate(valueOrDefault(params, "sub_image_pos_x", "85%"), "W", "w")
	y := overlayCoordinate(valueOrDefault(params, "sub_image_pos_y", "90%"), "H", "h")
	return x + ":" + y
}

func overlayCoordinate(value string, base string, overlay string) string {
	value = strings.TrimSpace(value)
	if strings.HasSuffix(value, "%") {
		ratio, err := strconv.ParseFloat(strings.TrimSuffix(value, "%"), 64)
		if err == nil {
			// 与云端一致：百分比表示图片左上角在视频宽/高的绝对位置，超出部分自然截断
			return base + "*" + formatFloat(ratio/100)
		}
	}
	if value == "" {
		return "0"
	}
	return value
}

func overlayEnable(params map[string]any) string {
	start, hasStart := numberField(params, "start_time")
	end, hasEnd := numberField(params, "end_time")
	if hasStart && hasEnd {
		return ":enable='between(t," + formatFloat(start) + "," + formatFloat(end) + ")'"
	}
	if hasStart {
		return ":enable='gte(t," + formatFloat(start) + ")'"
	}
	if hasEnd {
		return ":enable='lte(t," + formatFloat(end) + ")'"
	}
	return ""
}
