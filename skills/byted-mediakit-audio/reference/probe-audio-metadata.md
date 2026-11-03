# 音频元信息获取

## 能力用途

探测输入音频 URL，输出标准化媒资元信息，用于获取音频元信息。

## 参数填写规则

- 可选参数仅在用户明确指定，或可从用户意图准确确定时填写；不得伪造。不能准确确定时省略，确为正确完成任务所必需时先向用户澄清。
- 文档派生的 Cloud 公共请求字段只在用户明确提供时传递；不得由 Agent 生成、推断或补写。
- Local 仅使用 Local 参数；Cloud-only 或当前未实现字段不得传入。
## Cloud

### 命令与生命周期

- 命令：`mediakit-cli --cloud audio probe-audio-metadata`
- 生命周期：异步
- 返回方式：返回 `task_id`，再查询终态结果。

### 参数

| 参数路径 | CLI flag | 类型 | 必填 | 默认值 | 枚举/范围/结构 | 说明 |
| --- | --- | --- | --- | --- | --- | --- |
| `audio_url` | `--audio-url` | string | 是 | - | - | 待探测的音频 URL，支持 mp3、m4a、wav、wma、amr、aac、ogg、flac 等音频格式；支持公网 HTTP/HTTPS URL、本地文件路径、火山引擎视频点播 vod:// 和火山引擎对象存储 tos:// 四种输入协议。 |
| `callback_args` | `--callback-args` | string | 否 | - | - | 自定义回调参数；任务完成时会通过事件回调原样返回，用于关联业务；字段长度最大为 512 字节。 仅在用户明确提供该值时传递；不得由 Agent 生成、推断或补写。 |
| `callback_url` | `--callback-url` | string | 否 | - | - | 用于接收该任务结果回调的 URL 地址；提供后优先级高于全局回调地址；地址必须以 http:// 或 https:// 开头。 仅在用户明确提供该值时传递；不得由 Agent 生成、推断或补写。 |
| `client_token` | `--client-token` | string | 否 | - | - | 用户请求凭证，用于幂等控制；大小写敏感，长度不超过 64 个 ASCII 可打印字符。 用于控制异步任务幂等；同步任务不支持幂等校验。异步任务首次提交时默认不传，服务端仍会对相同核心参数执行 24 小时默认幂等。用户明确指定 token 时原样使用。因网络超时、响应丢失等原因重试同一次提交时，原请求有 token 则复用原值，原请求未传则继续不传。任务已有结果后，用户要求“结果不对，重试”“重新跑一次”或“重新发起任务”时，必须生成并传递新的 token，避免返回原任务。新对话中用户要求“重新发起任务”时，若当前上下文具备完整任务参数，直接生成新 token 并提交；若参数不足，先向用户澄清，补齐后再使用新 token 发起，不得直接省略 token。新任务后续发生提交重试时复用该新 token。已有 task_id 且任务尚未完成时，只查询原任务，不得重新提交。CLI runtime 不判断用户意图，也不自动生成 token。 |
| `queue_id` | `--queue-id` | string | 否 | - | - | 任务提交的目标队列 ID；不传时默认使用系统自动创建的队列 ID。可将不同业务或优先级的任务提交到不同队列，以按队列对应的项目进行分账。队列可创建和管理，系统会自动分配队列 ID。 仅在用户明确提供该值时传递；不得由 Agent 生成、推断或补写。 |

### 调用示例

```bash
MEDIAKIT_RUNTIME=<当前宿主> \
  mediakit-cli --cloud audio probe-audio-metadata \
  --audio-url <audio_url>
```

仅使用用户真实输入替换占位符；可选 flag 遵守参数填写规则，不得编造 URL、文件、枚举或业务参数。

### 返回结果

