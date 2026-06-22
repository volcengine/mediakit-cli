package generated

import "mediakit-cli/internal/local/core"

// Registrations returns all auto-generated local handlers.
func Registrations() []core.Registration {
	return []core.Registration{
		{
			Command: "fetch-file",
			Source:  "generated",
			Handler: &FetchFileHandler{},
		},
		{
			Command:      "extract-audio",
			Source:       "generated",
			Dependencies: []string{"ffmpeg", "libmp3lame"},
			Handler:      NewFFmpegHandler(buildExtractAudioPlan),
		},
		{
			Command:      "add-image-to-video",
			Source:       "generated",
			Dependencies: []string{"ffmpeg", "openh264", "libpng"},
			Handler:      NewFFmpegHandler(buildAddImageToVideoPlan),
		},
		{
			Command:      "add-subtitle-to-video",
			Source:       "generated",
			Dependencies: []string{"ffmpeg", "openh264", "libass", "libfreetype", "libfontconfig", "libfribidi", "libharfbuzz"},
			Handler:      NewFFmpegHandler(buildAddSubtitleToVideoPlan),
		},
		{
			Command:      "mux-audio-video",
			Source:       "generated",
			Dependencies: []string{"ffmpeg"},
			Handler:      NewFFmpegHandler(buildMuxAudioVideoPlan),
		},
		{
			Command:      "fade-video-audio",
			Source:       "generated",
			Dependencies: []string{"ffmpeg"},
			Handler:      NewFFmpegHandler(buildFadeVideoAudioPlan),
		},
		{
			Command:      "adjust-video-volume",
			Source:       "generated",
			Dependencies: []string{"ffmpeg"},
			Handler:      NewFFmpegHandler(buildAdjustVideoVolumePlan),
		},
		{
			Command:      "fade-audio",
			Source:       "generated",
			Dependencies: []string{"ffmpeg", "libmp3lame"},
			Handler:      NewFFmpegHandler(buildFadeAudioPlan),
		},
		{
			Command:      "adjust-audio-speed",
			Source:       "generated",
			Dependencies: []string{"ffmpeg"},
			Handler:      NewFFmpegHandler(buildAdjustAudioSpeedPlan),
		},
		{
			Command:      "mix-audio",
			Source:       "generated",
			Dependencies: []string{"ffmpeg", "libmp3lame"},
			Handler:      NewFFmpegHandler(buildMixAudioPlan),
		},
		{
			Command:      "probe-audio-metadata",
			Source:       "generated",
			Dependencies: []string{"ffprobe"},
			Handler:      core.HandlerFunc(buildProbeAudioMetadataResult),
		},
		{
			Command:      "probe-video-metadata",
			Source:       "generated",
			Dependencies: []string{"ffprobe"},
			Handler:      core.HandlerFunc(buildProbeVideoMetadataResult),
		},
		{
			Command:      "matte-greenscreen-video",
			Source:       "generated",
			Dependencies: []string{"ffmpeg", "prores_ks"},
			Handler:      core.HandlerFunc(buildMatteGreenscreenVideoResult),
		},
		{
			Command:      "concat-video",
			Source:       "generated",
			Dependencies: []string{"ffmpeg", "demuxer"},
			Handler:      NewFFmpegHandler(buildConcatVideoPlan),
		},
		{
			Command:      "flip-video",
			Source:       "generated",
			Dependencies: []string{"ffmpeg", "openh264"},
			Handler:      NewFFmpegHandler(buildFlipVideoPlan),
		},
		{
			Command:      "trim-video",
			Source:       "generated",
			Dependencies: []string{"ffmpeg"},
			Handler:      NewFFmpegHandler(buildTrimVideoPlan),
		},
		{
			Command:      "adjust-video-speed",
			Source:       "generated",
			Dependencies: []string{"ffmpeg", "openh264"},
			Handler:      NewFFmpegHandler(buildAdjustVideoSpeedPlan),
		},
		{
			Command:      "concat-audio",
			Source:       "generated",
			Dependencies: []string{"ffmpeg", "demuxer"},
			Handler:      NewFFmpegHandler(buildConcatAudioPlan),
		},
		{
			Command:      "trim-audio",
			Source:       "generated",
			Dependencies: []string{"ffmpeg"},
			Handler:      NewFFmpegHandler(buildTrimAudioPlan),
		},
	}
}
