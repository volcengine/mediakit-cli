package cloud

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
)

type APIError struct {
	StatusCode int
	Payload    map[string]any
	Body       string
}

func (e *APIError) Error() string {
	if e == nil {
		return ""
	}
	message := strings.TrimSpace(extractErrorMessage(e.Payload))
	if message == "" {
		message = strings.TrimSpace(e.Body)
	}
	if message == "" {
		message = http.StatusText(e.StatusCode)
	}
	return fmt.Sprintf("HTTP %d: %s", e.StatusCode, message)
}

func (c *Client) do(req *http.Request) (map[string]any, error) {
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	payloadBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= http.StatusBadRequest {
		return nil, newAPIError(resp.StatusCode, payloadBytes)
	}
	if len(payloadBytes) == 0 {
		return map[string]any{}, nil
	}

	var result map[string]any
	if err := json.Unmarshal(payloadBytes, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func newAPIError(statusCode int, payloadBytes []byte) error {
	apiErr := &APIError{
		StatusCode: statusCode,
		Body:       string(payloadBytes),
	}
	if len(payloadBytes) > 0 {
		var payload map[string]any
		if err := json.Unmarshal(payloadBytes, &payload); err == nil {
			apiErr.Payload = payload
		}
	}
	return apiErr
}

func errorResponse(err error, taskID string, requestID string) map[string]any {
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		if apiErr.Payload != nil && isFailureEnvelope(apiErr.Payload) {
			return mergeFailureIDs(
				businessFailureResponse(apiErr.Payload),
				taskID,
				requestID,
			)
		}
	}

	output := map[string]any{
		"success": false,
	}

	var taskType string

	if apiErr != nil {
		if apiErr.Payload != nil {
			output["error"] = apiErr.Payload
		} else {
			output["error"] = strings.TrimSpace(apiErr.Body)
		}
		if requestID == "" {
			requestID = extractRequestID(apiErr.Payload)
		}
		if taskID == "" {
			taskID = extractTaskID(apiErr.Payload)
		}
		if taskType == "" {
			taskType = extractTaskType(apiErr.Payload)
		}
	} else if err != nil {
		output["error"] = err.Error()
	}

	if output["error"] == nil || output["error"] == "" {
		output["error"] = "unknown error"
	}
	if taskID != "" {
		output["task_id"] = taskID
	}
	if taskType != "" {
		output["task_type"] = taskType
	}
	if requestID != "" {
		output["request_id"] = requestID
	}
	return output
}

// isFailureEnvelope 识别已是业务失败 envelope 的响应体：
// 同时带有 success=false 与 error 字段。此类 payload 应透传，
// 不能再整包塞进 error，否则会出现 error.error 嵌套。
func isFailureEnvelope(payload map[string]any) bool {
	if len(payload) == 0 {
		return false
	}
	if _, ok := payload["error"]; !ok {
		return false
	}
	value, ok := payload["success"]
	if !ok {
		return false
	}
	success, ok := value.(bool)
	return ok && !success
}

// isBusinessFailure 检测业务级失败：HTTP 2xx 但响应体中 success=false。
func isBusinessFailure(payload map[string]any) bool {
	if len(payload) == 0 {
		return false
	}
	value, ok := payload["success"]
	if !ok {
		return false
	}
	if success, ok := value.(bool); ok {
		return !success
	}
	return false
}

// businessFailureResponse 把业务级失败的响应体直接透传输出。
func businessFailureResponse(payload map[string]any) map[string]any {
	output := map[string]any{
		"success": false,
	}
	if errField, ok := payload["error"]; ok && errField != nil {
		output["error"] = errField
	} else {
		output["error"] = "unknown error"
	}
	if taskID := extractTaskID(payload); taskID != "" {
		output["task_id"] = taskID
	}
	if requestID := extractRequestID(payload); requestID != "" {
		output["request_id"] = requestID
	}
	return output
}

func mergeFailureIDs(output map[string]any, taskID string, requestID string) map[string]any {
	if taskID != "" {
		if existing := extractTaskID(output); existing == "" {
			output["task_id"] = taskID
		}
	}
	if requestID != "" {
		if existing := extractRequestID(output); existing == "" {
			output["request_id"] = requestID
		}
	}
	return output
}

func extractErrorMessage(payload map[string]any) string {
	if len(payload) == 0 {
		return ""
	}
	if nested, ok := payload["error"].(map[string]any); ok {
		for _, key := range []string{"message", "error", "msg"} {
			if value := strings.TrimSpace(fmt.Sprint(nested[key])); value != "" && value != "<nil>" {
				return value
			}
		}
	}
	for _, key := range []string{"message", "msg"} {
		if value := strings.TrimSpace(fmt.Sprint(payload[key])); value != "" && value != "<nil>" {
			return value
		}
	}
	if value, ok := payload["error"].(string); ok {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}

func extractTaskID(payload map[string]any) string {
	return extractStringField(payload, "task_id")
}

func extractRequestID(payload map[string]any) string {
	return extractStringField(payload, "request_id")
}

func extractTaskType(payload map[string]any) string {
	return extractStringField(payload, "task_type")
}

func extractStringField(payload map[string]any, key string) string {
	if len(payload) == 0 {
		return ""
	}
	value := strings.TrimSpace(fmt.Sprint(payload[key]))
	if value == "" || value == "<nil>" {
		return ""
	}
	return value
}
