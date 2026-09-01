package cloud

import (
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"mediakit-cli/internal/auth"
	"mediakit-cli/internal/cliexit"
	"mediakit-cli/internal/notice"
	"mediakit-cli/internal/surface"
)

// Executor is the unified cloud execution entry for generated capability handlers.
type Executor struct{}

const (
	queryTaskCommand        = "query-task"
	queryTaskIDParam        = "task_id"
	pollIntervalParam       = "poll_interval_seconds"
	maxPollAttemptsParam    = "max_poll_attempts"
	pollCompleteParam       = "poll_complete"
	maxPollTimeoutParam     = "max_poll_timeout_seconds"
)

type queryTaskPollOptions struct {
	PollInterval   time.Duration
	MaxAttempts    int
	PollComplete   bool
	MaxPollTimeout time.Duration
}

type queryTaskPoller interface {
	Call(apiName string, args map[string]any) (map[string]any, error)
}

func Execute(cmd *cobra.Command, command string, params map[string]any, authContext auth.Context, endpoint string, runtime string) error {
	normalizedCommand := normalizeCommand(command)
	normalizedParams := normalizeParams(params)
	applySeparateVoiceOutputFormatFallback(normalizedCommand, normalizedParams)
	pollOptions, requestParams, err := splitQueryTaskOptions(normalizedCommand, normalizedParams)
	if err != nil {
		return writeJSON(cmd.OutOrStdout(), errorResponse(err, extractTaskID(requestParams), ""))
	}

	client := NewClient(authContext, endpoint, runtime)
	requestParams, err = materializeCloudMediaInputs(client, normalizedCommand, requestParams)
	if err != nil {
		return writeJSON(cmd.OutOrStdout(), errorResponse(err, extractTaskID(requestParams), ""))
	}
	response, err := client.Call(normalizedCommand, requestParams)
	if err != nil {
		return writeJSON(cmd.OutOrStdout(), errorResponse(err, extractTaskID(requestParams), ""))
	}

	if normalizedCommand == queryTaskCommand {
		response, err = maybePollQueryTask(client, requestParams, response, pollOptions)
		if err != nil {
			return writeJSON(cmd.OutOrStdout(), errorResponse(err, extractTaskID(response), extractRequestID(response)))
		}
	}
	if normalizedCommand != queryTaskCommand &&
		!isSyncCommand(normalizedCommand) &&
		!isBusinessFailure(response) &&
		extractTaskID(response) == "" {
		return writeJSON(
			cmd.OutOrStdout(),
			errorResponse(
				fmt.Errorf("异步任务提交成功响应缺少 task_id"),
				"",
				extractRequestID(response),
			),
		)
	}

	return writeJSON(cmd.OutOrStdout(), formatCommandResponse(normalizedCommand, response))
}

func normalizeCommand(command string) string {
	command = strings.TrimSpace(command)
	command = strings.ReplaceAll(command, "_", "-")
	return strings.ToLower(command)
}

func normalizeParams(params map[string]any) map[string]any {
	if len(params) == 0 {
		return map[string]any{}
	}

	normalized := make(map[string]any, len(params))
	for key, value := range params {
		normalized[strings.ReplaceAll(strings.TrimSpace(key), "-", "_")] = value
	}
	return normalized
}

func applySeparateVoiceOutputFormatFallback(command string, params map[string]any) {
	if command != "separate-voice" {
		return
	}
	value, ok := params["output_format"]
	if !ok || strings.TrimSpace(fmt.Sprint(value)) == "" || fmt.Sprint(value) == "<nil>" {
		params["output_format"] = "mp3"
	}
}

