# 提取音频

## 能力描述
将视频文件中的音频流分离并保存为独立的音频文件。

## 执行方式

| 项目 | 说明 |
|------|------|
| Domain | `editing` |
| Tool | `extract-audio` |
| 是否异步 | `是` |
| 是否支持 local | `是` |
| 模式说明 | 支持 local / cloud；可通过 `--local` 或 `--cloud` 覆盖当前命令。 |
| 幂等行为 | 如命令支持 `client_token` 与 `callback_args`，重试时复用同一组值；强制重跑时更换新的 `client_token`。 |

## 参数
| 参数 | CLI flag | 类型 | 必填 | 默认值 | 说明 |
|------|----------|------|------|--------|------|
| video_url | `--video-url` | string | 是 | - | 输入视频，String 类型，支持http://xxx或https://xxx格式 URL |
| format | `--format` | string | 否 | m4a | 输出音频的格式，支持 mp3、m4a 格式。 默认m4a |
| callback_args | `--callback-args` | string | 否 | - | 可选，回调参数 |
| client_token | `--client-token` | string | 否 | - | 可选，用于幂等，默认幂等，用户可根据需求进行调整 |

## 调用示例
```bash
mediakit-cli editing extract-audio \
  --video-url https://example.com/video_url \
  --format mp3 \
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

- 当前命令：`mediakit-cli editing extract-audio`
- 推荐查询：`mediakit-cli shared query-task --task-id <task_id>`
