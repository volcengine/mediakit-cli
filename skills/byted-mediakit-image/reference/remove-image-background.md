# 图像背景移除

## 能力描述
自动识别并保留图像主体，移除背景并生成透明背景图片。
支持通用、人像、商品场景，可在人像/商品场景中生成主体描边或裁剪透明背景。

## 执行方式

| 项目 | 说明 |
|------|------|
| Domain | `image` |
| Tool | `remove-image-background` |
| 是否异步 | `否` |
| 是否支持 local | `否` |
| 模式说明 | cloud only；同步执行，调用成功后直接返回最终结果 |
| 幂等行为 | 如命令支持 `client_token` 与 `callback_args`，重试时复用同一组值；强制重跑时更换新的 `client_token` |

## 参数
| 参数 | CLI flag | 类型 | 必填 | 默认值 | 说明 |
|------|----------|------|------|--------|------|
| image_url | `--image-url` | string | 是 | - | 输入图片 URL，需为公网可访问的 png/jpg/jpeg/webp/tiff/bmp/ico 图片，单图不超过 10MB。 |
| scene | `--scene` | string | 是 | - | 背景移除场景：general 通用场景，适用于期望抠出图像主体但不确定主体分类的场景；human 人像抠图场景，仅需抠出人像主体；product 商品抠图场景，仅需抠出商品主体 |
| need_contour | `--need-contour` | boolean | 否 | false | 是否为主体生成描边；仅 human/product 场景生效，general 场景忽略 |
| contour_color | `--contour-color` | string | 否 | #FFFFFF | 主体描边颜色，十六进制 RGB；仅 need_contour=true 且 human/product 场景生效 |
| contour_size | `--contour-size` | integer | 否 | 10 | 主体描边宽度，单位 px；仅 need_contour=true 且 human/product 场景生效 |
| need_crop_background | `--need-crop-background` | boolean | 否 | false | 是否裁剪透明背景到刚好包住主体；仅 human/product 场景生效，general 场景忽略 |
| output_format | `--output-format` | string | 否 | png | 输出图片格式，可选值：png、jpeg、webp |
| callback_args | `--callback-args` | string | 否 | - | 可选，回调参数 |
| client_token | `--client-token` | string | 否 | - | 可选，用于幂等，默认幂等，用户可根据需求进行调整 |

## 调用示例
```bash
mediakit-cli image remove-image-background \
  --image-url https://example.com/image_url \
  --scene human \
  --need-contour false \
  --contour-color "#FFFFFF" \
  --contour-size 10 \
  --need-crop-background false \
  --output-format png \
  --callback-args sample-callback-args \
  --client-token demo-client-token
```

## 输出格式
```json
{
  "image_url": "https://example.com/nobg.png",
  "image_size": 102400,
  "image_format": "png",
  "image_width": 1080,
  "image_height": 1920
}
```