func splitQueryTaskOptions(command string, params map[string]any) (queryTaskPollOptions, map[string]any, error) {
	requestParams := cloneParams(params)
	if command != queryTaskCommand {
		return queryTaskPollOptions{}, requestParams, nil
	}

	options := queryTaskPollOptions{
		PollInterval: 10 * time.Second,
	}

	if value, ok := requestParams[pollIntervalParam]; ok {
		seconds, err := parseFloat64(value)
		if err != nil {
			return queryTaskPollOptions{}, nil, fmt.Errorf("%s 必须是数字", pollIntervalParam)
		}
		if seconds <= 0 {
			return queryTaskPollOptions{}, nil, fmt.Errorf("%s 必须大于 0", pollIntervalParam)
		}
		options.PollInterval = time.Duration(seconds * float64(time.Second))
		delete(requestParams, pollIntervalParam)
	}

	if value, ok := requestParams[maxPollAttemptsParam]; ok {
		attempts, err := parseInt(value)
		if err != nil {
			return queryTaskPollOptions{}, nil, fmt.Errorf("%s 必须是整数", maxPollAttemptsParam)
		}
		if attempts < 0 {
			return queryTaskPollOptions{}, nil, fmt.Errorf("%s 不能小于 0", maxPollAttemptsParam)
		}
		options.MaxAttempts = attempts
		delete(requestParams, maxPollAttemptsParam)
	}

	if value, ok := requestParams[pollCompleteParam]; ok {
		pollComplete, err := parseBool(value)
		if err != nil {
			return queryTaskPollOptions{}, nil, fmt.Errorf("%s 必须是布尔值", pollCompleteParam)
		}
		options.PollComplete = pollComplete
		delete(requestParams, pollCompleteParam)
	}

	if value, ok := requestParams[maxPollTimeoutParam]; ok {
		seconds, err := parseFloat64(value)
		if err != nil {
			return queryTaskPollOptions{}, nil, fmt.Errorf("%s 必须是数字", maxPollTimeoutParam)
		}
		if seconds < 0 {
			return queryTaskPollOptions{}, nil, fmt.Errorf("%s 不能小于 0", maxPollTimeoutParam)
		}
		if seconds > 0 {
			options.MaxPollTimeout = time.Duration(seconds * float64(time.Second))
		}
		delete(requestParams, maxPollTimeoutParam)
	}

	return options, requestParams, nil
}

func maybePollQueryTask(client queryTaskPoller, requestParams map[string]any, response map[string]any, options queryTaskPollOptions) (map[string]any, error) {
	if isTerminalTaskStatus(response) {
		return response, nil
	}
	if !options.PollComplete && options.MaxAttempts == 0 {
		return response, nil
	}

	taskID, ok := requestParams[queryTaskIDParam]
	if !ok || fmt.Sprint(taskID) == "" {
		return nil, fmt.Errorf("%s 轮询需要 %s", queryTaskCommand, queryTaskIDParam)
	}

	pollStart := time.Now()
	deadline, hasDeadline := options.pollDeadline(pollStart)

	if options.PollComplete {
		for {
			if pollDeadlineReached(deadline, hasDeadline) {
				return response, nil
			}
			time.Sleep(options.PollInterval)
			if pollDeadlineReached(deadline, hasDeadline) {
				return response, nil
			}
			next, err := client.Call(queryTaskCommand, requestParams)
			if err != nil {
				return nil, fmt.Errorf("%s 轮询失败: %w", queryTaskCommand, err)
			}
			response = next
			if isTerminalTaskStatus(response) {
				return response, nil
			}
		}
	}

	maxAttempts := cappedMaxPollAttempts(options)
	for attempts := 0; attempts < maxAttempts; attempts++ {
		if pollDeadlineReached(deadline, hasDeadline) {
			break
		}
		time.Sleep(options.PollInterval)
		if pollDeadlineReached(deadline, hasDeadline) {
			break
		}
		next, err := client.Call(queryTaskCommand, requestParams)
		if err != nil {
			return nil, fmt.Errorf("%s 轮询失败: %w", queryTaskCommand, err)
		}
		response = next
		if isTerminalTaskStatus(response) {
			break
		}
	}

	return response, nil
}

func (o queryTaskPollOptions) pollDeadline(start time.Time) (time.Time, bool) {
	if o.MaxPollTimeout <= 0 {
		return time.Time{}, false
	}
	return start.Add(o.MaxPollTimeout), true
}

func pollDeadlineReached(deadline time.Time, enabled bool) bool {
	return enabled && !time.Now().Before(deadline)
}

func cappedMaxPollAttempts(options queryTaskPollOptions) int {
	if options.MaxAttempts <= 0 {
		return 0
	}
	if options.MaxPollTimeout <= 0 || options.PollInterval <= 0 {
		return options.MaxAttempts
	}
	maxByTimeout := int(options.MaxPollTimeout / options.PollInterval)
	if maxByTimeout < options.MaxAttempts {
		return maxByTimeout
	}
	return options.MaxAttempts
}

