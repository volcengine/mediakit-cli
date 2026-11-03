package cloud

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	cliconfig "mediakit-cli/internal/config"
	"mediakit-cli/internal/netguard"
	"mediakit-cli/internal/surface"
)

const (
	imagePrecheckTimeout  = 15 * time.Second
	imageHeaderProbeBytes = int64(64 * 1024)
	imageMetadataTimeout  = 120 * time.Second
	allowImageMediaURIs   = true
)

type imageMetadata struct {
	width  int
	height int
	format string
}

type imageMetadataResolver func(string) (imageMetadata, error)

type imageMetadataBusinessError struct {
	message string
}

func (e *imageMetadataBusinessError) Error() string {
	return e.message
}

type imageInputLimits struct {
	maxBytes       int64
	minWidth       int
	minHeight      int
	maxWidth       int
	maxHeight      int
	maxEdge        int
	minLongEdge    int
	minShortEdge   int
	maxLongEdge    int
	maxShortEdge   int
	maxPixels      int64
	allowedFormats map[string]struct{}
}

type imageConditionalLimit struct {
	param       string
	equals      string
	inputFormat string
	limits      imageInputLimits
}

func defaultImageInputLimits() imageInputLimits {
	return imageInputLimits{}
}

// precheckImageInputs rejects image inputs before Cloud materialization/upload.
// Business limits are read only from the exact command's embedded Surface
// schema. A missing x-input-limits declaration must not manufacture a shared
// size, dimension, or format policy for otherwise unrelated image tools.
func precheckImageInputs(command string, params map[string]any) error {
	return precheckImageInputsWithResolver(command, params, nil)
}

func precheckImageInputsWithMetadataFallback(client *Client, command string, params map[string]any) error {
	var home string
	resolver := newImageMetadataResolver(client, func() (string, error) {
		if home != "" {
			return home, nil
		}
		resolved, err := cliconfig.ResolveHomeDir()
		if err != nil {
			return "", err
		}
		home = resolved
		return home, nil
	})
	return precheckImageInputsWithResolver(command, params, resolver)
}

func precheckImageInputsWithResolver(command string, params map[string]any, resolver imageMetadataResolver) error {
	if len(params) == 0 {
		return nil
	}
	capability, ok := surface.Lookup(command)
	if !ok || capability.Domain != "image" {
		return nil
	}
	paramsForConditions := applySchemaDefaults(capability.Parameters, params)
	return walkImageInputValues(
		paramsForConditions,
		capability.Parameters,
		paramsForConditions,
		resolver,
	)
}

func applySchemaDefaults(parameters map[string]any, params map[string]any) map[string]any {
	if len(params) == 0 {
		return params
	}
	result := make(map[string]any, len(params)+2)
	for key, value := range params {
		result[key] = value
	}
	properties, _ := parameters["properties"].(map[string]any)
	for name, raw := range properties {
		if _, present := result[name]; present {
			continue
		}
		schema, _ := raw.(map[string]any)
		if defaultValue, ok := schema["default"]; ok {
			result[name] = defaultValue
		}
	}
	return result
}

func imageInputPolicy(schema map[string]any) imageInputLimitsWithConditions {
	policy := imageInputLimitsWithConditions{limits: defaultImageInputLimits()}
	if rawLimits, ok := schema["x-input-limits"].(map[string]any); ok {
		applyImageLimitsMap(&policy.limits, rawLimits)
	}
	if rawConditions, ok := schema["x-input-conditional-limits"].([]any); ok {
		for _, rawCondition := range rawConditions {
			condition, ok := rawCondition.(map[string]any)
			if !ok {
				continue
			}
			conditional := imageConditionalLimit{}
			if when, ok := condition["when"].(map[string]any); ok {
				conditional.param, _ = when["param"].(string)
				conditional.equals, _ = when["equals"].(string)
				conditional.inputFormat, _ = when["input_format"].(string)
			}
			applyImageLimitsMap(&conditional.limits, condition)
			if conditional.param != "" || conditional.inputFormat != "" {
				policy.conditionals = append(policy.conditionals, conditional)
			}
		}
	}
	return policy
}

