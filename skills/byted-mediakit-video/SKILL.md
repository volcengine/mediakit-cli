---
name: byted-mediakit-video
version: "1.0.0"
license: "MIT"
description: "视频处理，涵盖视频画质增强、视频理解、字幕擦除等能力。包含能力：enhance-video, erase-video-subtitle-pro。当用户需要使用 video 域的 MediaKit CLI 能力时触发。"
permissions:
  - shell
metadata:
  requires:
    bins: ["mediakit-cli"]
  cliHelp: "mediakit-cli video --help"
  product: mediakit-cli/skills
  domain: video
  capability_count: 2
---
# Video Skills

## 前置说明

开始前必须先读取 `./reference/shared.md` 的内容，其中包含前置检查、异步任务机制、结果查询等说明。

## 工具列表

| 工具 | 说明 | 参数声明 | 参考文档 |
|------|------|----------|----------|
| enhance-video | 画质增强：针对 AIGC / UGC / 短剧 / 教育 / 游戏 / 老片修复等场景，提供画质提升 + 超分增强一站式解决方案。依托 AI MediaKit 智能媒体处理引擎，融合视频内容理解、画质指标智能决策、多维度增强原子算法，实现画质的全面优化。 支持格式：主流视频格式如mp4、flv、ts、avi、mov、wmv、mkv。 使用限制：单文件大小不超过100G。 | `video_url:string, scene?:string, tool_version?:string, resolution?:string, resolution_limit?:integer, fps?:number, callback_args?:string, client_token?:string` | [reference/enhance-video.md](reference/enhance-video.md) |
| erase-video-subtitle-pro | 针对视频中的字幕，实现高质量的无痕擦除，最大程度的还原视频画面。 支持格式：主流视频格式如mp4、flv、ts、avi、mov、wmv、mkv。 | `video_url:string, mode?:string, output_encode_mode?:string, erase_ratio_location?:array<object{top_left_x:number, top_left_y:number, bottom_right_x:number, bottom_right_y:number}>, callback_args?:string, client_token?:string` | [reference/erase-video-subtitle-pro.md](reference/erase-video-subtitle-pro.md) |
