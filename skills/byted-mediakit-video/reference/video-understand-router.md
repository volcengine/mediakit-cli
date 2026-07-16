# 视频理解智能策略

## 能力描述
基于视觉大模型的通用视频内容分析算子，对输入的视频 URL 列表进行智能分析，
输出视频级别的结构化理解结果，适用于内容审核、视频检索、标签生成等场景。
使用限制：单次最多输入 10 个视频，单个视频时长不超过 2 小时。

## 执行方式

| 项目 | 说明 |
|------|------|
| Domain | `video` |
| Tool | `video-understand-router` |
| 是否异步 | `是` |
| 是否支持 local | `否` |
| 模式说明 | cloud only；可通过 `--cloud` 强制当前调用 |
| 幂等行为 | 如命令支持 `client_token` 与 `callback_args`，重试时复用同一组值；强制重跑时更换新的 `client_token` |

## 参数
| 参数 | CLI flag | 类型 | 必填 | 默认值 | 说明 |
|------|----------|------|------|--------|------|
| video_urls | `--video-urls` | array<string> | 是 | - | 待处理的视频 URL 列表，支持 HTTP/HTTPS 公网可访问链接，最多 10 个视频。子项说明：视频 URL |
| prompt | `--prompt` | string | 是 | - | 提示词 |
| level | `--level` | string | 否 | Economy | 分析档位, 可选值: Economy, Balanced, Quality |
| prefer_endpoints | `--prefer-endpoints` | array<string> | 否 | - | 优先使用指定的自定义推理点列表，最多 10 个，优先级高于prefer_models。子项说明：指定使用的自定义推理点 |
| prefer_models | `--prefer-models` | array<string> | 否 | - | 优先使用指定的模型列表，最多 10 个模型。子项说明：指定使用的模型 |
| manual_option | `--manual-option` | object | 否 | - | 手动模式相关参数。CLI 传参时请使用 JSON 字符串，并用单引号包裹整个值 |
| callback_args | `--callback-args` | string | 否 | - | 可选，回调参数 |
| client_token | `--client-token` | string | 否 | - | 可选，用于幂等，默认幂等，用户可根据需求进行调整 |

## 调用示例
```bash
mediakit-cli video video-understand-router \
  --video-urls '["https://example.com/video_url"]' \
  --prompt "分析视频主要内容" \
  --level Economy \
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

- 当前命令：`mediakit-cli video video-understand-router`
- 推荐查询：`mediakit-cli shared query-task --task-id <task_id>`
