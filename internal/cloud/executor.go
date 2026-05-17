package cloud

import (
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

// Executor is the unified cloud execution entry for generated capability handlers.
type Executor struct{}

const (
	queryTaskCommand     = "query-task"
	queryTaskIDParam     = "task_id"
	pollIntervalParam    = "poll_interval_seconds"
	maxPollAttemptsParam = "max_poll_attempts"
	pollCompleteParam    = "poll_complete"
)

type queryTaskPollOptions struct {
	PollInterval time.Duration
	MaxAttempts  int
	PollComplete bool
}

func Execute(cmd *cobra.Command, command string, params map[string]any, apiKey string, endpoint string) error {
	if strings.TrimSpace(apiKey) == "" {
		return writeJSON(cmd.OutOrStdout(), errorResponse(fmt.Errorf("cloud 执行需要配置 MEDIAKIT_API_KEY"), "", ""))
	}

	normalizedCommand := normalizeCommand(command)
	normalizedParams := normalizeParams(params)
	pollOptions, requestParams, err := splitQueryTaskOptions(normalizedCommand, normalizedParams)
	if err != nil {
		return writeJSON(cmd.OutOrStdout(), errorResponse(err, extractTaskID(requestParams), ""))
	}

	client := NewClient(apiKey, endpoint)
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

	return options, requestParams, nil
}

func maybePollQueryTask(client *Client, requestParams map[string]any, response map[string]any, options queryTaskPollOptions) (map[string]any, error) {
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

	if options.PollComplete {
		for {
			time.Sleep(options.PollInterval)
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

	for attempts := 0; attempts < options.MaxAttempts; attempts++ {
		time.Sleep(options.PollInterval)
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

func formatCommandResponse(command string, response map[string]any) map[string]any {
	if command == queryTaskCommand {
		return queryTaskResponse(response)
	}
	return asyncTaskResponse(response)
}

func asyncTaskResponse(result map[string]any) map[string]any {
	output := map[string]any{
		"task_id": "",
	}
	if taskID, ok := result["task_id"]; ok {
		output["task_id"] = strings.TrimSpace(fmt.Sprint(taskID))
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
	if requestID := strings.TrimSpace(fmt.Sprint(result["request_id"])); requestID != "" && requestID != "<nil>" {
		output["request_id"] = requestID
	}
	if status := strings.TrimSpace(fmt.Sprint(result["status"])); status != "" && status != "<nil>" {
		output["status"] = status
	}
	taskResult, ok := result["result"].(map[string]any)
	if !ok {
		return output
	}
	for key, value := range taskResult {
		output[key] = value
	}
	return output
}

func isTerminalTaskStatus(response map[string]any) bool {
	status := strings.ToLower(strings.TrimSpace(fmt.Sprint(response["status"])))
	switch status {
	case "completed", "succeeded", "success", "failed", "cancelled", "canceled":
		return true
	default:
		return false
	}
}

func writeJSON(writer io.Writer, value any) error {
	encoder := json.NewEncoder(writer)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")
	return encoder.Encode(value)
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
