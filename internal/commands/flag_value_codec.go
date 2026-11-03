package commands

import (
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
)

var complexStdinConsumed bool

func resetComplexFlagStdinForTest() {
	complexStdinConsumed = false
}

func isSimpleScalarArray(schema map[string]any) bool {
	items, _ := schema["items"].(map[string]any)
	switch schemaType(items) {
	case "string", "number", "integer", "boolean":
		return true
	default:
		return false
	}
}

func simpleArrayItemType(schema map[string]any) string {
	items, _ := schema["items"].(map[string]any)
	return schemaType(items)
}

// scalarArrayFlagValue keeps CSV, repeated flags, and JSON-array encodings
// explicit instead of relying on pflag.StringSlice's pre-splitting behavior.
type scalarArrayFlagValue struct {
	values  []string
	changed bool
}

func newScalarArrayFlagValue(defaultValue []string) *scalarArrayFlagValue {
	return &scalarArrayFlagValue{values: append([]string(nil), defaultValue...)}
}

func (value *scalarArrayFlagValue) String() string {
	return strings.Join(value.values, ",")
}

func (value *scalarArrayFlagValue) Set(raw string) error {
	if !value.changed {
		value.values = nil
		value.changed = true
	}
	trimmed := strings.TrimSpace(raw)
	if strings.HasPrefix(trimmed, "[") {
		decoded, err := decodeJSONValue(trimmed)
		if err != nil {
			return err
		}
		items, ok := decoded.([]any)
		if !ok {
			return fmt.Errorf("需要 JSON 数组")
		}
		for _, item := range items {
			switch typed := item.(type) {
			case string:
				value.values = append(value.values, typed)
			case json.Number:
				value.values = append(value.values, typed.String())
			case bool:
				value.values = append(value.values, strconv.FormatBool(typed))
			default:
				return fmt.Errorf("JSON 数组只支持标量元素")
			}
		}
		return nil
	}
	value.values = append(value.values, strings.Split(raw, ",")...)
	return nil
}

func (value *scalarArrayFlagValue) Type() string { return "scalar-array" }

func simpleArrayFlagHelpSuffix() string {
	return "传参：JSON 数组、英文逗号分隔，或重复同名 flag。"
}

func complexFlagHelpSuffix() string {
	return "传参：内联 JSON，或 @文件 / - (stdin) / file://。"
}

func complexFlagEncodingHelp() string {
	return strings.Join([]string{
		"复杂参数（object / array<object>）传参方式:",
		"  1) 内联 JSON：--subtitles '[{\"subtitle_text\":\"你好\",\"start_time\":0,\"end_time\":3}]'",
		"  2) 文件：--subtitles @./subtitles.json",
		"  3) stdin：jq -nc '[...]' | mediakit-cli ... --subtitles -",
		"  4) file://：--subtitles file://./subtitles.json 或 file:///abs/path.json",
		"  说明：相对路径优先 @ 或 file://./；同一命令全局至多一个 -。",
		"基础标量数组（string/number/integer/boolean）传参方式:",
		"  --scales '[1.5,2.0,3.0]'、--scales 1.5,2.0,3.0 或重复 --scales 1.5 --scales 2.0",
	}, "\n")
}

func schemaNeedsComplexEncodingHelp(schema map[string]any) bool {
	switch schemaType(schema) {
	case "object":
		return true
	case "array":
		return !isSimpleScalarArray(schema)
	default:
		return false
	}
}

func readSimpleScalarArrayFlag(values []string, itemType string) (any, error) {
	return coerceScalarArray(stringSliceToAny(values), itemType)
}

func readComplexFlagValue(raw string) (any, error) {
	payload, err := loadJSONPayload(raw)
	if err != nil {
		return nil, err
	}
	return decodeJSONValue(payload)
}

func loadJSONPayload(raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", fmt.Errorf("不能为空")
	}
	switch {
	case trimmed == "-":
		if complexStdinConsumed {
			return "", fmt.Errorf("同一命令只能有一个复杂参数从 stdin (-) 读取")
		}
		complexStdinConsumed = true
		data, err := io.ReadAll(os.Stdin)
		if err != nil {
			return "", fmt.Errorf("读取 stdin 失败: %w", err)
		}
		text := strings.TrimSpace(string(data))
		if text == "" {
			return "", fmt.Errorf("stdin 不能为空")
		}
		return text, nil
	case strings.HasPrefix(trimmed, "@"):
		path := strings.TrimSpace(strings.TrimPrefix(trimmed, "@"))
		if path == "" {
			return "", fmt.Errorf("@ 后需要文件路径")
		}
		return readJSONFile(path)
	case strings.HasPrefix(strings.ToLower(trimmed), "file://"):
		path, err := fileURLPath(trimmed)
		if err != nil {
			return "", err
		}
		return readJSONFile(path)
	default:
		return trimmed, nil
	}
}