func formatCommandResponse(command string, response map[string]any) map[string]any {
	if command == queryTaskCommand {
		return queryTaskResponse(response)
	}
	if isBusinessFailure(response) {
		return businessFailureResponse(response)
	}
	if isSyncCommand(command) {
		return syncToolResponse(response)
	}
	return asyncTaskResponse(response)
}

func isSyncCommand(command string) bool {
	capability, ok := surface.Lookup(command)
	if !ok {
		return false
	}
	return !capability.Async
}

func syncToolResponse(result map[string]any) map[string]any {
	if len(result) == 0 {
		return map[string]any{}
	}

	output := map[string]any{}
	// 透传后端 success 字段：提交成功返回 true。
	// 失败路径在 formatCommandResponse 处已被 businessFailureResponse 截获，此处不会出现 success=false。
	if success, ok := result["success"].(bool); ok {
		output["success"] = success
	}
	if taskID := strings.TrimSpace(fmt.Sprint(result["task_id"])); taskID != "" && taskID != "<nil>" {
		output["task_id"] = taskID
	}
	if taskType := extractTaskType(result); taskType != "" {
		output["task_type"] = taskType
	}
	if requestID := strings.TrimSpace(fmt.Sprint(result["request_id"])); requestID != "" && requestID != "<nil>" {
		output["request_id"] = requestID
	}
	if status := strings.TrimSpace(fmt.Sprint(result["status"])); status != "" && status != "<nil>" {
		output["status"] = status
	}
	taskResult, ok := result["result"].(map[string]any)
	if ok {
		for key, value := range taskResult {
			output[key] = value
		}
	}
	// usage 只允许来自成功的同步 Cloud 响应 envelope；拍平 result 时先移除
	// 同名业务字段，避免 result.usage 冒充共享 Cloud 响应。
	delete(output, "usage")
	applyCloudUsage(output, result)
	return output
}

func asyncTaskResponse(result map[string]any) map[string]any {
	output := map[string]any{}
	// 透传后端 success 字段：发起任务成功返回 true，失败返回 false（详细 error 参考 error 字段）。
	if success, ok := result["success"].(bool); ok {
		output["success"] = success
	}
	if taskID := extractTaskID(result); taskID != "" {
		output["task_id"] = taskID
	}
	if taskType := extractTaskType(result); taskType != "" {
		output["task_type"] = taskType
	}
	if requestID := strings.TrimSpace(fmt.Sprint(result["request_id"])); requestID != "" && requestID != "<nil>" {
		output["request_id"] = requestID
	}
	return output
}

func queryTaskResponse(result map[string]any) map[string]any {
	if len(result) == 0 {
		return map[string]any{}
	}

	output := map[string]any{
		"task_id": strings.TrimSpace(fmt.Sprint(result["task_id"])),
	}
	if taskType := extractTaskType(result); taskType != "" {
		output["task_type"] = taskType
	}
	if requestID := strings.TrimSpace(fmt.Sprint(result["request_id"])); requestID != "" && requestID != "<nil>" {
		output["request_id"] = requestID
	}
	if status := strings.TrimSpace(fmt.Sprint(result["status"])); status != "" && status != "<nil>" {
		output["status"] = status
	}
	// query-task 失败终态：注入 success: false，统一交给 writeJSON 触发 sentinel。
	// 仅在此处做语义判定，避免污染非 query-task 路径。
	if isBusinessFailure(result) || isTerminalFailure(result) {
		output["success"] = false
		if errField, ok := result["error"]; ok && errField != nil {
			output["error"] = errField
		} else {
			output["error"] = "unknown error"
		}
	}
	taskResult, ok := result["result"].(map[string]any)
	if ok {
		for key, value := range taskResult {
			output[key] = value
		}
	}
	// query-task 只有 completed 成功终态允许透传 envelope usage。
	// pending、失败或取消终态均不得输出。
	delete(output, "usage")
	if isCompletedTaskStatus(result) &&
		!isBusinessFailure(result) &&
		!isTerminalFailure(result) {
		applyCloudUsage(output, result)
	}
	return output
}