type imageInputLimitsWithConditions struct {
	limits       imageInputLimits
	conditionals []imageConditionalLimit
}

func applyImageLimitsMap(limits *imageInputLimits, raw map[string]any) {
	if value, ok := integerValue(raw["max_bytes"]); ok {
		limits.maxBytes = value
	}
	if value, ok := integerValue(raw["min_width"]); ok {
		limits.minWidth = int(value)
	}
	if value, ok := integerValue(raw["min_height"]); ok {
		limits.minHeight = int(value)
	}
	if value, ok := integerValue(raw["max_width"]); ok {
		limits.maxWidth = int(value)
	}
	if value, ok := integerValue(raw["max_height"]); ok {
		limits.maxHeight = int(value)
	}
	if value, ok := integerValue(raw["max_edge"]); ok {
		limits.maxEdge = int(value)
	}
	if value, ok := integerValue(raw["min_long_edge"]); ok {
		limits.minLongEdge = int(value)
	}
	if value, ok := integerValue(raw["min_short_edge"]); ok {
		limits.minShortEdge = int(value)
	}
	if value, ok := integerValue(raw["max_long_edge"]); ok {
		limits.maxLongEdge = int(value)
	}
	if value, ok := integerValue(raw["max_short_edge"]); ok {
		limits.maxShortEdge = int(value)
	}
	if value, ok := integerValue(raw["max_pixels"]); ok {
		limits.maxPixels = value
	}
	if formats, ok := raw["allowed_formats"].([]any); ok {
		limits.allowedFormats = make(map[string]struct{}, len(formats))
		for _, rawFormat := range formats {
			if format, ok := rawFormat.(string); ok {
				limits.allowedFormats[normalizeImageFormat(format)] = struct{}{}
			}
		}
	}
}

func integerValue(value any) (int64, bool) {
	switch typed := value.(type) {
	case int:
		return int64(typed), true
	case int64:
		return typed, true
	case float64:
		return int64(typed), typed == float64(int64(typed))
	case json.Number:
		parsed, err := typed.Int64()
		return parsed, err == nil
	default:
		return 0, false
	}
}

func walkImageInputValues(
	value any,
	schema map[string]any,
	allParams map[string]any,
	resolver imageMetadataResolver,
) error {
	switch typed := value.(type) {
	case map[string]any:
		for childKey, childValue := range typed {
			childSchema := imageSchemaProperty(schema, childKey)
			if len(childSchema) == 0 {
				continue
			}
			if err := walkImageInputValues(childValue, childSchema, allParams, resolver); err != nil {
				return err
			}
		}
	case []any:
		itemSchema := imageSchemaItems(schema)
		if len(itemSchema) == 0 {
			return nil
		}
		for _, childValue := range typed {
			if err := walkImageInputValues(childValue, itemSchema, allParams, resolver); err != nil {
				return err
			}
		}
	case []string:
		itemSchema := imageSchemaItems(schema)
		inputSchema := schema
		if isImageInputSchema(itemSchema) {
			inputSchema = itemSchema
		}
		if !isImageInputSchema(inputSchema) {
			return nil
		}
		policy := imageInputPolicy(inputSchema)
		for _, childValue := range typed {
			if err := validateImageInputURLWithResolver(childValue, policy, allParams, resolver); err != nil {
				return err
			}
		}
	case string:
		if !isImageInputSchema(schema) {
			return nil
		}
		return validateImageInputURLWithPolicy(
			typed,
			imageInputPolicy(schema),
			allParams,
			resolver,
		)
	}
	return nil
}

func imageSchemaProperty(schema map[string]any, name string) map[string]any {
	if schema == nil {
		return nil
	}
	properties, _ := schema["properties"].(map[string]any)
	child, _ := properties[name].(map[string]any)
	return child
}

func imageSchemaItems(schema map[string]any) map[string]any {
	if schema == nil {
		return nil
	}
	items, _ := schema["items"].(map[string]any)
	return items
}

