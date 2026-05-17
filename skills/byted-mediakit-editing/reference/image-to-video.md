# 图片转视频

## 能力描述
多张图片生成动画视频。

## 执行方式

| 项目 | 说明 |
|------|------|
| Domain | `editing` |
| Tool | `image-to-video` |
| 是否异步 | `是` |
| 是否支持 local | `否` |
| 模式说明 | cloud only；可通过 `--cloud` 强制当前调用。 |
| 幂等行为 | 如命令支持 `client_token` 与 `callback_args`，重试时复用同一组值；强制重跑时更换新的 `client_token`。 |

## 参数
| 参数 | CLI flag | 类型 | 必填 | 默认值 | 说明 |
|------|----------|------|------|--------|------|
| images | `--images` | array<object{image_url:string, duration?:number, animation_type?:string, animation_in?:number, animation_out?:number}> | 是 | - | 待合成的图片列表，Array<Image>类型。最少传入1个，最多传入100个 子项说明：待合成的图片。Image类型 CLI 传参时请使用 JSON 字符串，并用单引号包裹整个值，例如 `--images '[{"image_url": "https://example.com/image_url", "duration": 1.0, "animation_type": "<animation_type>", "animation_in": 1.0, "animation_out": 1.0}]'`。 |
| images[].image_url | `--images[].image_url` | string | 是 | - | 输入图片。String 类型，支持http://xxx或https://xxx格式 URL |
| images[].duration | `--images[].duration` | number | 否 | - | 图片播放时长，选填，默认值：3，单位：秒，支持 2 位小数。 |
| images[].animation_type | `--images[].animation_type` | string | 否 | - | 图片的动画类型，选填，不填时无动画效果。 move_up：向上移动 move_down：向下移动 move_left：向左移动 move_right：向右移动 zoom_in：缩小 zoom_out：放大 |
| images[].animation_in | `--images[].animation_in` | number | 否 | - | 动画结束时间，选填，支持2位小数。默认为图片展示时长，表示动画随图片播放同时结束，单位：秒 |
| images[].animation_out | `--images[].animation_out` | number | 否 | - | 动画结束时间，选填，支持2位小数。默认为图片展示时长，表示动画随图片播放同时结束，单位：秒 |
| transitions | `--transitions` | array<string> | 否 | - | 转场效果 ID，Array<string> 类型。如果不提供，则没有转场。当视频数量超过转场数量 2 个及以上时，系统将自动循环使用转场。例如有 10 个视频，2 种转场效果，那么在 9 处拼接点上，这 2 种转场效果将被依次循环使用。 子项说明：转场效果 ID 分类：交替出场，ID：1182359 分类：旋转放大，ID：1182360 分类：泛开，ID：1182358 分类：六角形，ID：1182365 分类：故障转换，ID：1182367 分类：飞眼，ID：1182368 分类：梦幻放大，ID：1182369 分类：开门展现，ID：1182370 分类：立方转换，ID：1182373 分类：透镜变换，ID：1182374 分类：晚霞转场，ID：1182375 分类：圆形交替，ID：1182378 |
| callback_args | `--callback-args` | string | 否 | - | 可选，回调参数 |
| client_token | `--client-token` | string | 否 | - | 可选，用于幂等，默认幂等，用户可根据需求进行调整 |

## 调用示例
```bash
mediakit-cli editing image-to-video \
  --images '[{"image_url": "https://example.com/image_url", "duration": 1.0, "animation_type": "<animation_type>", "animation_in": 1.0, "animation_out": 1.0}]' \
  --transitions item_1,item_2 \
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

- 当前命令：`mediakit-cli editing image-to-video`
- 推荐查询：`mediakit-cli shared query-task --task-id <task_id>`
