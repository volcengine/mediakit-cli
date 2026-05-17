---
name: byted-mediakit-shared
version: "1.0.0"
description: "mediakit-cli 共享：环境检查、初始化配置、命令结构、认证配置、异步任务响应与错误处理。"
permissions:
  - shell
metadata:
  requires:
    bins: ["mediakit-cli"]
  cliHelp: "mediakit-cli --help"
  product: mediakit-cli/skills
  domain: shared
  capability_count: 14
---
# MediaKit 共享规则

本技能指导你如何通过 mediakit-cli 操作媒体资源，以及调用过程中的通用规则和注意事项。

## 前置检查

### 依赖安装

首次使用前，确认 CLI 已安装：

```bash
# 安装
npm install -g @ai-mediakit/cli

# 验证
mediakit-cli --version
```

### 鉴权信息检查

优先级：环境变量 > 配置文件（文件路径 `~/.mediakit/config.json`）

#### 字段说明

- 环境变量/配置文件：`MEDIAKIT_API_KEY`、`MEDIAKIT_ENDPOINT`

| 变量 | 必填 | 说明 |
|------|------|------|
| `MEDIAKIT_API_KEY` | 云端模式必填 | API 认证 Token |
| `MEDIAKIT_ENDPOINT` | 否 | API 访问点 |
| `MEDIAKIT_SURFACE` | 否 | 仅 CLI 云端调用支持；Skill 建议 `skill`，Plugin 建议 `plugin`，最终上报 `cli/skill` 或 `cli/plugin` |
| `MEDIAKIT_RUNTIME` | 否 | 请求来源 Header `x-runtime`；按宿主设置为 `claude`、`arkclaw` 等 |

任一必填项缺失时，终止执行并输出所有缺失项的列表及修复建议。

云端调用会自动携带 `x-surface` / `x-runtime`。当本 Skill/Plugin 通过 `mediakit-cli` 调用云端能力时，运行环境应注入 `MEDIAKIT_SURFACE=skill|plugin` 与 `MEDIAKIT_RUNTIME=<宿主>`；CLI 会保留原始产物前缀并上报 `x-surface=cli/skill|cli/plugin`。否则 CLI 默认按 `x-surface=cli`、`x-runtime=unknown` 上报。

### 来源上报约束

- Skill 调用 `mediakit-cli` 时，必须显式设置 `MEDIAKIT_SURFACE=skill`，不能依赖用户已有环境变量。
- Plugin 调用 `mediakit-cli` 时，必须显式设置 `MEDIAKIT_SURFACE=plugin`，不能复用 Skill 的取值。
- 宿主环境标识建议同时显式设置 `MEDIAKIT_RUNTIME=<宿主>`；若未设置，CLI 会回退为环境探测值或 `unknown`。

```bash
MEDIAKIT_SURFACE=skill MEDIAKIT_RUNTIME=<runtime> mediakit-cli editing add-image-to-video

MEDIAKIT_SURFACE=plugin MEDIAKIT_RUNTIME=<runtime> mediakit-cli editing add-image-to-video
```

## CLI 使用方式

### 初始化配置

首次使用建议先运行初始化向导：

```bash
mediakit-cli init
```

初始化后常用命令如下：

```bash
# 查看当前配置
mediakit-cli config show

# 切换默认模式到本地优先
mediakit-cli config set mode local-first

# 切换默认模式到云端优先
mediakit-cli config set mode cloud-first

# 刷新环境检查并查看依赖状态
mediakit-cli doctor
```

### 命令结构

MediaKit CLI 统一使用 `domain + tool` 的调用方式：

```bash
mediakit-cli {domain} {tool} [flags]
```

常见帮助命令：

```bash
# 查看所有 domain
mediakit-cli --domains

# 查看某个分组下的工具列表
mediakit-cli {domain} --help

# 查看具体工具的参数
mediakit-cli {domain} {tool} --help
```

当前产物覆盖的 domain 包括：`editing`, `video`。

### 单次调用模式覆盖

除 `config set mode` 设置默认模式外，还支持仅对当前命令生效的临时覆盖：

```bash
mediakit-cli --local editing add-image-to-video

mediakit-cli --cloud editing add-image-to-video
```

补充规则：

- `--local` / `--cloud` 只影响当前命令，不修改全局 `config.mode`
- `--local` 与 `--cloud` 互斥，不能同时传入

## 异步任务

提交异步媒体处理任务成功后会返回 `task_id` 字段。通过 `shared query-task` 命令查询结果。

```bash
mediakit-cli shared query-task --task-id <task_id>
```

## local / cloud 约束

- `query-task` 是 **cloud only** 工具
- local 模式下不支持 query-task
- 当前本轮能力以云端执行为主；如需显式声明，请优先使用 `--cloud`

## 幂等参数维护

| 参数 | 作用 | 维护建议 |
|------|------|----------|
| `client_token` | 主动控制幂等 | 请求重试时复用同一值；强制重新执行时传新的唯一值 |
| `callback_args` | 透传回调参数 | 建议与 `client_token` 一起维护，便于回调对账与重试追踪 |

补充规则：

- `client_token` 长度不超过 64 个字符
- `callback_args` 可用于回调透传与对账追踪

## 轮询策略

| 参数 | 描述 | 默认值 |
|------|------|--------|
| `poll-interval-seconds` | 轮询间隔 | 10s |
| `max-poll-attempts` | 轮询次数，0 代表不查询 | 0 |
| `poll-complete` | 阻塞至终态 | - |