func isImageInputSchema(schema map[string]any) bool {
	format, _ := schema["format"].(string)
	return strings.HasPrefix(
		strings.ToLower(strings.TrimSpace(format)),
		"media-to-",
	)
}

func validateImageInputURL(value string, policy imageInputLimitsWithConditions, allParams map[string]any) error {
	return validateImageInputURLWithResolver(value, policy, allParams, nil)
}

func validateImageInputURLWithResolver(value string, policy imageInputLimitsWithConditions, allParams map[string]any, resolver imageMetadataResolver) error {
	return validateImageInputURLWithPolicy(value, policy, allParams, resolver)
}

func validateImageInputURLWithPolicy(value string, policy imageInputLimitsWithConditions, allParams map[string]any, resolver imageMetadataResolver) error {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	lower := strings.ToLower(value)
	switch {
	case strings.HasPrefix(lower, "http://"), strings.HasPrefix(lower, "https://"):
		return validateRemoteImageURLWithResolver(value, policy, allParams, resolver)
	case strings.HasPrefix(lower, "vod://"),
		strings.HasPrefix(lower, "tos://"):
		if allowImageMediaURIs {
			return validateImageMetadataFromResolver(value, value, policy, allParams, resolver)
		}
		return fmt.Errorf("图片输入仅支持本地文件路径或 HTTP/HTTPS URL，不支持: %s", schemeOf(value))
	case strings.HasPrefix(lower, "mediakit://"):
		return fmt.Errorf("图片输入仅支持本地文件路径、HTTP/HTTPS URL、vod:// 或 tos://，不支持: %s", schemeOf(value))
	default:
		identity, ok, err := resolveLocalMediaIdentity(value)
		if err != nil {
			return err
		}
		if !ok {
			return nil
		}
		return validateLocalImageFileWithResolver(identity.AbsPath, identity.Size, policy, allParams, resolver)
	}
}

func schemeOf(value string) string {
	if idx := strings.Index(value, "://"); idx > 0 {
		return value[:idx+3]
	}
	return value
}

func validateLocalImageFile(path string, size int64, policy imageInputLimitsWithConditions, allParams map[string]any) error {
	return validateLocalImageFileWithResolver(path, size, policy, allParams, nil)
}

func validateLocalImageFileWithResolver(path string, size int64, policy imageInputLimitsWithConditions, allParams map[string]any, resolver imageMetadataResolver) error {
	if err := validateImageSize(size, path, policy.limits.maxBytes); err != nil {
		return err
	}
	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("无法打开图片文件: %w", err)
	}
	defer file.Close()
	return validateImageConfigWithResolver(file, path, policy, allParams, resolver)
}

func validateRemoteImageURL(rawURL string, policy imageInputLimitsWithConditions, allParams map[string]any) error {
	return validateRemoteImageURLWithResolver(rawURL, policy, allParams, nil)
}

func validateRemoteImageURLWithResolver(rawURL string, policy imageInputLimitsWithConditions, allParams map[string]any, resolver imageMetadataResolver) error {
	if _, err := netguard.ValidateRemoteURL(context.Background(), rawURL); err != nil {
		return fmt.Errorf("图片 URL 不可访问: %w", err)
	}
	return validateRemoteImageURLWithClient(
		netguard.NewHTTPClient(imagePrecheckTimeout),
		rawURL,
		policy,
		allParams,
		resolver,
	)
}

