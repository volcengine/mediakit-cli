# 视频添加滤镜

## 能力描述
为视频添加指定滤镜效果，输出mp4，分辨率与原片一致。

## 执行方式

| 项目 | 说明 |
|------|------|
| Domain | `editing` |
| Tool | `apply-video-filter` |
| 是否异步 | `是` |
| 是否支持 local | `否` |
| 模式说明 | cloud only；可通过 `--cloud` 强制当前调用 |
| 幂等行为 | 如命令支持 `client_token` 与 `callback_args`，重试时复用同一组值；强制重跑时更换新的 `client_token` |

## 参数
| 参数 | CLI flag | 类型 | 必填 | 默认值 | 说明 |
|------|----------|------|------|--------|------|
| video_url | `--video-url` | string | 是 | - | 输入视频，支持 mp4、mov、flv、ts、avi、wmv、mkv 等格式，最高 4K |
| filter_style | `--filter-style` | string | 否 | spring | 滤镜风格。可选值：spring（春日滤镜）、sunset（晚霞滤镜）、vivid（鲜亮滤镜）、fair_skin（白皙滤镜）、food（食物滤镜） |
| callback_args | `--callback-args` | string | 否 | - | 可选，回调参数 |
| client_token | `--client-token` | string | 否 | - | 可选，用于幂等，默认幂等，用户可根据需求进行调整 |

## 调用示例
```bash
mediakit-cli editing apply-video-filter \
  --video-url https://example.com/video_url \
  --filter-style spring \
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

- 当前命令：`mediakit-cli editing apply-video-filter`
- 推荐查询：`mediakit-cli shared query-task --task-id <task_id>`
