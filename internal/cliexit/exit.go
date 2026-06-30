// Package cliexit 定义 CLI 业务失败的哨兵错误。
//
// "业务失败" 在 CLI 视角下采用宽口径定义：本次命令未达成功状态。
// 涵盖三档：
//  1. 业务级失败 — cloud success=false / query-task 失败终态 / local error 字段
//  2. 框架/参数错误 — capability 参数校验失败、handler 未实现、依赖缺失
//  3. 透传错误 — 底层 modes/cloud/local 抛出的非 sentinel 错误经 writeCapabilityError 包装
//
// 三档统一映射到 exit=1，但结构化错误已经写入 stdout JSON；
// main.go 通过 errors.Is 识别此哨兵，跳过 stderr 重复打印。
//
// 详见 AGENTS.md Rule 32: CLI Exit Code Contract。
package cliexit

import "errors"

// ErrBusinessFailure 表示 CLI 本次命令未达成功状态。
//
// 由以下三个 writer 统一抛出：
//   - internal/cloud/executor.go::writeJSON（cloud 主路径与 query-task 终态）
//   - internal/local/executor.go::writeJSON（local 主路径）
//   - internal/commands/registry.go::writeCapabilityError（参数/依赖/兜底错误）
//
// cmd/mediakit/main.go 用 errors.Is 识别后：
//   - 跳过 stderr 打印（错误结构已在 stdout JSON 中）
//   - 进程退出码 os.Exit(1)
var ErrBusinessFailure = errors.New("mediakit-cli: business failure")
