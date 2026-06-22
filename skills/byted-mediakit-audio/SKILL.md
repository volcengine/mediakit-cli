---
name: byted-mediakit-audio
version: "1.0.0"
license: "MIT"
description: "音频处理，涵盖音频处理和增强、内容理解等能力。包含能力：separate-voice, probe-audio-metadata。当用户需要使用 audio 域的 MediaKit CLI 能力时触发。"
permissions:
  - shell
metadata:
  requires:
    bins: ["mediakit-cli"]
  cliHelp: "mediakit-cli audio --help"
  product: mediakit-cli/skills
  domain: audio
  capability_count: 2
---
# Audio Skills

## 前置说明

开始前必须先读取 `./reference/shared.md` 的内容，其中包含前置检查、异步任务机制、结果查询等说明。

## 工具列表

| 工具 | 说明 | 参数声明 | 参考文档 |
|------|------|----------|----------|
| separate-voice | 将音频中的人声与背景音精准分离，输出为两个独立的音轨文件 | `video_url?:string, audio_url?:string, callback_args?:string, client_token?:string` | [reference/separate-voice.md](reference/separate-voice.md) |
| probe-audio-metadata | 获取指定音频的详细元信息，输出容器层信息与音频流元信息 | `audio_url:string, callback_args?:string, client_token?:string` | [reference/probe-audio-metadata.md](reference/probe-audio-metadata.md) |
