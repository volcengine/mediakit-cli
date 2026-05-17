# 视频加字幕

## 能力描述
将字幕文件或文本内容，以指定样式压制到视频画面中，生成带内嵌字幕的新视频。

## 执行方式

| 项目 | 说明 |
|------|------|
| Domain | `editing` |
| Tool | `add-subtitle-to-video` |
| 是否异步 | `是` |
| 是否支持 local | `是` |
| 模式说明 | 支持 local / cloud；可通过 `--local` 或 `--cloud` 覆盖当前命令。 |
| 幂等行为 | 如命令支持 `client_token` 与 `callback_args`，重试时复用同一组值；强制重跑时更换新的 `client_token`。 |

## 参数
| 参数 | CLI flag | 类型 | 必填 | 默认值 | 说明 |
|------|----------|------|------|--------|------|
| video_url | `--video-url` | string | 是 | - | 输入视频。String 类型，支持http://xxx或https://xxx格式 URL |
| subtitle_url | `--subtitle-url` | string | 否 | - | 字幕文件 URL、filename。常见的字幕文件为 SRT、VTT、ASS 等格式。 |
| subtitles | `--subtitles` | array<object{subtitle_text:string, start_time:number, end_time:number}> | 否 | - | 字幕列表，Array<object>类型。 CLI 传参时请使用 JSON 字符串，并用单引号包裹整个值，例如 `--subtitles '[{"subtitle_text": "<subtitle_text>", "start_time": 1.0, "end_time": 1.0}]'`。 |
| subtitles[].subtitle_text | `--subtitles[].subtitle_text` | string | 是 | - | 字幕文本 |
| subtitles[].start_time | `--subtitles[].start_time` | number | 是 | - | 字幕开始时间。单位：秒。 |
| subtitles[].end_time | `--subtitles[].end_time` | number | 是 | - | 字幕结束时间。单位：秒。 |
| subtitle_pos_preset | `--subtitle-pos-preset` | string | 否 | bottom_center | 预设字幕位置。底部居中（默认常用） bottom_center；顶部居中 top_center；画面正中央 center；偏下三分之一处 lower_third |
| subtitle_font_size | `--subtitle-font-size` | integer | 否 | 50 | 字幕的字体大小，单位：像素。 |
| subtitle_font_color | `--subtitle-font-color` | string | 否 | #FFFFFFFF | 字幕的字体颜色，RGBA 格式。默认#FFFFFFFF |
| subtitle_font_type | `--subtitle-font-type` | string | 否 | sy_black | 字幕的字体 ID。 思源黑体：sy_black （经典无衬线黑体，端正百搭，正文首选） 庞门正道标题体：pm_zhengdao （粗壮有力，硬汉气场，大标题/封面神器） 阿里巴巴普惠体：ali_puhui （现代感极强，结构饱满，屏幕阅读体验极佳） 站酷快乐体：zhanku_kuaile （圆润活泼，带手写感，适合轻松搞笑的 Vlog 氛围） |
| callback_args | `--callback-args` | string | 否 | - | 可选，回调参数 |
| client_token | `--client-token` | string | 否 | - | 可选，用于幂等，默认幂等，用户可根据需求进行调整 |

## 调用示例
```bash
mediakit-cli editing add-subtitle-to-video \
  --video-url https://example.com/video_url \
  --subtitle-url https://example.com/subtitle_url \
  --subtitles '[{"subtitle_text": "<subtitle_text>", "start_time": 1.0, "end_time": 1.0}]' \
  --subtitle-pos-preset bottom_center \
  --subtitle-font-size 1 \
  --subtitle-font-color <subtitle_font_color> \
  --subtitle-font-type sy_black \
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

- 当前命令：`mediakit-cli editing add-subtitle-to-video`
- 推荐查询：`mediakit-cli shared query-task --task-id <task_id>`
