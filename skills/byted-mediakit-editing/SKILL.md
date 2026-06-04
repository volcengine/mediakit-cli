---
name: byted-mediakit-editing
version: "1.0.0"
license: "MIT"
description: "音视频剪辑，涵盖音视频拼接、裁剪、合成等能力。包含能力：add-image-to-video, add-subtitle-to-video, adjust-video-speed, concat-audio, concat-video, extract-audio, flip-video, image-to-video, mux-audio-video, trim-audio, trim-video。当用户需要使用 editing 域的 MediaKit CLI 能力时触发。"
permissions:
  - shell
metadata:
  requires:
    bins: ["mediakit-cli"]
  cliHelp: "mediakit-cli editing --help"
  product: mediakit-cli/skills
  domain: editing
  capability_count: 11
---
# Editing Skills

## 前置说明

开始前必须先读取 `./reference/shared.md` 的内容，其中包含前置检查、异步任务机制、结果查询等说明。

## 工具列表

| 工具 | 说明 | 参数声明 | 参考文档 |
|------|------|----------|----------|
| add-image-to-video | 视频加图片，可用作加图片水印。 | `video_url:string, sub_image_url:string, sub_image_height?:string, sub_image_width?:string, sub_image_pos_x?:string, sub_image_pos_y?:string, start_time?:number, end_time?:number, callback_args?:string, client_token?:string` | [reference/add-image-to-video.md](reference/add-image-to-video.md) |
| add-subtitle-to-video | 将字幕文件或文本内容，以指定样式压制到视频画面中，生成带内嵌字幕的新视频。 | `video_url:string, subtitle_url?:string, subtitles?:array<object{subtitle_text:string, start_time:number, end_time:number}>, subtitle_pos_preset?:string, subtitle_font_size?:integer, subtitle_font_color?:string, subtitle_font_type?:string, callback_args?:string, client_token?:string` | [reference/add-subtitle-to-video.md](reference/add-subtitle-to-video.md) |
| adjust-video-speed | 调整视频的播放倍速，实现快放或慢放效果。 | `video_url:string, speed?:number, callback_args?:string, client_token?:string` | [reference/adjust-video-speed.md](reference/adjust-video-speed.md) |
| concat-audio | 拼接多个音频片段。 | `audio_urls:array<string>, callback_args?:string, client_token?:string` | [reference/concat-audio.md](reference/concat-audio.md) |
| concat-video | 拼接多个视频片段，支持添加转场效果。 | `video_urls:array<string>, transitions?:array<string>, callback_args?:string, client_token?:string` | [reference/concat-video.md](reference/concat-video.md) |
| extract-audio | 将视频文件中的音频流分离并保存为独立的音频文件。 | `video_url:string, format?:string, callback_args?:string, client_token?:string` | [reference/extract-audio.md](reference/extract-audio.md) |
| flip-video | 对视频画面进行上下或左右镜像翻转。 | `video_url:string, is_flip_vertical?:boolean, is_flip_horizontal?:boolean, callback_args?:string, client_token?:string` | [reference/flip-video.md](reference/flip-video.md) |
| image-to-video | 多张图片生成动画视频。 | `images:array<object{image_url:string, duration?:number, animation_type?:string, animation_in?:number, animation_out?:number}>, transitions?:array<string>, callback_args?:string, client_token?:string` | [reference/image-to-video.md](reference/image-to-video.md) |
| mux-audio-video | 音视频合成。 | `video_url:string, audio_url:string, is_audio_reserve?:boolean, is_video_audio_sync?:boolean, sync_mode?:string, sync_method?:string, callback_args?:string, client_token?:string` | [reference/mux-audio-video.md](reference/mux-audio-video.md) |
| trim-audio | 按起止时间点（秒级）裁剪音频，生成新片段。 | `audio_url:string, start_time?:number, end_time?:number, callback_args?:string, client_token?:string` | [reference/trim-audio.md](reference/trim-audio.md) |
| trim-video | 按起止时间点裁剪视频，生成新片段。 | `video_url:string, start_time?:number, end_time?:number, callback_args?:string, client_token?:string` | [reference/trim-video.md](reference/trim-video.md) |
