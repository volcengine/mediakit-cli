# 图像画质增强

## 能力描述
基于图像内容理解智能决策，全方位提升图片分辨率、清晰度与色彩表现。

## 执行方式

| 项目 | 说明 |
|------|------|
| Domain | `image` |
| Tool | `enhance-image` |
| 是否异步 | `否` |
| 是否支持 local | `否` |
| 模式说明 | cloud only；同步执行，调用成功后直接返回最终结果 |
| 幂等行为 | 如命令支持 `client_token` 与 `callback_args`，重试时复用同一组值；强制重跑时更换新的 `client_token` |

## 参数
| 参数 | CLI flag | 类型 | 必填 | 默认值 | 说明 |
|------|----------|------|------|--------|------|
| image_url | `--image-url` | string | 是 | - | 输入图片。String 类型，支持http://xxx或https://xxx格式 URL |
| tool_version | `--tool-version` | string | 否 | standard | 画质增强选用的模型版本，标准版:standard；专业版：professional |
| multiple | `--multiple` | number | 否 | - | 图像处理后较原图的分辨率倍数，支持 2 位小数。取值范围 [1,30]，最大不超过 30。standard 模式下最大不超过 8。处理后宽高不能超过 target_width、target_height 上限 |
| target_width | `--target-width` | integer | 否 | - | 图像处理后的宽度，单位 px，取值不能超过 10240。standard 模式下最大不超过 6144，且分辨率倍数不能超过 8 |
| target_height | `--target-height` | integer | 否 | - | 图像处理后的高度，单位 px，取值不能超过 10240。standard 模式下最大不超过 6144，且分辨率倍数不能超过 8 |
| callback_args | `--callback-args` | string | 否 | - | 可选，回调参数 |
| client_token | `--client-token` | string | 否 | - | 可选，用于幂等，默认幂等，用户可根据需求进行调整 |

## 调用示例
```bash
mediakit-cli image enhance-image \
  --image-url https://example.com/image_url \
  --tool-version standard \
  --multiple 2 \
  --callback-args sample-callback-args \
  --client-token demo-client-token
```

## 输出格式
```json
{
  "image_url": "https://example.com/enhanced.png",
  "image_size": 204800,
  "image_format": "png",
  "image_width": 2160,
  "image_height": 3840
}
```
