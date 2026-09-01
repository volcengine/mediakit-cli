# AI MediaKit CLI

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Go Version](https://img.shields.io/badge/go-%3E%3D1.22-blue.svg)](https://go.dev/)
[![npm version](https://img.shields.io/npm/v/@volcengine/mediakit-cli.svg)](https://www.npmjs.com/package/@volcengine/mediakit-cli)

[中文版](./README.md) | [English](./README.en.md)

The official Mediakit CLI — an FFmpeg-compatible command surface. The same command can run FFmpeg locally for editing operations such as trimming, concatenation, subtitling, mixing, and audio extraction, or switch to the cloud with a single flag to invoke AI capabilities that FFmpeg cannot deliver — quality enhancement, subtitle erasure, ASR, OCR, storyline analysis, and more. It already covers atomic capabilities across video, image, and audio modalities plus 5 AI Agent [Skills](./skills/), with 100+ audio/video atomic capabilities planned.

[Installation](#installation--quick-start) · [AI Agent Skills](#agent-skills) · [Authentication](#authentication) · [Command Structure](#command-structure) · [Advanced Usage](#advanced-usage) · [License](#license)

## Why choose mediakit-cli?

- **Comprehensive capability matrix**: spans video, image, and audio modalities, from low-level processing such as trimming / concatenation / subtitling to high-level understanding such as quality enhancement, subtitle erasure, ASR, OCR, and storyline analysis — a single command covers the full pipeline from preprocessing to final output.
- **FFmpeg-compatible, seamless migration**: local mode is built on `ffmpeg` / `ffprobe`, covering common capabilities such as trimming, concatenation, image overlay, subtitle overlay, speed adjustment, volume adjustment, flipping, fade in/out, mixing, audio/video composition, audio extraction, green-screen keying, and metadata probing — aligned with FFmpeg command intuition. Complex / AI capabilities such as filters, image-to-video, and concatenation transitions are handled in the cloud.
- **Cloud is faster and more powerful**: append `--cloud` to the same command to unlock capabilities FFmpeg cannot deliver — quality enhancement / generative quality restoration, subtitle erasure (standard / fine-grained), ASR, video OCR, highlight clipping (short drama / mini-game), storyline analysis, scene segmentation, green-screen / portrait keying, and other AI atomic capabilities. Cloud elastic compute provides second-level concurrency.
- **One command, two modes**: `--local` / `--cloud` can be switched per command; local mode is zero-cost and cloud provides elastic compute, complementing each other. They share the same parameters and `--schema`, so Agents / scripts can switch with zero modification.
- **Cost-effective processing**: leverages cloud elastic resource scheduling and off-peak batch processing strategies to provide highly competitive pricing for large batches of media tasks, significantly reducing overall token consumption and operational cost for AI applications.

## Features

| Domain               | Capabilities                                                                                                                                                                                                                                                                                                                                                                  | Runtime            |
| -------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------------------ |
| 🎬 **Editing** (23)  | 视频加图片 · 视频加字幕 · 音频调速 · 视频调速 · 调整视频音量 · 视频添加运镜 · 视频添加滤镜 · 音频拼接 · 视频拼接 · 视频画面裁剪 · 视频截取动图 · 提取音频 · 音频声音淡入淡出 · 视频声音淡入淡出 · 视频画面翻转 · 图片转视频 · 音频混合 · 视频加音频 · 视频画面旋转 · 视频画面拼接 · 文字生成滚屏视频 · 音频裁剪 · 视频裁剪 | Cloud **or** Local              |
| 🎚️ **Audio** (4)  | 语音端点识别 · 音频元信息获取 · 人声背景音分离 · 音频转码 | Cloud              |
| 🖼️ **Image AI** (21)  | 添加图文水印 · 图像调整 · 图像压缩 · 图像裁剪 · 图像画质增强 · 图像擦除修复 · 图像画质评估 · 图像人脸打码 · 图像翻转 · 图像高斯模糊 · 图像文字识别OCR · 图像负片 · 图像打码 · 图像元信息获取 · 图像背景移除 · 图像缩放 · 图像旋转 · 圆角矩形 · 图像锐化 · 集智瘦身 · 图像智能裁剪 | Cloud              |
| 🎥 **Video AI** (30)  | 添加视频暗水印 · 高光片段提取 · 剧情故事线分析 · 语音转字幕（ASR) · 视频画质检测（VQScore） · 解说视频生成 · 解说视频生成（短剧行业模型） · 剧本还原 · 画质增强 · 视频画质增强极速版 · 生成式画质增强 · 字幕擦除（标准版） · 精细化字幕擦除 · 视频抽帧 · 提取视频暗水印 · 视频人脸打码 · 视频人脸融合 · 高光智剪-短剧 · 高光智剪-小游戏 · 高光智剪-影视拆条 · 极智超清 · 视频绿幕抠图 · 视频人像抠图 · 视频元信息获取 · 视频转封装 · 场景切分 · 智能语义切片 · 视频转码 · 视频识别字幕（OCR） · 视频理解智能策略 | Cloud              |
| 🔧 **Common** (2)    | Async task query · Remote file fetch                                                                                                                                                                                                                                                                                                                                          | Local / Cloud              |
| 🚧 **Coming soon**   | Video translation · Narration generation · Manga-to-animation (rolling out)                                                                                                                                                                                                                                                                                                   | Cloud              |

## Installation & Quick Start

### Requirements

Before you begin, make sure you have:

- Node.js `>=18` (`npm` / `npx`)

- Local mode: `ffmpeg` `5.1.x` and `ffprobe`

### Quick Start (Human Users)

#### Installation

Choose **one** of the following methods:

**Option 1 — one-click install:**

```bash
npx @volcengine/mediakit-cli install -y
```

**Option 2 — build from source:**

Requires Go `v1.22`+.

```bash
git clone https://github.com/volcengine/mediakit-cli.git
cd mediakit-cli
make build          # Artifact: .mediakit/build/dev/mediakit-cli

# Install AI Agent Skills from local skills directory (required)
npx -y skills add ./skills -g -y
```

#### Configuration & Usage

```bash
# 1. Initialize configuration (interactive wizard)
mediakit-cli init

# 2. Environment self-check (cloud connectivity, local dependencies, install suggestions)
mediakit-cli doctor

# 3. Local editing (synchronous, no API Key needed): run FFmpeg locally to trim
mediakit-cli --local editing trim-video --video-url ./in.mp4 --start-time 3 --end-time 8

# 4. Cloud AI (async): enhance a video to 1080p, then poll for the final result
mediakit-cli --cloud video enhance-video --video-url <url> --resolution 1080p
mediakit-cli shared query-task --task-id <task_id> --poll-complete
```

### Quick Start (AI Agent)

> The following steps are designed for AI Agents and support fully unattended workflows.

**Step 1 — Install**

```bash
npx @volcengine/mediakit-cli install -y
```

**Step 2 — Non-interactive initialization (`--yes` mode)**

```bash
# Get an API Key at: https://console.volcengine.com/imp/ai-mediakit/settings
mediakit-cli init \
  --mode cloud-first \
  --api-key <your-api-key> \
  --yes
```

**Step 3 — Verify**

```bash
mediakit-cli doctor
mediakit-cli version
```

## Agent Skills

| Skill                    | Description                                                                                                                                                                                                                                                                                                                                                                |
| ------------------------ | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `byted-mediakit-shared`  | Common capabilities: task query — required by all other skills                                                                                                                                                                                                                                                                                                                                                                                                                                                                                         |
| `byted-mediakit-editing` | Editing: video image overlay, video subtitle overlay, audio speed, video speed, video volume adjust, camera motion, video filter, audio concat, video concat, video crop, animated image extract, extract audio, audio fade in/out, video audio fade in/out, video flip, image to video, audio mix, mux audio/video, video rotate, video stitch, text to scrolling video, audio trim, video trim |
| `byted-mediakit-audio`   | Audio: voice activity detection, audio metadata, voice / background separation, audio transcode                                                                                                                                                                                                                                                                                                                                                                                                                                                      |
| `byted-mediakit-image`   | Image AI: image watermark, image color adjust, image compress, image crop, image quality enhancement, image erase & inpaint, image quality assessment, face blur, image flip, gaussian blur, image OCR, image invert, image mosaic, image metadata, background removal, image resize, image rotate, round corner, image sharpen, slim image, smart crop |
| `byted-mediakit-video`   | Video AI: invisible watermark add/extract, highlight extraction, storyline analysis, ASR subtitles, video quality assessment, drama recap, drama recap (vertical), drama script, quality enhancement, fast enhancement, generative enhancement, subtitle erasure, fine-grained subtitle erasure, frame extract, face blur, face swap, highlight clipping (microdrama / minigame / movie), martencode, green-screen keying, portrait keying, video metadata, remux, scene segmentation, semantic segment, transcode, video OCR, video understand router |

## Authentication

`mediakit-cli` uses minimal authentication: just an API Key — no OAuth / STS / IAM role configuration required.

```bash
# Option A: choose a storage method during init (config / shell / export)
mediakit-cli init --api-key <your-api-key> --credential-store config --yes

# Option B: inject temporarily via environment variables
export MEDIAKIT_API_KEY=<your-api-key>
export MEDIAKIT_OUTPUT_PATH=<optional-custom-endpoint>
```

| Environment variable   | Description                                                                                        |
| ---------------------- | -------------------------------------------------------------------------------------------------- |
| `MEDIAKIT_API_KEY`     | Cloud API Key ([get it from the console](https://console.volcengine.com/imp/ai-mediakit/settings)) |
| `MEDIAKIT_OUTPUT_PATH` | Local mode output directory, defaults to `~/.mediakit/temp`                                        |

## Command Structure

```
mediakit-cli [--cloud|--local] <domain> <tool> [flags]
```

- **Two modes, one command surface**: `--cloud` uses cloud elastic compute (asynchronously returns a `task_id`); `--local` uses local FFmpeg (synchronous, zero cost). Default is `cloud-first`, and it can be overridden per command with `--cloud` / `--local`.
- **Output**: cloud results are returned as URLs; local results land in `~/.mediakit/temp` (override with `--output-path` or `MEDIAKIT_OUTPUT_PATH`).

System commands:

| Command                                      | Description                                                                                 |
| -------------------------------------------- | ------------------------------------------------------------------------------------------- |
| `mediakit-cli init [--yes]`                  | Initialize configuration, interactive or non-interactive (Agent-friendly)                   |
| `mediakit-cli doctor`                        | Check cloud connectivity, local dependencies, and install suggestions                       |
| `mediakit-cli config`                        | View / modify configuration                                                                 |
| `mediakit-cli version [--check]`             | Show version; `--check` compares against the latest npm release                             |
| `mediakit-cli update [--check]`              | Update the CLI and Skills via `npm install -g`; `--check` only checks without installing    |
| `mediakit-cli --domains`                     | List all domains                                                                            |
| `mediakit-cli --help-full`                   | List the full capability index                                                              |
| `mediakit-cli <domain> <tool> --schema`      | Output the JSON Schema for the capability (Mode / Async / polling command metadata)         |
| `mediakit-cli shared query-task --task-id X` | Query an async task; add `--poll-complete` to poll until terminal state                     |

## Advanced Usage

### Schema Introspection

Every capability command supports `--schema`, which outputs the input / output schema plus Mode and Async information for Agents to discover tool capabilities dynamically:

```bash
mediakit-cli video enhance-video --schema
mediakit-cli --local editing trim-video --schema
```

### Local Mode Output Naming

Local mode output files are named by the following priority:

1. Explicit `--output-path` with a complete file path (including extension) → used directly
2. Input filename available → `{original_filename}_{tool_name}.{ext}`; if a file with the same name already exists, a 6-digit random number is appended
3. No input filename → `{tool_name}-{timestamp}.{ext}`

## License

This project is open-sourced under the **MIT License**.

At runtime this software calls MediaKit cloud APIs. Using those APIs is subject to the following agreements:

- [Video Cloud Services Specific Terms](https://www.volcengine.com/docs/6448/79646?lang=zh)
- [Intelligent Processing Service Billing Rules](https://www.volcengine.com/docs/6448/104992?lang=zh)
- [Intelligent Processing Service SLA](https://www.volcengine.com/docs/6448/79648?lang=zh)
