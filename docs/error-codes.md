# mediakit-cli 错误码与退出码契约

本文档描述 `mediakit-cli` 在 cloud / local 两种执行路径下的错误协议、字段含义与进程退出码（exit code）规则，作为 SDK / Skill / Plugin 等上层调用方解析的稳定契约。

## 一、退出码总览

| 退出码 | 含义                            | 触发场景                                                                                                       |
| ------ | ------------------------------- | -------------------------------------------------------------------------------------------------------------- |
| `0`    | 命令成功                        | 所有路径成功完成；同步成功 / 异步提交成功（拿到 task_id）/ query-task 终态为 `completed`                       |
| `1`    | 业务失败或框架/参数错误         | 任意 stdout JSON 出现 `success: false`（cloud）、`error` 字段（local）、`status: failed/canceled/cancelled`（query-task 终态），或参数解析失败 |

> `mediakit-cli` 不再使用退出码区分"框架错误"和"业务错误"；上层调用方应**优先解析 stdout JSON**，把退出码当作辅助信号。

## 二、cloud 路径（远程 API）

### 2.1 输出落点

- 业务结果与业务错误：stdout（一个 JSON 对象，pretty 缩进）
- stderr：保留给 update notice、调试日志；**不再写业务错误**

### 2.2 成功响应（同步 / 异步提交）

异步主任务提交成功时（HTTP 2xx + `success=true`），CLI 直接透传：

```json
{
  "success": true,
  "task_id": "task_xxx",
  "request_id": "req_xxx",
  "result": { "...": "..." }
}
```

退出码：`0`。

### 2.3 业务失败响应

任意以下情形都视为业务失败，CLI 写完 JSON 后退出码 `1`：

| 情形                                              | 判定字段                                                       |
| ------------------------------------------------- | -------------------------------------------------------------- |
| 主调用 HTTP 2xx 但 `success=false`                | `success == false`                                             |
| 主调用 HTTP ≥ 400 网络/服务错误                   | `errorResponse(...)` 统一写 `success: false` + `error: {...}`  |
| `query-task` 终态为 `failed` / `canceled` / `cancelled` | `status` 字段（注意 queryTaskResponse 没有 `success` 字段） |

业务失败 JSON 结构：

```json
{
  "success": false,
  "error": {
    "code": "InvalidParameter",
    "message": "video_url is required"
  },
  "task_id": "task_xxx",
  "request_id": "req_xxx"
}
```

### 2.4 query-task 终态字段

`mediakit-cli shared query-task` 返回的 `status` 取值：

| status      | 含义                       | 退出码 |
| ----------- | -------------------------- | ------ |
| `running`   | 进行中（非终态）           | `0`    |
| `queued`    | 排队中（非终态）           | `0`    |
| `completed` | 已完成（终态，成功）       | `0`    |
| `failed`    | 已失败（终态，业务失败）   | `1`    |
| `canceled`  | 已取消（终态，业务失败）   | `1`    |
| `cancelled` | 同 `canceled`（兼容拼写）  | `1`    |

只有 `failed/canceled/cancelled` 三个状态被识别为"业务失败终态"，退出码为 `1`；其他状态退出码 `0`。

## 三、local 路径（本地执行）

### 3.1 输出落点

- 成功结果：stdout（平铺业务字段，与 swagger response schema 对齐，详见 Rule 23）
- 业务失败：stdout 写 `{"error": {...}}` 结构化错误
- stderr：保留给 update notice、调试日志

### 3.2 业务失败响应

```json
{
  "error": {
    "type": "InvalidParameter | SecurityViolation | EnvironmentError | ExecutionError",
    "code": "MissingRequiredParam | InvalidParamType | ParamOutOfRange | ParamInsufficient | UnsupportedValue | ForbiddenOperation | NotWhitelisted | UnsafeCharacters | HandlerNotImplemented | LocalUnsupported | DependencyMissing | ExecutionFailed | DownloadFailed | Unknown",
    "message": "<原始错误信息（不截断、不摘要）>"
  }
}
```

退出码：`1`。

`type` 与 `code` 取值范围见 `AGENTS.md` Rule 25。

## 四、上层调用方解析建议

1. 优先解析 stdout JSON：
   - cloud：`success === false` 即业务失败；终态 `status` 为 `failed/canceled/cancelled` 也视为失败
   - local：存在非空 `error` 字段即业务失败
2. 退出码作为辅助信号：`exit code === 0` 表示无业务失败信号，`exit code === 1` 表示存在业务失败或框架/参数错误
3. stderr 不再承载业务错误内容；只用来读取 update notice 与调试日志

## 五、Boolean flag 传参

所有 boolean 类型 flag 必须使用以下三种形式之一：

- 裸 flag：`--enable-foo`（等价于 `--enable-foo=true`）
- 显式赋值：`--enable-foo=true` / `--enable-foo=false`

禁止使用空格分隔写法（`--enable-foo true`）：cobra 会把空格后的 token 视为 positional 参数。`mediakit-cli` 会在该 capability 含 boolean 参数时给出明确提示。

## 六、--schema 输出中的 Async 字段

每个 capability 命令的 `--schema` 输出在 description 中固定包含一行 `Async: 是/否`；异步 capability 额外打印 `轮询命令: mediakit-cli shared query-task --task-id <id> --poll-complete`，供上层 agent 识别"需要轮询拿终态结果"的语义。
