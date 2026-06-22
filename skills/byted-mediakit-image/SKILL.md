---
name: byted-mediakit-image
version: "1.0.0"
license: "MIT"
description: "图像处理，涵盖图像压缩、图像增强、AI处理等能力。包含能力：image-ocr, erase-image, remove-image-background, enhance-image, evaluate-image-quality。当用户需要使用 image 域的 MediaKit CLI 能力时触发。"
permissions:
  - shell
metadata:
  requires:
    bins: ["mediakit-cli"]
  cliHelp: "mediakit-cli image --help"
  product: mediakit-cli/skills
  domain: image
  capability_count: 5
---
# Image Skills

## 前置说明

开始前必须先读取 `./reference/shared.md` 的内容，其中包含前置检查、结果处理等说明。

> 本域工具均为同步执行，调用成功后直接返回最终结果，无需 `query-task` 轮询。

## 工具列表

| 工具 | 说明 | 参数声明 | 参考文档 |
|------|------|----------|----------|
| image-ocr | 识别图片中的通用印刷体文字，返回可编辑文本、文字框坐标和置信度 | `image_url:string, callback_args?:string, client_token?:string` | [reference/image-ocr.md](reference/image-ocr.md) |
| erase-image | 自动检测并擦除图片中的常见图标、文字或指定区域内容，并对擦除区域进行背景智能填充 | `image_url:string, tool_version?:string, standard_scene?:string, standard_erase_text?:string, output_format?:string, callback_args?:string, client_token?:string` | [reference/erase-image.md](reference/erase-image.md) |
| remove-image-background | 自动识别并保留图像主体，移除背景并生成透明背景图片 | `image_url:string, scene:string, need_contour?:boolean, contour_color?:string, contour_size?:integer, need_crop_background?:boolean, output_format?:string, callback_args?:string, client_token?:string` | [reference/remove-image-background.md](reference/remove-image-background.md) |
| enhance-image | 基于图像内容理解智能决策，全方位提升图片分辨率、清晰度与色彩表现 | `image_url:string, tool_version?:string, multiple?:number, target_width?:integer, target_height?:integer, callback_args?:string, client_token?:string` | [reference/enhance-image.md](reference/enhance-image.md) |
| evaluate-image-quality | 对输入图片进行主客观画质和美学评分 | `image_url:string, tool_version?:string, standard_evaluate_items?:array<string>, callback_args?:string, client_token?:string` | [reference/evaluate-image-quality.md](reference/evaluate-image-quality.md) |
