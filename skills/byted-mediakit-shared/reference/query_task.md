# 查询任务

## 能力描述
异步任务结果查询通过task_id查询任务信息

## 执行方式

| 项目 | 说明 |
|------|------|
| Domain | `shared` |
| Tool | `query-task` |
| 是否异步 | `否` |
| 是否支持 local | `否` |
| 模式说明 | cloud only；当前命令用于查询云端异步任务状态。 |
| 幂等行为 | 当前命令无额外幂等参数要求。 |

## 参数
| 参数 | CLI flag | 类型 | 必填 | 默认值 | 说明 |
|------|----------|------|------|--------|------|
| task_id | `--task-id` | string | 是 | - | 需要查询的任务 ID。 |
| poll_interval_seconds | `--poll-interval-seconds` | number | 否 | 10 | 轮询间隔秒数。 |
| max_poll_attempts | `--max-poll-attempts` | integer | 否 | 0 | 最大轮询次数，0 表示不自动轮询。 |
| poll_complete | `--poll-complete` | boolean | 否 | - | 是否轮询直到任务完成。 |

## 调用示例
```bash
mediakit-cli shared query-task \
  --task-id task_demo_001 \
  --poll-interval-seconds 10 \
  --max-poll-attempts 12
```

## 受理响应
异步媒体处理命令提交成功后通常先返回如下受理结果：

```json
{
  "task_id": "task_demo_001",
  "request_id": "req_demo_001"
}
```

## 输出格式
```json
{
  "success": true,
  "task_id": "task_demo_001",
  "task_type": "extract-audio",
  "status": "completed",
  "result": {
    "audio_url": "https://example.com/audio.m4a"
  },
  "request_id": "req_demo_001"
}
```

## 任务结果查询
当前命令本身就是任务查询入口，无需再次查询。

- 当前命令：`mediakit-cli shared query-task`