| 字段路径 | 类型 | 必含 | 模式 | 说明 |
| --- | --- | --- | --- | --- |
| `request_id` | string | 否 | Cloud | 请求标识；仅在后端返回时出现。 |
| `task_id` | string | 是 | Cloud | 异步任务的唯一标识，用于查询任务状态并获取最终结果。 |
| `task_type` | string | 否 | Cloud | 任务类型；仅在后端实际返回非空值时出现。 |
| `audio_stream_meta` | object / null | 是 | Cloud 终态 | 主音频流元信息。无音频流时返回 null。 |
| `audio_stream_meta.bitrate` | number / null | 否 | Cloud 终态 | 音频流码率，单位为 bps。 |
| `audio_stream_meta.channels` | number / null | 否 | Cloud 终态 | 音频声道数。 |
| `audio_stream_meta.codec` | string / null | 否 | Cloud 终态 | 音频编码格式，例如 aac。 |
| `audio_stream_meta.duration` | number / null | 否 | Cloud 终态 | 音频流时长，单位为秒。 |
| `audio_stream_meta.sample_rate` | number / null | 否 | Cloud 终态 | 音频采样率，单位为 Hz。 |
| `format_meta` | object / null | 是 | Cloud 终态 | 容器层元信息，包含封装格式、码率、时长、大小等。 |
| `format_meta.bitrate` | number / null | 否 | Cloud 终态 | 容器码率，单位为 bps。 |
| `format_meta.container` | string / null | 否 | Cloud 终态 | 容器格式（封装格式），例如 mp3。 |
| `format_meta.duration` | number / null | 否 | Cloud 终态 | 容器声明的时长，单位为秒。 |
| `format_meta.md5` | string / null | 否 | Cloud 终态 | 文件 MD5 值（如可获取）。 |
| `format_meta.size` | number / null | 否 | Cloud 终态 | 文件大小，单位为 Byte。 |

Cloud 调用成功后读取 `task_id`，再查询终态结果：

```bash
MEDIAKIT_RUNTIME=<当前宿主> \
  mediakit-cli shared query-task --task-id <task_id> --poll-complete
```

### 机器合同

以下命令只读取本模式的实时 help/schema，不发起业务调用：

```bash
mediakit-cli --cloud audio probe-audio-metadata --help
mediakit-cli --cloud audio probe-audio-metadata --schema
```

## Local

### 命令与生命周期

- 命令：`mediakit-cli --local audio probe-audio-metadata`
- 生命周期：同步
- 返回方式：直接返回本地结果，不产生 `task_id`。

### 参数

| 参数路径 | CLI flag | 类型 | 必填 | 默认值 | 枚举/范围/结构 | 说明 |
| --- | --- | --- | --- | --- | --- | --- |
| `audio_url` | `--audio-url` | string | 是 | - | - | 待探测的音频公网 HTTP/HTTPS URL。 |

### Local CLI 选项

| CLI flag | 必填 | 说明 |
| --- | --- | --- |
| `--output-path` | 否 | 本地文件输出目录或完整输出文件路径；仅在用户明确指定输出位置时传递。 |

### 调用示例

```bash
MEDIAKIT_RUNTIME=<当前宿主> \
  mediakit-cli --local audio probe-audio-metadata \
  --audio-url <audio_url>
```

仅使用用户真实输入替换占位符；可选 flag 遵守参数填写规则，不得编造 URL、文件、枚举或业务参数。

### 返回结果

| 字段路径 | 类型 | 必含 | 模式 | 说明 |
| --- | --- | --- | --- | --- |
| `audio_stream_meta` | object / null | 是 | Local | 主音频流元信息。无音频流时返回 null。 |
| `audio_stream_meta.bitrate` | number / null | 否 | Local | 音频流码率，单位为 bps。 |
| `audio_stream_meta.channels` | number / null | 否 | Local | 音频声道数。 |
| `audio_stream_meta.codec` | string / null | 否 | Local | 音频编码格式，例如 aac。 |
| `audio_stream_meta.duration` | number / null | 否 | Local | 音频流时长，单位为秒。 |
| `audio_stream_meta.sample_rate` | number / null | 否 | Local | 音频采样率，单位为 Hz。 |
| `format_meta` | object / null | 是 | Local | 容器层元信息，包含封装格式、码率、时长、大小等。 |
| `format_meta.bitrate` | number / null | 否 | Local | 容器码率，单位为 bps。 |
| `format_meta.container` | string / null | 否 | Local | 容器格式（封装格式），例如 mp3。 |
| `format_meta.duration` | number / null | 否 | Local | 容器声明的时长，单位为秒。 |
| `format_meta.md5` | string / null | 否 | Local | 文件 MD5 值（如可获取）。 |
| `format_meta.size` | number / null | 否 | Local | 文件大小，单位为 Byte。 |

### 机器合同

以下命令只读取本模式的实时 help/schema，不发起业务调用：

```bash
mediakit-cli --local audio probe-audio-metadata --help
mediakit-cli --local audio probe-audio-metadata --schema
```