func applyCloudUsage(output map[string]any, envelope map[string]any) {
	// usage 是 Cloud 响应 envelope 字段，不接受 result.usage 冒充。
	delete(output, "usage")
	usage, ok := surface.SharedCloudUsage(envelope["usage"])
	if !ok {
		return
	}
	output["usage"] = usage
}

// isTerminalTaskStatus 判断 query-task 是否处于终态。
//
// 协议 status 枚举：running / queued / completed / failed / canceled。
// 终态 = completed / failed / canceled（cancelled 拼写兼容）。
// 非终态 = running / queued。
func isTerminalTaskStatus(response map[string]any) bool {
	status := strings.ToLower(strings.TrimSpace(fmt.Sprint(response["status"])))
	switch status {
	case "completed", "failed", "canceled", "cancelled":
		return true
	default:
		return false
	}
}

func isCompletedTaskStatus(response map[string]any) bool {
	status := strings.ToLower(strings.TrimSpace(fmt.Sprint(response["status"])))
	return status == "completed"
}

// isTerminalFailure 判断异步任务是否处于失败终态。
// 区别于 isTerminalTaskStatus：后者还包含 completed 这一成功终态；
// 本函数仅匹配 failed / canceled（cancelled 拼写兼容），用于驱动 ErrBusinessFailure sentinel。
func isTerminalFailure(payload map[string]any) bool {
	status := strings.ToLower(strings.TrimSpace(fmt.Sprint(payload["status"])))
	switch status {
	case "failed", "canceled", "cancelled":
		return true
	default:
		return false
	}
}

// isSuccessFalse 判断 payload 中 success 字段是否显式为 false。
func isSuccessFalse(payload map[string]any) bool {
	value, ok := payload["success"]
	if !ok {
		return false
	}
	if success, ok := value.(bool); ok {
		return !success
	}
	return false
}

func writeJSON(writer io.Writer, value any) error {
	if m, ok := value.(map[string]any); ok {
		notice.Inject(m)
	}
	encoder := json.NewEncoder(writer)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(value); err != nil {
		return err
	}
	// 单信号判定：cloud 所有失败路径（HTTP 错误、success=false、query-task 失败终态）
	// 都会在最终响应中显式写入 success=false，writeJSON 只需识别该字段。
	if m, ok := value.(map[string]any); ok {
		if isSuccessFalse(m) {
			return cliexit.ErrBusinessFailure
		}
	}
	return nil
}

func cloneParams(params map[string]any) map[string]any {
	cloned := make(map[string]any, len(params))
	for key, value := range params {
		cloned[key] = value
	}
	return cloned
}

func parseBool(value any) (bool, error) {
	switch typed := value.(type) {
	case bool:
		return typed, nil
	case string:
		return strconv.ParseBool(strings.TrimSpace(typed))
	default:
		return false, fmt.Errorf("unsupported bool value %T", value)
	}
}

func parseInt(value any) (int, error) {
	switch typed := value.(type) {
	case int:
		return typed, nil
	case int8:
		return int(typed), nil
	case int16:
		return int(typed), nil
	case int32:
		return int(typed), nil
	case int64:
		return int(typed), nil
	case float32:
		return int(typed), nil
	case float64:
		return int(typed), nil
	case json.Number:
		parsed, err := typed.Int64()
		return int(parsed), err
	case string:
		parsed, err := strconv.Atoi(strings.TrimSpace(typed))
		return parsed, err
	default:
		return 0, fmt.Errorf("unsupported int value %T", value)
	}
}

func parseFloat64(value any) (float64, error) {
	switch typed := value.(type) {
	case float32:
		return float64(typed), nil
	case float64:
		return typed, nil
	case int:
		return float64(typed), nil
	case int8:
		return float64(typed), nil
	case int16:
		return float64(typed), nil
	case int32:
		return float64(typed), nil
	case int64:
		return float64(typed), nil
	case json.Number:
		return typed.Float64()
	case string:
		return strconv.ParseFloat(strings.TrimSpace(typed), 64)
	default:
		return 0, fmt.Errorf("unsupported float value %T", value)
	}
}