func fileURLPath(raw string) (string, error) {
	parsed, err := url.Parse(raw)
	if err != nil {
		return "", fmt.Errorf("无效的 file:// URL: %w", err)
	}
	if !strings.EqualFold(parsed.Scheme, "file") {
		return "", fmt.Errorf("无效的 file:// URL")
	}

	// file://./rel.json → Host="." Path="/rel.json"
	if parsed.Host == "." || parsed.Host == ".." {
		return filepath.Clean(parsed.Host + parsed.Path), nil
	}
	// file:///abs/path or file://localhost/abs/path
	if parsed.Host == "" || strings.EqualFold(parsed.Host, "localhost") {
		path := parsed.Path
		if parsed.Opaque != "" {
			path = parsed.Opaque
		}
		if path == "" {
			return "", fmt.Errorf("file:// 缺少路径")
		}
		return fileURLPathForPlatform(path), nil
	}
	// Windows also commonly emits file://C:/path, where the drive is parsed
	// as the URL authority. Unix treats that form as an unsupported host.
	if isWindowsDriveHost(parsed.Host) {
		if runtime.GOOS != "windows" {
			return "", fmt.Errorf("Windows 盘符 file:// URL 仅支持 Windows: %q", raw)
		}
		return fileURLPathForPlatform(parsed.Host + parsed.Path), nil
	}
	if runtime.GOOS == "windows" && parsed.Path != "" {
		return fileURLPathForPlatform("//" + parsed.Host + parsed.Path), nil
	}
	// file://rel/path without leading ./ — host is first segment.
	if parsed.Path == "" && parsed.Opaque != "" {
		return parsed.Opaque, nil
	}
	if parsed.Path != "" {
		return "", fmt.Errorf(
			"file:// 含 host %q 时仅支持 localhost；相对路径请用 file://./... 或 @路径",
			parsed.Host,
		)
	}
	return "", fmt.Errorf("file:// 缺少路径")
}

func isWindowsDriveHost(host string) bool {
	return len(host) == 2 && ((host[0] >= 'A' && host[0] <= 'Z') || (host[0] >= 'a' && host[0] <= 'z')) && host[1] == ':'
}

func fileURLPathForPlatform(path string) string {
	if runtime.GOOS == "windows" {
		if len(path) >= 3 && path[0] == '/' && isWindowsDriveHost(path[1:3]) {
			path = path[1:]
		}
		return filepath.FromSlash(path)
	}
	return filepath.FromSlash(path)
}

func readJSONFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("读取 JSON 文件失败: %w", err)
	}
	text := strings.TrimSpace(string(data))
	if text == "" {
		return "", fmt.Errorf("JSON 文件不能为空: %s", path)
	}
	return text, nil
}

func decodeJSONValue(raw string) (any, error) {
	decoder := json.NewDecoder(strings.NewReader(raw))
	decoder.UseNumber()
	var value any
	if err := decoder.Decode(&value); err != nil {
		return nil, fmt.Errorf("需要合法 JSON: %w", err)
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			return nil, fmt.Errorf("JSON 后不能有多余内容")
		}
		return nil, fmt.Errorf("JSON 后不能有多余内容: %w", err)
	}
	return value, nil
}

func stringSliceToAny(values []string) []any {
	out := make([]any, len(values))
	for i, value := range values {
		out[i] = value
	}
	return out
}

func coerceScalarArray(values []any, itemType string) ([]any, error) {
	out := make([]any, 0, len(values))
	for index, value := range values {
		coerced, err := coerceScalarItem(value, itemType)
		if err != nil {
			return nil, fmt.Errorf("第 %d 项无效: %w", index+1, err)
		}
		out = append(out, coerced)
	}
	return out, nil
}

func coerceScalarItem(value any, itemType string) (any, error) {
	switch itemType {
	case "string":
		switch typed := value.(type) {
		case string:
			return typed, nil
		case json.Number:
			return typed.String(), nil
		case float64:
			return strconv.FormatFloat(typed, 'f', -1, 64), nil
		case bool:
			return strconv.FormatBool(typed), nil
		default:
			return fmt.Sprint(value), nil
		}
	case "number":
		return coerceNumber(value)
	case "integer":
		number, err := coerceNumber(value)
		if err != nil {
			return nil, err
		}
		asFloat, ok := number.(float64)
		if !ok {
			return nil, fmt.Errorf("需要整数")
		}
		if asFloat != float64(int64(asFloat)) {
			return nil, fmt.Errorf("需要整数")
		}
		return int64(asFloat), nil
	case "boolean":
		switch typed := value.(type) {
		case bool:
			return typed, nil
		case string:
			switch strings.ToLower(strings.TrimSpace(typed)) {
			case "true":
				return true, nil
			case "false":
				return false, nil
			default:
				return nil, fmt.Errorf("需要 true/false")
			}
		default:
			return nil, fmt.Errorf("需要 true/false")
		}
	default:
		return nil, fmt.Errorf("不支持的数组元素类型: %s", itemType)
	}
}

func coerceNumber(value any) (any, error) {
	switch typed := value.(type) {
	case float64:
		return typed, nil
	case json.Number:
		number, err := typed.Float64()
		if err != nil {
			return nil, fmt.Errorf("需要数字: %w", err)
		}
		return number, nil
	case int:
		return float64(typed), nil
	case int64:
		return float64(typed), nil
	case string:
		number, err := strconv.ParseFloat(strings.TrimSpace(typed), 64)
		if err != nil {
			return nil, fmt.Errorf("需要数字: %w", err)
		}
		return number, nil
	default:
		return nil, fmt.Errorf("需要数字")
	}
}
