# 视频加音频

## 能力描述
音视频合成。

## 执行方式

| 项目 | 说明 |
|------|------|
| Domain | `editing` |
| Tool | `mux-audio-video` |
| 是否异步 | `是` |
| 是否支持 local | `是` |
| 模式说明 | 支持 local / cloud；可通过 `--local` 或 `--cloud` 覆盖当前命令。 |
| 幂等行为 | 如命令支持 `client_token` 与 `callback_args`，重试时复用同一组值；强制重跑时更换新的 `client_token`。 |

## 参数
| 参数 | CLI flag | 类型 | 必填 | 默认值 | 说明 |
|------|----------|------|------|--------|------|
| video_url | `--video-url` | string | 是 | - | 输入视频。String 类型，支持http://xxx或https://xxx格式 URL |
| audio_url | `--audio-url` | string | 是 | - | 输入音频。String 类型，支持http://xxx或https://xxx格式 URL |
| is_audio_reserve | `--is-audio-reserve` | boolean | 否 | True | Boolean 类型，是否保留原视频流中的音频。默认值 true：保留。false：不保留。 |
| is_video_audio_sync | `--is-video-audio-sync` | boolean | 否 | False | Boolean 类型，是否对齐音频和视频时长。 true：通过 output_sync 配置，对齐音频和视频时长。 false（默认值）：保持原样输出，不做音视频对齐。最终合成的视频时长，以较长的流为准。 |
| sync_mode | `--sync-mode` | string | 否 | video | String 类型，设置 is_video_audio_sync 为 true 时生效；当音频和视频时长不相等时，可指定对齐基准，可选项：video、audio。 video：【默认值】以视频的时长为准。 audio：以音频的时长为准。 |
| sync_method | `--sync-method` | string | 否 | trim | String 类型，设置 is_video_audio_sync 为 true 时生效；指定对齐方式，支持通过裁剪或加速的方式，对齐音频和视频的时长。可选项：speed、trim。 speed：通过加快音频或视频的速度，对齐音频和视频的时长。 trim：【默认值】通过裁剪音频或视频，对齐音频和视频的时长。从头开始计算并裁剪。 |
| callback_args | `--callback-args` | string | 否 | - | 可选，回调参数 |
| client_token | `--client-token` | string | 否 | - | 可选，用于幂等，默认幂等，用户可根据需求进行调整 |

## 调用示例
```bash
mediakit-cli editing mux-audio-video \
  --video-url https://example.com/video_url \
  --audio-url https://example.com/audio_url \
  --is-audio-reserve true \
  --is-video-audio-sync true \
  --sync-mode <sync_mode> \
  --sync-method <sync_method> \
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

- 当前命令：`mediakit-cli editing mux-audio-video`
- 推荐查询：`mediakit-cli shared query-task --task-id <task_id>`
