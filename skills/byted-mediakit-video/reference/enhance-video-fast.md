# 视频画质增强极速版

## 能力描述
集成轻量级超分与智能画质增强能力，采用速度优先算法优化策略，高效兼顾处理效率与画面效果，适配各类时延敏感型业务场景。

## 执行方式

| 项目 | 说明 |
|------|------|
| Domain | `video` |
| Tool | `enhance-video-fast` |
| 是否异步 | `是` |
| 是否支持 local | `否` |
| 模式说明 | cloud only；可通过 `--cloud` 强制当前调用 |
| 幂等行为 | 如命令支持 `client_token` 与 `callback_args`，重试时复用同一组值；强制重跑时更换新的 `client_token` |

## 参数
| 参数 | CLI flag | 类型 | 必填 | 默认值 | 说明 |
|------|----------|------|------|--------|------|
| video_url | `--video-url` | string | 是 | - | 输入视频。String 类型，支持http://xxx或https://xxx格式 URL |
| resolution | `--resolution` | string | 否 | - | 目标分辨率。支持的取值：240p / 360p / 480p / 540p / 720p / 1080p / 2k / 4k。配置此参数后，不可同时配置resolution_limit字段 |
| resolution_limit | `--resolution-limit` | integer | 否 | - | 指定输出视频的短边像素值，取值范围为 [128, 2160]。设置后，系统将锁定视频的短边像素值为设定值，并在保持原视频宽高比的前提下，等比缩放至该限制值。配置此参数后，不可同时配置resolution字段 |
| bitrate_level | `--bitrate-level` | string | 否 | medium | 码率档位。输出视频的目标平均码率。取值：low（低码率）/ medium（中码率，推荐）/ high（高码率）。默认为 medium |
| fps | `--fps` | number | 否 | - | 目标帧率，单位为 fps。取值范围为 [15, 120]。 |
| callback_args | `--callback-args` | string | 否 | - | 可选，回调参数 |
| client_token | `--client-token` | string | 否 | - | 可选，用于幂等，默认幂等，用户可根据需求进行调整 |

## 调用示例
```bash
mediakit-cli video enhance-video-fast \
  --video-url https://example.com/video_url \
  --resolution 720p \
  --bitrate-level medium \
  --fps 30 \
  --callback-args sample-callback-args \
  --client-token demo-client-token
```

## 输出格式
```json
{
  "task_id": "task_demo_001",
  "request_id": "req_demo_001"
}
```

## 任务结果查询
提交成功后会返回 `task_id`，再执行 `mediakit-cli shared query-task --task-id <task_id>` 查询。

- 当前命令：`mediakit-cli video enhance-video-fast`
- 推荐查询：`mediakit-cli shared query-task --task-id <task_id>`
