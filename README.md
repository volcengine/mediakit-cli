# AI MediaKit CLI

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Go Version](https://img.shields.io/badge/go-%3E%3D1.22-blue.svg)](https://go.dev/)
[![npm version](https://img.shields.io/npm/v/@volcengine/mediakit-cli.svg)](https://www.npmjs.com/package/@volcengine/mediakit-cli)

[中文版](./README.md) | [English](./README.en.md)

Mediakit 官方 CLI —— 兼容 FFmpeg 的命令面，同一条命令既能在本地跑 FFmpeg 完成裁剪 / 拼接 / 加字幕 / 混音 / 提取音频等剪辑操作，也能一键切到云端调用画质增强、字幕擦除、ASR、OCR、剧情线分析等 FFmpeg 做不到的 AI 能力。目前已覆盖视频、图像、音频等多模态原子能力和 5 个 AI Agent [Skills](./skills/)，未来预计提供 100+ 音视频原子能力。

[安装](#安装与快速开始) · [AI Agent Skills](#agent-skills) · [鉴权](#鉴权) · [命令结构](#命令结构) · [进阶用法](#进阶用法) · [许可证](#许可证)

## 为什么选 mediakit-cli？

- **完备的能力矩阵**：横跨视频、图像、音频三大模态，从裁剪 / 拼接 / 加字幕等底层处理，到画质增强、字幕擦除、ASR、OCR、剧情线分析等上层理解，一条命令覆盖预处理到成片输出的全链路。
- **兼容 FFmpeg，无缝迁移**：本地模式基于 `ffmpeg` / `ffprobe`，覆盖裁剪、拼接、加图片、加字幕、调速、调音量、翻转、淡入淡出、混音、音视频合成、提取音频、绿幕抠图、元信息探测等常用能力，与 FFmpeg 命令直觉对齐；滤镜、图片转视频、拼接转场等复杂 / AI 能力交由云端处理。
- **云端更快更强**：同一条命令加 `--cloud` 就能升维到 FFmpeg 做不到的能力 —— 画质增强 / 生成式画质修复、字幕擦除（标准版 / 精细化）、ASR、视频 OCR、高光智剪（短剧 / 小游戏）、剧情故事线分析、场景切分、绿幕 / 人像抠图等 AI 原子能力，云端弹性算力提供秒级并发。
- **一命令双模态**：`--local` / `--cloud` 可逐命令切换，本地零成本 + 云端弹性算力互补；共用同一份参数与 `--schema`，Agent / 脚本零改造切换。
- **高性价比处理**：依托云端弹性资源调度与闲时批量处理策略，为大批量媒体任务提供极具竞争力的价格，显著降低 AI 应用的整体 Token 消耗与运行成本。

## 功能

| 领域                | 能力                                                                                                                                                                                                                               | 运行             |
| ------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ---------------- |
| 🎬 **剪辑** (23)    | 视频加图片 · 视频加字幕 · 音频调速 · 视频调速 · 调整视频音量 · 视频添加运镜 · 视频添加滤镜 · 音频拼接 · 视频拼接 · 视频画面裁剪 · 视频截取动图 · 提取音频 · 音频声音淡入淡出 · 视频声音淡入淡出 · 视频画面翻转 · 图片转视频 · 音频混合 · 视频加音频 · 视频画面旋转 · 视频画面拼接 · 文字生成滚屏视频 · 音频裁剪 · 视频裁剪 | 云端 **或** 本地             |
| 🎚️ **音频** (4)    | 语音端点识别 · 音频元信息获取 · 人声背景音分离 · 音频转码 | 云端             |
| 🖼️ **图像 AI** (21)    | 添加图文水印 · 图像调整 · 图像压缩 · 图像裁剪 · 图像画质增强 · 图像擦除修复 · 图像画质评估 · 图像人脸打码 · 图像翻转 · 图像高斯模糊 · 图像文字识别OCR · 图像负片 · 图像打码 · 图像元信息获取 · 图像背景移除 · 图像缩放 · 图像旋转 · 圆角矩形 · 图像锐化 · 集智瘦身 · 图像智能裁剪 | 云端             |
| 🎥 **智能视频** (30)    | 添加视频暗水印 · 高光片段提取 · 剧情故事线分析 · 语音转字幕（ASR) · 视频画质检测（VQScore） · 解说视频生成 · 解说视频生成（短剧行业模型） · 剧本还原 · 画质增强 · 视频画质增强极速版 · 生成式画质增强 · 字幕擦除（标准版） · 精细化字幕擦除 · 视频抽帧 · 提取视频暗水印 · 视频人脸打码 · 视频人脸融合 · 高光智剪-短剧 · 高光智剪-小游戏 · 高光智剪-影视拆条 · 极智超清 · 视频绿幕抠图 · 视频人像抠图 · 视频元信息获取 · 视频转封装 · 场景切分 · 智能语义切片 · 视频转码 · 视频识别字幕（OCR） · 视频理解智能策略 | 云端             |
| 🔧 **通用** (2)     | 异步任务查询 · 远程文件拉取                                                                                                                                                                                                        | 本地 / 云端      |
| 🚧 **即将上线**     | 视频翻译 · 解说生成 · 漫剧转绘（陆续上线）                                                                                                                                                                                         | 云端             |

## 安装与快速开始

### 环境要求

开始之前，请确保具备以下条件：

- Node.js `>=18`（`npm` / `npx`）

- 本地模式：`ffmpeg` `5.1.x` 与 `ffprobe`

### 快速开始（人类用户）

#### 安装

以下两种方式**任选其一**：

**方式一 — 一键安装：**

```bash
npx @volcengine/mediakit-cli install -y
```

**方式二 — 从源码构建：**

需要 Go `v1.22`+。

```bash
git clone https://github.com/volcengine/mediakit-cli.git
cd mediakit-cli
make build          # 产物：.mediakit/build/dev/mediakit-cli

# 从本地 skills 目录安装 AI Agent Skills（必需）
npx -y skills add ./skills -g -y
```

#### 配置与使用

```bash
# 1. 初始化配置（交互式引导）
mediakit-cli init

# 2. 环境自检（检查云端连通性、本地依赖、安装建议）
mediakit-cli doctor

# 3. 本地剪辑（同步、无需 API Key）：本机运行 FFmpeg 完成裁剪
mediakit-cli --local editing trim-video --video-url ./in.mp4 --start-time 3 --end-time 8

# 4. 云端 AI（异步）：将视频画质增强至 1080p，然后轮询获取最终结果
mediakit-cli --cloud video enhance-video --video-url <url> --resolution 1080p
mediakit-cli shared query-task --task-id <task_id> --poll-complete
```

### 快速开始（AI Agent）

> 以下步骤面向 AI Agent，全流程支持无人值守。

**第 1 步 — 安装**

```bash
npx @volcengine/mediakit-cli install -y
```

**第 2 步 — 非交互式初始化（`--yes` 模式）**

```bash
# API Key 获取地址：https://console.volcengine.com/imp/ai-mediakit/settings
mediakit-cli init \
  --mode cloud-first \
  --api-key <your-api-key> \
  --yes
```

**第 3 步 — 验证**

```bash
mediakit-cli doctor
mediakit-cli version
```

## Agent Skills

| Skill                    | 说明                                                                                                                                                                                                                              |
| ------------------------ | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `byted-mediakit-shared`  | 通用能力：查询任务                                                                                                                                                                                                                |
| `byted-mediakit-editing` | 剪辑：视频加图片、视频加字幕、音频调速、视频调速、调整视频音量、视频添加运镜、视频添加滤镜、音频拼接、视频拼接、视频画面裁剪、视频截取动图、提取音频、音频声音淡入淡出、视频声音淡入淡出、视频画面翻转、图片转视频、音频混合、视频加音频、视频画面旋转、视频画面拼接、文字生成滚屏视频、音频裁剪、视频裁剪 |
| `byted-mediakit-audio`   | 音频：语音端点识别、音频元信息获取、人声背景音分离、音频转码                                                                                                                                                                      |
| `byted-mediakit-image`   | 图像 AI：添加图文水印、图像调整、图像压缩、图像裁剪、图像画质增强、图像擦除修复、图像画质评估、图像人脸打码、图像翻转、图像高斯模糊、图像文字识别 OCR、图像负片、图像打码、图像元信息获取、图像背景移除、图像缩放、图像旋转、圆角矩形、图像锐化、集智瘦身、图像智能裁剪                             |
| `byted-mediakit-video`   | 智能视频：添加视频暗水印、高光片段提取、剧情故事线分析、语音转字幕（ASR）、视频画质检测（VQScore）、解说视频生成、解说视频生成（短剧行业模型）、剧本还原、画质增强、视频画质增强极速版、生成式画质增强、字幕擦除（标准版）、精细化字幕擦除、视频抽帧、提取视频暗水印、视频人脸打码、视频人脸融合、高光智剪-短剧、高光智剪-小游戏、高光智剪-影视拆条、极智超清、视频绿幕抠图、视频人像抠图、视频元信息获取、视频转封装、场景切分、智能语义切片、视频转码、视频识别字幕（OCR）、视频理解智能策略 |

## 鉴权

`mediakit-cli` 采用极简鉴权：只需一个 API Key，无需 OAuth / STS / IAM 角色配置。

```bash
# 方式 A：init 时选择存储方式（config / shell / export）
mediakit-cli init --api-key <your-api-key> --credential-store config --yes

# 方式 B：临时通过环境变量注入
export MEDIAKIT_API_KEY=<your-api-key>
export MEDIAKIT_OUTPUT_PATH=<optional-custom-endpoint>
```

| 环境变量               | 说明                                                                                  |
| ---------------------- | ------------------------------------------------------------------------------------- |
| `MEDIAKIT_API_KEY`     | 云端 API Key（[控制台获取](https://console.volcengine.com/imp/ai-mediakit/settings)） |
| `MEDIAKIT_OUTPUT_PATH` | 本地模式输出目录，默认 `~/.mediakit/temp`                                             |

## 命令结构

```
mediakit-cli [--cloud|--local] <domain> <tool> [flags]
```

- **两种模式，同一命令面**：`--cloud` 走云端弹性算力（异步返回 `task_id`）；`--local` 走本地 FFmpeg（同步、零成本）。默认为 `cloud-first`，可用 `--cloud` / `--local` 逐命令覆盖。
- **输出**：云端结果以 URL 返回；本地结果落到 `~/.mediakit/temp`（可用 `--output-path` 或 `MEDIAKIT_OUTPUT_PATH` 覆盖）。

系统命令：

| 命令                                         | 说明                                                        |
| -------------------------------------------- | ----------------------------------------------------------- |
| `mediakit-cli init [--yes]`                  | 初始化配置，支持交互式或非交互式（Agent 友好）              |
| `mediakit-cli doctor`                        | 检查云端连通性、本地依赖与安装建议                          |
| `mediakit-cli config`                        | 查看 / 修改配置项                                           |
| `mediakit-cli version [--check]`             | 显示版本；`--check` 对比 npm 最新版                         |
| `mediakit-cli update [--check]`              | 通过 `npm install -g` 更新 CLI 和 Skills；`--check` 只检查不安装 |
| `mediakit-cli --domains`                     | 列出所有域                                                  |
| `mediakit-cli --help-full`                   | 列出全部能力索引                                            |
| `mediakit-cli <domain> <tool> --schema`      | 输出该能力的 JSON Schema（Mode / Async / 轮询命令等元信息） |
| `mediakit-cli shared query-task --task-id X` | 查询异步任务；加 `--poll-complete` 轮询至终态               |

## 进阶用法

### Schema 自省

每个能力命令都支持 `--schema`，输出输入 / 输出 schema、Mode 与 Async 信息，供 Agent 动态发现工具能力：

```bash
mediakit-cli video enhance-video --schema
mediakit-cli --local editing trim-video --schema
```

### 本地模式输出命名

本地模式输出文件按以下优先级命名：

1. 显式 `--output-path` 指定完整文件路径（含扩展名）→ 直接使用
2. 有输入文件名 → `{原文件名}_{工具名}.{ext}`；同名文件已存在时追加 6 位随机数字
3. 无输入文件名 → `{工具名}-{时间戳}.{ext}`

## 许可证

本项目基于 **MIT 许可证** 开源。

该软件运行时会调用 MediaKit 云端 API，使用这些 API 需要遵守如下协议：

- [视频云服务专有条款](https://www.volcengine.com/docs/6448/79646?lang=zh)
- [智能处理服务计费规则](https://www.volcengine.com/docs/6448/104992?lang=zh)
- [智能处理服务 SLA](https://www.volcengine.com/docs/6448/79648?lang=zh)
