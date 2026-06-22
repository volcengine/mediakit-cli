# 图像画质评估

## 能力描述
对输入图片进行主客观画质和美学评分，适用于质量监控、低质图筛查、内容审核、推荐排序和训练数据清洗等场景。
支持标准版多维评分与专业版大模型评分。

## 执行方式

| 项目 | 说明 |
|------|------|
| Domain | `image` |
| Tool | `evaluate-image-quality` |
| 是否异步 | `否` |
| 是否支持 local | `否` |
| 模式说明 | cloud only；同步执行，调用成功后直接返回最终结果 |
| 幂等行为 | 如命令支持 `client_token` 与 `callback_args`，重试时复用同一组值；强制重跑时更换新的 `client_token` |

## 参数
| 参数 | CLI flag | 类型 | 必填 | 默认值 | 说明 |
|------|----------|------|------|--------|------|
| image_url | `--image-url` | string | 是 | - | 输入图片 URL，需为公网可访问的 png/jpeg/webp/heic 图片，单图不超过 10MB。 |
| tool_version | `--tool-version` | string | 否 | standard | 画质评估模型版本，standard 标准版，professional 专业版 |
| standard_evaluate_items | `--standard-evaluate-items` | array<string> | 否 | ["vqscore","noise","aesthetic","blur"] | 标准版选用的评估工具。子项说明：评估工具。CLI 传参时请使用 JSON 字符串，并用单引号包裹整个值，例如 `--standard-evaluate-items '["vqscore","noise"]'` |
| callback_args | `--callback-args` | string | 否 | - | 可选，回调参数 |
| client_token | `--client-token` | string | 否 | - | 可选，用于幂等，默认幂等，用户可根据需求进行调整 |

## 调用示例
```bash
mediakit-cli image evaluate-image-quality \
  --image-url https://example.com/image_url \
  --tool-version standard \
  --standard-evaluate-items '["vqscore","noise","aesthetic","blur"]' \
  --callback-args sample-callback-args \
  --client-token demo-client-token
```

## 输出格式
```json
{
  "vqscore": 78.5,
  "aesthetic": 82.1,
  "noise": 12.3,
  "blur": 8.6
}
```
