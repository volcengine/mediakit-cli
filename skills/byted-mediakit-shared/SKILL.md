---
name: byted-mediakit-shared
version: '1.0.0'
license: 'MIT'
description: '1. mediakit-cli: supports a variety of operations such as audio/video processing, editing, and images, with some capabilities covering both cloud and local modes; 2. mediakit-cli shared: environment checks, initialization config, command structure, authentication config, async task responses, and error handling.'
permissions:
  - shell
metadata:
  requires:
    bins: ['mediakit-cli']
  cliHelp: 'mediakit-cli --help'
  product: mediakit-cli/skills
  domain: shared
  capability_count: 14
---

# MediaKit Shared Rules

This skill explains how to operate on media resources via `mediakit-cli`, and the shared rules and caveats that apply to every call.

## Prerequisites

### Install the CLI

Before first use, confirm the CLI is installed:

```bash
# Install
npm install -g @volcengine/mediakit-cli

# Verify
mediakit-cli --version
```

### Authentication check

Priority: environment variables > config file (path: `~/.mediakit/config.json`).

#### Field reference

- Environment variables / config file: `MEDIAKIT_API_KEY`, `MEDIAKIT_ENDPOINT`, `MEDIAKIT_SURFACE`, `MEDIAKIT_RUNTIME`

| Variable            | Required             | Description                                                                                                                                              |
| ------------------- | -------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `MEDIAKIT_API_KEY`  | Required (cloud mode) | API authentication token                                                                                                                                 |
| `MEDIAKIT_ENDPOINT` | Optional             | API endpoint                                                                                                                                             |
| `MEDIAKIT_SURFACE`  | Optional             | Request-source Header `x-surface`. Defaults to `cli`. Skill should set `skill`, Plugin should set `plugin`; CLI reports them as `cli/skill` or `cli/plugin`. |
| `MEDIAKIT_RUNTIME`  | Optional             | Request-source Header `x-runtime`. Set to host name such as `claude` / `arkclaw`; falls back to environment probing or `unknown` when not configured.    |

When any required field is missing, stop execution and output the full list of missing items together with fix suggestions.

Cloud calls automatically carry `x-surface` / `x-runtime`. Header priority: environment variables > `~/.mediakit/config.json` > defaults / environment probing. When this Skill/Plugin invokes cloud capabilities through `mediakit-cli`, the runtime should inject `MEDIAKIT_SURFACE=skill|plugin` and `MEDIAKIT_RUNTIME=<host>`. The CLI keeps the original product prefix and reports `x-surface=cli/skill` or `x-surface=cli/plugin`. If no explicit configuration is provided, the CLI defaults to `x-surface=cli`; `x-runtime` falls back to `IDENTITY_NAME` / `OPENCLAW_SERVICE_MARKER` environment probing, and finally to `unknown`.

### Source-reporting constraints

- When a Skill invokes `mediakit-cli`, it must explicitly set `MEDIAKIT_SURFACE=skill` and must not rely on the user's pre-existing environment variables.
- When a Plugin invokes `mediakit-cli`, it must explicitly set `MEDIAKIT_SURFACE=plugin` and must not reuse the Skill value.
- The host runtime identifier should also be explicitly set via `MEDIAKIT_RUNTIME=<host>`; if unset, the CLI falls back to environment probing or `unknown`.

```bash
MEDIAKIT_SURFACE=skill MEDIAKIT_RUNTIME=<runtime> mediakit-cli editing add-image-to-video

MEDIAKIT_SURFACE=plugin MEDIAKIT_RUNTIME=<runtime> mediakit-cli editing add-image-to-video
```

## CLI usage

### Initialization

Run the init wizard before first use:

```bash
mediakit-cli init
```

For non-interactive initialization (Agents), pass the request-source and runtime configuration explicitly:

```bash
mediakit-cli init --mode cloud-first --api-key <key> --runtime <runtime> --surface cli --yes
mediakit-cli init --mode local-first --api-key <key> --endpoint <url> --output-path ~/mediakit-output --runtime <runtime> --surface cli --credential-store config --yes
```

Common commands after initialization:

```bash
# Show current configuration
mediakit-cli config show

# Switch the default mode to local-first
mediakit-cli config set mode local-first

# Switch the default mode to cloud-first
mediakit-cli config set mode cloud-first

# Refresh environment checks and view dependency status
mediakit-cli doctor
```

### Command structure

MediaKit CLI consistently uses the `domain + tool` invocation shape:

```bash
mediakit-cli {domain} {tool} [flags]
```

Common help commands:

```bash
# List all domains
mediakit-cli --domains

# List tools in a given domain
mediakit-cli {domain} --help

# Show parameters of a specific tool
mediakit-cli {domain} {tool} --help

# Dynamically discover tool capability and return structure
mediakit-cli {domain} {tool} --schema
mediakit-cli --local {domain} {tool} --schema
```

Domains currently covered by this product: `editing`, `video`.

### Schema discovery

Every capability command supports `--schema`, used by Agents to dynamically read tool capabilities. Required business parameters are not enforced when `--schema` is used.

The returned structure contains:

- `name`: tool name in snake_case, e.g. `add_image_to_video`
- `description`: tool description, automatically including `Mode` and `Async` information
- `input_schema`: input parameter JSON Schema
- `output_schema`: return structure for the current execution mode

Output disambiguation rules:

- By default the return surface is resolved against the global `mode` configuration
- `--local ... --schema` outputs the local-mode return surface; local mode returns the final result fields directly
- Cloud async tools output `task_id` / `request_id`, and describe the completed-state result of `query-task` under `final_result`
- `query-task` is cloud only; its schema describes task status plus the completed-state result

Examples:

```bash
mediakit-cli editing trim-video --schema
mediakit-cli --local editing trim-video --schema
```

### Per-invocation mode override

In addition to `config set mode` for setting the default mode, you can override the mode for a single command only:

```bash
mediakit-cli --local editing add-image-to-video

mediakit-cli --cloud editing add-image-to-video
```

Additional rules:

- `--local` / `--cloud` only affect the current command and do not modify the global `config.mode`
- `--local` and `--cloud` are mutually exclusive and cannot be passed together

## Async tasks

When an async media-processing task is accepted successfully, the response contains a `task_id` field. Use the `shared query-task` command to poll the result.

```bash
mediakit-cli shared query-task --task-id <task_id>
```

## local / cloud constraints

- `query-task` is a **cloud only** tool
- local mode does not support `query-task`
- The current capability set runs primarily in cloud mode; when an explicit declaration is required, prefer `--cloud`

### Cloud-mode media input notes

- When a command runs with `--cloud` or under the `cloud-first` strategy, media input parameters (`video_url`, `audio_url`, `image_url`, `subtitle_url`, `sub_image_url`, and their corresponding array / object sub-fields) accept `http://` / `https://` URLs, `mediakit://...` file_ids, or local file paths
- `http://` / `https://` URLs and `mediakit://...` file_ids are submitted as-is; local file paths are first uploaded by the CLI as `mediakit://...` file_ids and then submitted to the cloud tool
- Parameter descriptions in each tool's reference are sourced from the APIHub/OpenAPI raw field descriptions; even when they show public URLs or HTTP/HTTPS URLs, that only reflects the resource form the cloud API ultimately receives — it does not restrict the CLI's ability to pre-process local paths in cloud mode

### Local-mode notes

- Local output directory priority: `--output-path` > `MEDIAKIT_OUTPUT_PATH` > config `output_path` > `~/.mediakit/temp`
- When `--output-path` points to a concrete media filename, it is used as the final output file directly; otherwise the filename is built as `{source_name}_{tool_name}.{ext}`, with a 6-digit random suffix appended on collision
- When no filename can be extracted from the input URL or path, fall back to `{tool_name}-{UnixNano}.{ext}`
- Local mode depends on `ffmpeg` / `ffprobe`; when missing, the error response includes an `install_guide`
- Local-mode media processing output must conform to the corresponding interface response schema; internal execution metadata must not be emitted

### Error responses

- CLI cloud mode forwards the original error object returned by the API as-is, without extracting `message`
- CLI local mode returns a structured error: `{"error":{"type":"...","code":"...","message":"..."}}`
- MCP `error_response` forwards the original error content as-is; a dict is used directly as the `error` field value

## Idempotency parameter maintenance

| Parameter        | Purpose                | Maintenance guidance                                                                |
| ---------------- | ---------------------- | ----------------------------------------------------------------------------------- |
| `client_token`   | Explicit idempotency   | Reuse the same value on request retries; use a new unique value to force re-execute |
| `callback_args`  | Passthrough to callback | Maintain together with `client_token` to ease callback reconciliation and retry tracking |

Additional rules:

- `client_token` must not exceed 64 characters
- `callback_args` is useful for callback passthrough and reconciliation tracking

## Polling strategy

| Parameter               | Description                       | Default |
| ----------------------- | --------------------------------- | ------- |
| `poll-interval-seconds` | Polling interval                  | 10s     |
| `max-poll-attempts`     | Polling attempts; 0 disables it   | 0       |
| `poll-complete`         | Block until terminal status       | -       |