func validateRemoteImageURLWithClient(client *http.Client, rawURL string, policy imageInputLimitsWithConditions, allParams map[string]any, resolvers ...imageMetadataResolver) error {
	var resolver imageMetadataResolver
	if len(resolvers) > 0 {
		resolver = resolvers[0]
	}
	req, err := http.NewRequest(http.MethodGet, rawURL, nil)
	if err != nil {
		return fmt.Errorf("无效的图片 URL: %w", err)
	}
	req.Header.Set("Range", fmt.Sprintf("bytes=0-%d", imageHeaderProbeBytes-1))
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("图片 URL 不可访问: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("图片 URL 返回 HTTP %d: %s", resp.StatusCode, rawURL)
	}
	remoteSize := resp.ContentLength
	if total, ok := contentRangeTotal(resp.Header.Get("Content-Range")); ok {
		remoteSize = total
	}
	if err := validateImageSize(remoteSize, rawURL, policy.limits.maxBytes); err != nil {
		return err
	}
	readLimit := imageHeaderProbeBytes
	if policy.limits.maxBytes > 0 && policy.limits.maxBytes < readLimit {
		readLimit = policy.limits.maxBytes
	}
	limited := &io.LimitedReader{R: resp.Body, N: readLimit + 1}
	return validateImageConfigWithResolver(limited, rawURL, policy, allParams, resolver)
}

func contentRangeTotal(value string) (int64, bool) {
	value = strings.TrimSpace(value)
	slash := strings.LastIndex(value, "/")
	if slash < 0 || slash == len(value)-1 || value[slash+1:] == "*" {
		return 0, false
	}
	total, err := strconv.ParseInt(value[slash+1:], 10, 64)
	return total, err == nil && total >= 0
}

func validateImageSize(size int64, label string, maxBytes int64) error {
	if size > 0 && maxBytes > 0 && size > maxBytes {
		return fmt.Errorf("图片文件超过 %dMB 限制（实际 %d 字节）: %s", maxBytes/(1024*1024), size, label)
	}
	return nil
}

func validateImageConfig(r io.Reader, label string, policy imageInputLimitsWithConditions, allParams map[string]any) error {
	return validateImageConfigWithResolver(r, label, policy, allParams, nil)
}

func validateImageConfigWithResolver(r io.Reader, label string, policy imageInputLimitsWithConditions, allParams map[string]any, resolver imageMetadataResolver) error {
	cfg, decodedFormat, err := image.DecodeConfig(r)
	format := normalizeImageFormat(decodedFormat)
	if format == "" {
		format = imageFormatFromLabel(label)
	}
	if err != nil {
		if resolver != nil && !isLocallyDecodableImageFormat(format) {
			return validateImageMetadataFromResolver(label, label, policy, allParams, resolver)
		}
		if isLocalDecoderUnavailable(format) {
			if !isAllowedImageFormat(format, policy.limits.allowedFormats) {
				return fmt.Errorf("图片格式 %s 不在当前工具支持的格式范围内: %s", format, label)
			}
			// The accepted format list comes from the command's surface schema.
			// This branch only reflects the local Go decoder set; the cloud
			// service remains authoritative for accepted containers not decoded
			// by the standard library.
			return nil
		}
		return fmt.Errorf("无法解析图片文件头: %w", err)
	}
	return validateImageMetadata(
		imageMetadata{width: cfg.Width, height: cfg.Height, format: format},
		label,
		policy,
		allParams,
	)
}

func validateImageMetadataFromResolver(value string, label string, policy imageInputLimitsWithConditions, allParams map[string]any, resolver imageMetadataResolver) error {
	if resolver == nil {
		return nil
	}
	metadata, err := resolver(value)
	if err != nil {
		return err
	}
	return validateImageMetadata(metadata, label, policy, allParams)
}

