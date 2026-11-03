package cloud

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"mediakit-cli/internal/netguard"
	"mediakit-cli/internal/surface"
)

const (
	urlReachabilityTimeout = 5 * time.Second
	allowCloudMediaURIs    = true
)

// IsReachable reports whether an HTTP(S) URL responds with 2xx/3xx.
// HEAD is preferred; 404/405/501 responses fall back to GET with
// Range: bytes=0-0 because some short-link/CDN endpoints reject HEAD while
// still allowing downloads.
func IsReachable(ctx context.Context, rawURL string) bool {
	return CheckReachable(ctx, rawURL) == nil
}

// CheckReachable returns a descriptive error when the URL is not reachable.
func CheckReachable(ctx context.Context, rawURL string) error {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return fmt.Errorf("URL 不能为空")
	}
	ctx, cancel := context.WithTimeout(ctx, urlReachabilityTimeout)
	defer cancel()

	if _, err := netguard.ValidateRemoteURL(ctx, rawURL); err != nil {
		return fmt.Errorf("URL 不可访问: %w", err)
	}
	return checkReachable(ctx, netguard.NewHTTPClient(urlReachabilityTimeout), rawURL)
}

func checkReachable(ctx context.Context, client *http.Client, rawURL string) error {
	status, err := doReachabilityRequest(ctx, client, http.MethodHead, rawURL, false)
	if err != nil {
		return fmt.Errorf("URL 不可访问: %w", err)
	}
	if status >= 200 && status < 400 {
		return nil
	}
	if !shouldFallbackToRangeGET(status) {
		return fmt.Errorf("URL 返回 HTTP %d: %s", status, rawURL)
	}
	status, err = doReachabilityRequest(ctx, client, http.MethodGet, rawURL, true)
	if err != nil {
		return fmt.Errorf("URL 不可访问: %w", err)
	}
	if status >= 200 && status < 400 {
		return nil
	}
	return fmt.Errorf("URL 返回 HTTP %d: %s", status, rawURL)
}

func shouldFallbackToRangeGET(status int) bool {
	return status == http.StatusNotFound ||
		status == http.StatusMethodNotAllowed ||
		status == http.StatusNotImplemented
}

func doReachabilityRequest(
	ctx context.Context,
	client *http.Client,
	method string,
	rawURL string,
	withRange bool,
) (int, error) {
	req, err := http.NewRequestWithContext(ctx, method, rawURL, nil)
	if err != nil {
		return 0, err
	}
	if withRange {
		req.Header.Set("Range", "bytes=0-0")
	}
	resp, err := client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	return resp.StatusCode, nil
}

// precheckReachableMediaURLs validates every HTTP(S) media input declared by
// the command's embedded Surface schema. Field names are intentionally ignored:
// nested and capability-specific inputs such as mask_url or target_face_url are
// media only when their exact schema node is marked with format=media-to-*.
func precheckReachableMediaURLs(command string, params map[string]any) error {
	if len(params) == 0 {
		return nil
	}
	schema, domain := mediaInputSchemaForCommand(command)
	if len(schema) == 0 {
		return nil
	}
	if err := validateMediaInputProtocols(params, schema); err != nil {
		return err
	}
	ctx := context.Background()
	checked := map[string]struct{}{}
	return walkMediaURLValues(params, schema, domain == "image", func(value string) error {
		value = strings.TrimSpace(value)
		if value == "" {
			return nil
		}
		lower := strings.ToLower(value)
		if !strings.HasPrefix(lower, "http://") && !strings.HasPrefix(lower, "https://") {
			return nil
		}
		if _, ok := checked[value]; ok {
			return nil
		}
		checked[value] = struct{}{}
		return CheckReachable(ctx, value)
	})
}

func mediaInputSchemaForCommand(command string) (map[string]any, string) {
	if capability, ok := surface.Lookup(normalizeCommand(command)); ok {
		return capability.Parameters, capability.Domain
	}
	if support, ok := surface.LookupSupportTool(normalizeCommand(command)); ok {
		return support.InputSchema, support.Domain
	}
	return nil, ""
}

func validateMediaInputProtocols(value any, schema map[string]any) error {
	switch typed := value.(type) {
	case map[string]any:
		properties, _ := schema["properties"].(map[string]any)
		for childKey, childValue := range typed {
			childSchema, _ := properties[childKey].(map[string]any)
			if len(childSchema) == 0 {
				continue
			}
			if err := validateMediaInputProtocols(childValue, childSchema); err != nil {
				return err
			}
		}
	case []any:
		itemSchema, _ := schema["items"].(map[string]any)
		for _, childValue := range typed {
			if err := validateMediaInputProtocols(childValue, itemSchema); err != nil {
				return err
			}
		}
	case []string:
		itemSchema, _ := schema["items"].(map[string]any)
		inputSchema := schema
		if isMediaToInputSchema(itemSchema) {
			inputSchema = itemSchema
		}
		if !isMediaToInputSchema(inputSchema) {
			return nil
		}
		for _, childValue := range typed {
			if err := validateMediaInputProtocolString(childValue); err != nil {
				return err
			}
		}
	case string:
		if isMediaToInputSchema(schema) {
			return validateMediaInputProtocolString(typed)
		}
	}
	return nil
}

func validateMediaInputProtocolString(value string) error {
	value = strings.TrimSpace(value)
	lower := strings.ToLower(value)
	if value == "" || !strings.Contains(lower, "://") ||
		strings.HasPrefix(lower, "http://") ||
		strings.HasPrefix(lower, "https://") {
		return nil
	}
	if allowCloudMediaURIs && (strings.HasPrefix(lower, "vod://") ||
		strings.HasPrefix(lower, "tos://")) {
		return nil
	}
	return fmt.Errorf("媒体输入不支持该 URI 协议: %s", schemeOf(value))
}

func walkMediaURLValues(value any, schema map[string]any, imageDomain bool, validate func(string) error) error {
	switch typed := value.(type) {
	case map[string]any:
		properties, _ := schema["properties"].(map[string]any)
		for childKey, childValue := range typed {
			childSchema, _ := properties[childKey].(map[string]any)
			if len(childSchema) == 0 {
				continue
			}
			if err := walkMediaURLValues(childValue, childSchema, imageDomain, validate); err != nil {
				return err
			}
		}
	case []any:
		itemSchema, _ := schema["items"].(map[string]any)
		if len(itemSchema) == 0 {
			return nil
		}
		for _, childValue := range typed {
			if err := walkMediaURLValues(childValue, itemSchema, imageDomain, validate); err != nil {
				return err
			}
		}
	case []string:
		itemSchema, _ := schema["items"].(map[string]any)
		inputSchema := schema
		if isMediaToInputSchema(itemSchema) {
			inputSchema = itemSchema
		}
		if !isMediaToInputSchema(inputSchema) || imageDomain {
			return nil
		}
		for _, childValue := range typed {
			if err := validate(childValue); err != nil {
				return err
			}
		}
	case string:
		if isMediaToInputSchema(schema) && !imageDomain {
			return validate(typed)
		}
	}
	return nil
}

func isMediaToInputSchema(schema map[string]any) bool {
	format, _ := schema["format"].(string)
	return strings.HasPrefix(strings.ToLower(strings.TrimSpace(format)), "media-to-")
}
