# 图像文字识别OCR

## 能力描述
识别图片中的通用印刷体文字，返回可编辑文本、文字框坐标和置信度。
本期支持简体中文和英文通用场景识别。

## 执行方式

| 项目 | 说明 |
|------|------|
| Domain | `image` |
| Tool | `image-ocr` |
| 是否异步 | `否` |
| 是否支持 local | `否` |
| 模式说明 | cloud only；同步执行，调用成功后直接返回最终结果 |
| 幂等行为 | 如命令支持 `client_token` 与 `callback_args`，重试时复用同一组值；强制重跑时更换新的 `client_token` |

## 参数
| 参数 | CLI flag | 类型 | 必填 | 默认值 | 说明 |
|------|----------|------|------|--------|------|
| image_url | `--image-url` | string | 是 | - | 输入图片 URL，需为公网可访问的 png/jpg/jpeg/webp/heic/avif 图片，单图不超过 10MB。 |
| callback_args | `--callback-args` | string | 否 | - | 可选，回调参数 |
| client_token | `--client-token` | string | 否 | - | 可选，用于幂等，默认幂等，用户可根据需求进行调整 |

## 调用示例
```bash
mediakit-cli image image-ocr \
  --image-url https://example.com/image_url \
  --callback-args sample-callback-args \
  --client-token demo-client-token
```

## 输出格式
```json
{
  "ocr_result": [
    {
      "content": "示例文字",
      "confidence": 0.98,
      "top_left_x": 10,
      "top_left_y": 20,
      "bottom_right_x": 120,
      "bottom_right_y": 60
    }
  ]
}
```