func validateImageMetadata(metadata imageMetadata, label string, policy imageInputLimitsWithConditions, allParams map[string]any) error {
	format := normalizeImageFormat(metadata.format)
	if !isAllowedImageFormat(format, policy.limits.allowedFormats) {
		return fmt.Errorf("图片格式 %s 不在当前工具支持的格式范围内: %s", format, label)
	}
	limits := effectiveImageLimits(policy, allParams, format)
	width, height := metadata.width, metadata.height
	longEdge, shortEdge := width, height
	if shortEdge > longEdge {
		longEdge, shortEdge = shortEdge, longEdge
	}
	if limits.minWidth > 0 && width < limits.minWidth || limits.minHeight > 0 && height < limits.minHeight {
		return fmt.Errorf("图片分辨率低于当前工具的最小尺寸限制（实际 %dx%d）: %s", width, height, label)
	}
	if limits.maxWidth > 0 && width > limits.maxWidth || limits.maxHeight > 0 && height > limits.maxHeight {
		return fmt.Errorf("图片分辨率超过当前工具的宽高限制（实际 %dx%d）: %s", width, height, label)
	}
	if limits.minLongEdge > 0 && longEdge < limits.minLongEdge || limits.minShortEdge > 0 && shortEdge < limits.minShortEdge {
		return fmt.Errorf("图片长短边低于当前工具的最小尺寸限制（实际 %dx%d）: %s", width, height, label)
	}
	if limits.maxEdge > 0 && (width > limits.maxEdge || height > limits.maxEdge) {
		return fmt.Errorf("图片宽或高超过 %d 像素限制（实际 %dx%d）: %s", limits.maxEdge, width, height, label)
	}
	if limits.maxLongEdge > 0 && longEdge > limits.maxLongEdge {
		return fmt.Errorf("图片长边超过 %d 像素限制（实际 %dx%d）: %s", limits.maxLongEdge, width, height, label)
	}
	if limits.maxShortEdge > 0 && shortEdge > limits.maxShortEdge {
		return fmt.Errorf("图片短边超过 %d 像素限制（实际 %dx%d）: %s", limits.maxShortEdge, width, height, label)
	}
	if limits.maxPixels > 0 && int64(width)*int64(height) > limits.maxPixels {
		return fmt.Errorf("图片像素总数超过 %d 限制（实际 %d）: %s", limits.maxPixels, int64(width)*int64(height), label)
	}
	return nil
}

func newImageMetadataResolver(client *Client, resolveHome func() (string, error)) imageMetadataResolver {
	deadline := time.Now().Add(imageMetadataTimeout)
	cache := make(map[string]imageMetadata)
	return func(value string) (imageMetadata, error) {
		if metadata, ok := cache[value]; ok {
			return metadata, nil
		}
		source := value
		if !isRemoteOrMediaKitURL(source) {
			home, err := resolveHome()
			if err != nil {
				return imageMetadata{}, fmt.Errorf("获取图片元信息失败，请重试")
			}
			ctx, cancel := context.WithDeadline(context.Background(), deadline)
			source, err = materializeCloudMediaStringWithContext(ctx, client, home, "probe-image-metadata", source)
			cancel()
			if err != nil {
				return imageMetadata{}, fmt.Errorf("获取图片元信息失败，请重试")
			}
		}
		metadata, err := probeImageMetadata(deadline, client, source)
		if err != nil {
			return imageMetadata{}, err
		}
		cache[value] = metadata
		return metadata, nil
	}
}

func probeImageMetadata(deadline time.Time, client *Client, imageURL string) (imageMetadata, error) {
	ctx, cancel := context.WithDeadline(context.Background(), deadline)
	defer cancel()
	for attempt := 0; attempt < 2; attempt++ {
		response, err := client.CallWithContext(ctx, "probe-image-metadata", map[string]any{
			"image_url": imageURL,
			"info_type": "metadata",
		})
		if err == nil {
			metadata, parseErr := parseImageMetadataResponse(response)
			if parseErr == nil {
				return metadata, nil
			}
			err = parseErr
		}
		if !isRetryableImageMetadataError(err) || attempt == 1 {
			var businessErr *imageMetadataBusinessError
			if errors.As(err, &businessErr) {
				return imageMetadata{}, businessErr
			}
			return imageMetadata{}, fmt.Errorf("获取图片元信息失败，请重试")
		}
		timer := time.NewTimer(500 * time.Millisecond)
		select {
		case <-ctx.Done():
			timer.Stop()
			return imageMetadata{}, fmt.Errorf("获取图片元信息失败，请重试")
		case <-timer.C:
		}
	}
	return imageMetadata{}, fmt.Errorf("获取图片元信息失败，请重试")
}

