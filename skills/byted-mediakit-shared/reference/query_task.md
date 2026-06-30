# Query Task

## Capability

Query async task information by `task_id`.

## Execution

| Item            | Value                                                                      |
| --------------- | -------------------------------------------------------------------------- |
| Domain          | `shared`                                                                   |
| Tool            | `query-task`                                                               |
| Async           | `No`                                                                       |
| Local supported | `No`                                                                       |
| Mode notes      | cloud only; this command is used to query the status of cloud async tasks. |
| Idempotency     | No additional idempotency parameters required for this command.            |

## Parameters

| Parameter             | CLI flag                  | Type    | Required | Default | Description                                        |
| --------------------- | ------------------------- | ------- | -------- | ------- | -------------------------------------------------- |
| task_id               | `--task-id`               | string  | Yes      | -       | Task ID to query.                                  |
| poll_interval_seconds | `--poll-interval-seconds` | number  | No       | 10      | Polling interval in seconds.                       |
| max_poll_attempts     | `--max-poll-attempts`     | integer | No       | 0       | Maximum polling attempts; 0 disables auto-polling. |
| poll_complete         | `--poll-complete`         | boolean | No       | -       | Whether to poll until the task is complete.        |

## Example

```bash
mediakit-cli shared query-task \
  --task-id task_demo_001 \
  --poll-interval-seconds 10 \
  --max-poll-attempts 12
```

## Acceptance response

Async media-processing commands typically return an acceptance response on successful submission:

```json
{
  "task_id": "task_demo_001",
  "request_id": "req_demo_001"
}
```

## Output format

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

## Task result lookup

This command is itself the task-query entry point; no further query is required.

- Current command: `mediakit-cli shared query-task`
