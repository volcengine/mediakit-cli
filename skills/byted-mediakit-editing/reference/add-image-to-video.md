# 视频加图片

## 能力描述
视频加图片，可用作加图片水印。

## 执行方式

| 项目 | 说明 |
|------|------|
| Domain | `editing` |
| Tool | `add-image-to-video` |
| 是否异步 | `是` |
| 是否支持 local | `是` |
| 模式说明 | 支持 local / cloud；可通过 `--local` 或 `--cloud` 覆盖当前命令。 |
| 幂等行为 | 如命令支持 `client_token` 与 `callback_args`，重试时复用同一组值；强制重跑时更换新的 `client_token`。 |

## 参数
| 参数 | CLI flag | 类型 | 必填 | 默认值 | 说明 |
|------|----------|------|------|--------|------|
| video_url | `--video-url` | string | 是 | - | 输入视频。String 类型，支持http://xxx或https://xxx格式 URL |
| sub_image_url | `--sub-image-url` | string | 是 | - | 图片URL。支持http://xxx或https://xxx格式 URL |
| sub_image_height | `--sub-image-height` | string | 否 | - | 图片的高度，字符串类型，支持具体像素值（如 '100'）或百分比（如 '20%'，相对于视频高度）。不传时 local 模式保持原始图片高度。 |
| sub_image_width | `--sub-image-width` | string | 否 | - | 图片的宽度，字符串类型，支持具体像素值（如 '100'）或百分比（如 '20%'，相对于视频宽度）。不传时 local 模式保持原始图片宽度。 |
| sub_image_pos_x | `--sub-image-pos-x` | string | 否 | 85% | 图片左上角在水平方向（X 轴）的位置，以视频左上角为原点，百分比表示视频宽度的绝对位置；超出画面时自然截断。 |
| sub_image_pos_y | `--sub-image-pos-y` | string | 否 | 90% | 图片左上角在垂直方向（Y 轴）的位置，以视频左上角为原点，百分比表示视频高度的绝对位置；超出画面时自然截断。 |
| start_time | `--start-time` | number | 否 | - | 图片的开始时间，单位：秒。不传默认同视频开始时间 |
| end_time | `--end-time` | number | 否 | - | 图片的结束时间，单位：秒。注意：如果设置的开始/结束时间超出原始视频时长，输出视频长度将以该结束时间为准，超出部分以黑屏形式延续。不传默认同视频结束时间 |
| callback_args | `--callback-args` | string | 否 | - | 可选，回调参数 |
| client_token | `--client-token` | string | 否 | - | 可选，用于幂等，默认幂等，用户可根据需求进行调整 |

## 调用示例
```bash
mediakit-cli editing add-image-to-video \
  --video-url https://example.com/video_url \
  --sub-image-url https://example.com/sub_image_url \
  --sub-image-height <sub_image_height> \
  --sub-image-width <sub_image_width> \
  --sub-image-pos-x <sub_image_pos_x> \
  --sub-image-pos-y <sub_image_pos_y> \
  --start-time 1.0 \
  --end-time 1.0 \
  --callback-args sample-callback-args \
  --client-token demo-client-token
```

## Local 行为说明

- 不传 `--sub-image-width` / `--sub-image-height` 时，local 模式保持图片原始尺寸，与云端默认效果对齐
- 传入 `--sub-image-width` / `--sub-image-height` 时，local 模式按指定像素或百分比缩放图片
- `--sub-image-pos-x 95% --sub-image-pos-y 95%` 表示图片左上角位于视频宽/高的 95% 位置，右侧或底部超出画面的部分会被截断

## 输出格式
```json
{
  "task_id": "task_demo_001",
  "request_id": "req_demo_001"
}
```

## 任务结果查询
提交成功后会返回 `task_id`，再执行 `mediakit-cli shared query-task --task-id <task_id>` 查询。

- 当前命令：`mediakit-cli editing add-image-to-video`
- 推荐查询：`mediakit-cli shared query-task --task-id <task_id>`