func parseImageMetadataResponse(response map[string]any) (imageMetadata, error) {
	if success, ok := response["success"].(bool); ok && !success {
		message := strings.TrimSpace(extractErrorMessage(response))
		if message == "" {
			message = "服务未返回失败原因"
		}
		return imageMetadata{}, &imageMetadataBusinessError{message: "无法获取图片元信息：" + message}
	}
	result, _ := response["result"].(map[string]any)
	width, widthOK := integerValue(result["width"])
	height, heightOK := integerValue(result["height"])
	format, formatOK := result["format"].(string)
	if !widthOK || width <= 0 || !heightOK || height <= 0 || !formatOK || strings.TrimSpace(format) == "" {
		return imageMetadata{}, fmt.Errorf("invalid metadata response")
	}
	return imageMetadata{width: int(width), height: int(height), format: format}, nil
}

func isRetryableImageMetadataError(err error) bool {
	if err == nil || errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		return apiErr.StatusCode == http.StatusRequestTimeout ||
			apiErr.StatusCode == http.StatusTooManyRequests ||
			apiErr.StatusCode >= http.StatusInternalServerError
	}
	var networkErr net.Error
	return errors.As(err, &networkErr)
}

func effectiveImageLimits(policy imageInputLimitsWithConditions, params map[string]any, format string) imageInputLimits {
	limits := policy.limits
	for _, conditional := range policy.conditionals {
		matched := false
		if conditional.param != "" {
			if value, ok := params[conditional.param]; ok && strings.EqualFold(fmt.Sprint(value), conditional.equals) {
				matched = true
			}
		}
		if conditional.inputFormat != "" && strings.EqualFold(format, normalizeImageFormat(conditional.inputFormat)) {
			matched = true
		}
		if matched {
			mergeImageLimits(&limits, conditional.limits)
		}
	}
	return limits
}

func mergeImageLimits(dst *imageInputLimits, src imageInputLimits) {
	if src.minWidth > 0 {
		dst.minWidth = src.minWidth
	}
	if src.minHeight > 0 {
		dst.minHeight = src.minHeight
	}
	if src.maxWidth > 0 {
		dst.maxWidth = src.maxWidth
	}
	if src.maxHeight > 0 {
		dst.maxHeight = src.maxHeight
	}
	if src.maxEdge > 0 {
		dst.maxEdge = src.maxEdge
	}
	if src.minLongEdge > 0 {
		dst.minLongEdge = src.minLongEdge
	}
	if src.minShortEdge > 0 {
		dst.minShortEdge = src.minShortEdge
	}
	if src.maxLongEdge > 0 {
		dst.maxLongEdge = src.maxLongEdge
	}
	if src.maxShortEdge > 0 {
		dst.maxShortEdge = src.maxShortEdge
	}
	if src.maxPixels > 0 {
		dst.maxPixels = src.maxPixels
	}
}

func isAllowedImageFormat(format string, allowed map[string]struct{}) bool {
	if len(allowed) == 0 || format == "" {
		return true
	}
	_, ok := allowed[normalizeImageFormat(format)]
	return ok
}

func normalizeImageFormat(format string) string {
	format = strings.ToLower(strings.TrimPrefix(strings.TrimSpace(format), "."))
	switch format {
	case "jpg":
		return "jpeg"
	case "tif":
		return "tiff"
	default:
		return format
	}
}

func imageFormatFromLabel(label string) string {
	candidate := label
	if parsed, err := url.Parse(label); err == nil && parsed.Path != "" {
		candidate = parsed.Path
	}
	return normalizeImageFormat(filepath.Ext(candidate))
}

// isLocalDecoderUnavailable reports the formats for which this binary has no
// registered image decoder. It is about the local runtime, not a capability's
// accepted-format policy; accepted formats are supplied by the surface schema.
func isLocalDecoderUnavailable(format string) bool {
	switch normalizeImageFormat(format) {
	case "webp", "avif", "heic", "heif", "tiff", "bmp":
		return true
	default:
		return false
	}
}

func isLocallyDecodableImageFormat(format string) bool {
	switch normalizeImageFormat(format) {
	case "gif", "jpeg", "png":
		return true
	default:
		return false
	}
}
