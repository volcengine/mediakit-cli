# 音频混合

## 能力描述
将多个音频文件（如背景音乐、音效、人声）进行混音，生成一个新的音频文件。
处理耗时与原片时长正相关，平均 RTF（处理耗时/原片时长）为 1。
输出音频的时长以最长的音频为准。
输出格式：mp3。

## 执行方式

| 项目 | 说明 |
|------|------|
| Domain | `editing` |
| Tool | `mix-audio` |
| 是否异步 | `是` |
| 是否支持 local | `是` |
| 模式说明 | 支持 local / cloud；可通过 `--local` 或 `--cloud` 覆盖当前命令 |
| 幂等行为 | 如命令支持 `client_token` 与 `callback_args`，重试时复用同一组值；强制重跑时更换新的 `client_token` |

## 参数
| 参数 | CLI flag | 类型 | 必填 | 默认值 | 说明 |
|------|----------|------|------|--------|------|
| audio_urls | `--audio-urls` | array<string> | 是 | - | 输入音频列表。 CLI 传参时请使用 JSON 字符串，并用单引号包裹整个值 |
| callback_args | `--callback-args` | string | 否 | - | 可选，回调参数 |
| client_token | `--client-token` | string | 否 | - | 可选，用于幂等，默认幂等，用户可根据需求进行调整 |

## 调用示例
```bash
mediakit-cli editing mix-audio \
  --audio-urls '["https://example.com/a.mp3","https://example.com/b.mp3"]' \
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

- 当前命令：`mediakit-cli editing mix-audio`
- 推荐查询：`mediakit-cli shared query-task --task-id <task_id>`
