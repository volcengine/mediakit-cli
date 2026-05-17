package cloud

import (
	"encoding/json"
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
	output := map[string]any{
		"error": "",
	}

	if apiErr, ok := err.(*APIError); ok {
		output["error"] = extractErrorMessage(apiErr.Payload)
		if output["error"] == "" {
			output["error"] = strings.TrimSpace(apiErr.Body)
		}
		if requestID == "" {
			requestID = extractRequestID(apiErr.Payload)
		}
		if taskID == "" {
			taskID = extractTaskID(apiErr.Payload)
		}
	} else if err != nil {
		output["error"] = err.Error()
	}

	if output["error"] == "" {
		output["error"] = "unknown error"
	}
	if taskID != "" {
		output["task_id"] = taskID
	}
	if requestID != "" {
		output["request_id"] = requestID
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
