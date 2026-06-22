# 视频声音淡入淡出

## 能力描述
对输入视频的声轨实现淡入淡出效果。
输出 mp4，分辨率与原片一致。

## 执行方式

| 项目 | 说明 |
|------|------|
| Domain | `editing` |
| Tool | `fade-video-audio` |
| 是否异步 | `是` |
| 是否支持 local | `是` |
| 模式说明 | 支持 local / cloud；可通过 `--local` 或 `--cloud` 覆盖当前命令 |
| 幂等行为 | 如命令支持 `client_token` 与 `callback_args`，重试时复用同一组值；强制重跑时更换新的 `client_token` |

## 参数
| 参数 | CLI flag | 类型 | 必填 | 默认值 | 说明 |
|------|----------|------|------|--------|------|
| video_url | `--video-url` | string | 是 | - | 输入视频，支持 mp4、mov、flv、ts、avi、wmv、mkv 等格式，最高 4K |
| fade_in_duration | `--fade-in-duration` | number | 否 | 1 | 声音淡入时长。单位：秒，可传小数（最多3位小数）。0 表示不淡入 |
| fade_out_duration | `--fade-out-duration` | number | 否 | 1 | 声音淡出时长。单位：秒，可传小数（最多3位小数）。0 表示不淡出 |
| callback_args | `--callback-args` | string | 否 | - | 可选，回调参数 |
| client_token | `--client-token` | string | 否 | - | 可选，用于幂等，默认幂等，用户可根据需求进行调整 |

## 调用示例
```bash
mediakit-cli editing fade-video-audio \
  --video-url https://example.com/video_url \
  --fade-in-duration 1 \
  --fade-out-duration 1 \
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

- 当前命令：`mediakit-cli editing fade-video-audio`
- 推荐查询：`mediakit-cli shared query-task --task-id <task_id>`
