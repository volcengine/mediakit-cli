# 图像擦除修复

## 能力描述
自动检测并擦除图片中的常见图标、文字或指定区域内容，并对擦除区域进行背景智能填充。

## 执行方式

| 项目 | 说明 |
|------|------|
| Domain | `image` |
| Tool | `erase-image` |
| 是否异步 | `否` |
| 是否支持 local | `否` |
| 模式说明 | cloud only；同步执行，调用成功后直接返回最终结果 |
| 幂等行为 | 如命令支持 `client_token` 与 `callback_args`，重试时复用同一组值；强制重跑时更换新的 `client_token` |

## 参数
| 参数 | CLI flag | 类型 | 必填 | 默认值 | 说明 |
|------|----------|------|------|--------|------|
| image_url | `--image-url` | string | 是 | - | 输入图片 URL，需为公网可访问的 png/jpg/jpeg/webp/tiff/bmp/heic 图片，单图不超过 10MB。 |
| tool_version | `--tool-version` | string | 否 | standard | 图像擦除修复选用的模型版本。standard：标准版，基于明确的规则（如文本匹配、矩形框坐标）擦除指定内容。适用于简单、明确的擦除任务 |
| standard_scene | `--standard-scene` | string | 否 | full_screen_text_erase | 标准版擦除场景，仅 standard 版本生效。full_screen_text_erase：全屏文字擦除，可通过 standard_erase_text 指定要擦除的文字，不指定则默认擦除所有文字内容。full_screen_icon_erase：全屏图标擦除 |
| standard_erase_text | `--standard-erase-text` | string | 否 | - | 标准版文字擦除，指定要擦除的文字，不指定则默认擦除所有文字内容 |
| output_format | `--output-format` | string | 否 | webp | 输出图片格式，可选值：png、jpeg、webp |
| callback_args | `--callback-args` | string | 否 | - | 可选，回调参数 |
| client_token | `--client-token` | string | 否 | - | 可选，用于幂等，默认幂等，用户可根据需求进行调整 |

## 调用示例
```bash
mediakit-cli image erase-image \
  --image-url https://example.com/image_url \
  --tool-version standard \
  --standard-scene full_screen_text_erase \
  --output-format webp \
  --callback-args sample-callback-args \
  --client-token demo-client-token
```

## 输出格式
```json
{
  "image_url": "https://example.com/erased.webp",
  "image_size": 102400,
  "image_format": "webp",
  "image_width": 1080,
  "image_height": 1920
}
```
