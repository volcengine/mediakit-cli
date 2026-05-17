# 视频拼接

## 能力描述
拼接多个视频片段，支持添加转场效果。

## 执行方式

| 项目 | 说明 |
|------|------|
| Domain | `editing` |
| Tool | `concat-video` |
| 是否异步 | `是` |
| 是否支持 local | `是` |
| 模式说明 | 支持 local / cloud；可通过 `--local` 或 `--cloud` 覆盖当前命令。 |
| 幂等行为 | 如命令支持 `client_token` 与 `callback_args`，重试时复用同一组值；强制重跑时更换新的 `client_token`。 |

## 参数
| 参数 | CLI flag | 类型 | 必填 | 默认值 | 说明 |
|------|----------|------|------|--------|------|
| video_urls | `--video-urls` | array<string> | 是 | - | 待拼接的视频列表，Array<string>类型。最少传入1个，最多传入100个 子项说明：待拼接的输入视频。String 类型，支持http://xxx或https://xxx格式 URL |
| transitions | `--transitions` | array<string> | 否 | - | 转场效果 ID，Array<string> 类型。如果不提供，则没有转场。当视频数量超过转场数量 2 个及以上时，系统将自动循环使用转场。例如有 10 个视频，2 种转场效果，那么在 9 处拼接点上，这 2 种转场效果将被依次循环使用。 子项说明：转场效果 ID 分类：交替出场，ID：1182359 分类：旋转放大，ID：1182360 分类：泛开，ID：1182358 分类：六角形，ID：1182365 分类：故障转换，ID：1182367 分类：飞眼，ID：1182368 分类：梦幻放大，ID：1182369 分类：开门展现，ID：1182370 分类：立方转换，ID：1182373 分类：透镜变换，ID：1182374 分类：晚霞转场，ID：1182375 分类：圆形交替，ID：1182378 |
| callback_args | `--callback-args` | string | 否 | - | 可选，回调参数 |
| client_token | `--client-token` | string | 否 | - | 可选，用于幂等，默认幂等，用户可根据需求进行调整 |

## 调用示例
```bash
mediakit-cli editing concat-video \
  --video-urls item_1,item_2 \
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

- 当前命令：`mediakit-cli editing concat-video`
- 推荐查询：`mediakit-cli shared query-task --task-id <task_id>`
