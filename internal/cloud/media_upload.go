package cloud

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
)

type mediaUploadTarget struct {
	FileID  string
	Method  string
	URL     string
	Headers map[string]string
}

func (c *Client) uploadLocalMediaFile(command string, path string) (string, error) {
	target, err := c.requestMediaUploadTarget(command)
	if err != nil {
		return "", err
	}
	if target.FileID == "" || target.URL == "" {
		return "", fmt.Errorf("申请上传地址返回缺少 file_id 或 upload_url")
	}

	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	method := strings.ToUpper(strings.TrimSpace(target.Method))
	if method == "" {
		method = http.MethodPut
	}
	req, err := http.NewRequest(method, target.URL, file)
	if err != nil {
		return "", err
	}
	for key, value := range target.Headers {
		if strings.TrimSpace(key) != "" {
			req.Header.Set(key, value)
		}
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= http.StatusBadRequest {
		return "", fmt.Errorf("上传媒体文件失败: HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	return target.FileID, nil
}

func (c *Client) requestMediaUploadTarget(command string) (mediaUploadTarget, error) {
	req, err := c.newRequest(http.MethodPost, "/api/v1/tools-sync/request-media-upload-url", nil, map[string]any{
		"tool_name": command,
	})
	if err != nil {
		return mediaUploadTarget{}, err
	}
	payload, err := c.do(req)
	if err != nil {
		return mediaUploadTarget{}, err
	}
	if isBusinessFailure(payload) {
		return mediaUploadTarget{}, fmt.Errorf("申请上传地址失败: %v", payload["error"])
	}
	result, ok := payload["result"].(map[string]any)
	if !ok {
		return mediaUploadTarget{}, fmt.Errorf("申请上传地址响应缺少 result")
	}
	return mediaUploadTarget{
		FileID:  strings.TrimSpace(fmt.Sprint(result["file_id"])),
		Method:  strings.TrimSpace(fmt.Sprint(result["method"])),
		URL:     strings.TrimSpace(fmt.Sprint(result["upload_url"])),
		Headers: parseUploadHeaders(result["upload_headers"]),
	}, nil
}

func parseUploadHeaders(value any) map[string]string {
	headers := map[string]string{}
	items, ok := value.([]any)
	if !ok {
		return headers
	}
	for _, item := range items {
		switch typed := item.(type) {
		case map[string]any:
			key := strings.TrimSpace(fmt.Sprint(firstPresent(typed, "key", "name", "header")))
			val := strings.TrimSpace(fmt.Sprint(firstPresent(typed, "value", "val")))
			if key != "" && key != "<nil>" {
				headers[key] = val
			}
		case string:
			parts := strings.SplitN(typed, ":", 2)
			if len(parts) == 2 {
				headers[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
			}
		}
	}
	return headers
}

func firstPresent(values map[string]any, keys ...string) any {
	for _, key := range keys {
		if value, ok := values[key]; ok {
			return value
		}
	}
	return ""
}
