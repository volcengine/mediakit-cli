//go:build !mediakit_cloud_only

package core

import (
	"fmt"
	"strconv"
	"strings"
)

var metadataFields = map[string]struct{}{
	"title":   {},
	"artist":  {},
	"comment": {},
}

// FFmpegPolicy validates flags before invoking ffmpeg.
type FFmpegPolicy struct {
	AllowedFlags        map[string]struct{}
	AllowedFlagPrefixes []string
}

func DefaultFFmpegPolicy() FFmpegPolicy {
	return FFmpegPolicy{
		AllowedFlags: map[string]struct{}{
			"-hide_banner": {},
			"-loglevel":    {},
			"-y":           {},
			"-n":           {},
			"-i":           {},
			"-ss":          {},
			"-to":          {},
			"-t":           {},
			"-map":         {},
			"-vf":          {},
			"-af":          {},
			"-f":           {},
			"-safe":        {},
			"-shortest":    {},
			"-vn":          {},
			"-an":          {},
			"-sn":          {},
			"-r":           {},
			"-ar":          {},
			"-ac":          {},
			"-pix_fmt":     {},
			"-profile:v":   {},
			"-movflags":    {},
			"-preset":      {},
			"-crf":         {},
			"-threads":     {},
		},
		AllowedFlagPrefixes: []string{
			"-c",
			"-codec",
			"-b:",
			"-filter",
			"-metadata",
		},
	}
}

func (p FFmpegPolicy) ValidateArgs(args []string) error {
	for _, arg := range args {
		if err := ValidateSafeText(arg, "ffmpeg 参数"); err != nil {
			return err
		}
		if IsRemoteURL(arg) {
			return fmt.Errorf("禁止将远程 URL 直接传给 ffmpeg")
		}
		if !strings.HasPrefix(arg, "-") {
			continue
		}
		if _, ok := p.AllowedFlags[arg]; ok {
			continue
		}
		allowed := false
		for _, prefix := range p.AllowedFlagPrefixes {
			if strings.HasPrefix(arg, prefix) {
				allowed = true
				break
			}
		}
		if !allowed {
			return fmt.Errorf("ffmpeg 参数不在白名单内: %s", arg)
		}
	}
	return nil
}

func ValidateSafeText(value string, field string) error {
	if containsUnsafeRune(value) {
		if strings.TrimSpace(field) == "" {
			field = "文本"
		}
		return fmt.Errorf("%s 包含不安全字符", field)
	}
	return nil
}

func SanitizeText(value string) string {
	var builder strings.Builder
	for _, r := range value {
		if isUnsafeRune(r) {
			continue
		}
		builder.WriteRune(r)
	}
	return strings.TrimSpace(builder.String())
}

func ValidateParams(value any) error {
	switch typed := value.(type) {
	case string:
		return ValidateSafeText(typed, "参数")
	case []string:
		for _, item := range typed {
			if err := ValidateParams(item); err != nil {
				return err
			}
		}
	case []any:
		for _, item := range typed {
			if err := ValidateParams(item); err != nil {
				return err
			}
		}
	case map[string]any:
		for key, item := range typed {
			if err := ValidateSafeText(key, "参数名"); err != nil {
				return err
			}
			if err := ValidateParams(item); err != nil {
				return err
			}
		}
	}
	return nil
}

func SanitizeResult(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		sanitized := make(map[string]any, len(typed))
		for key, item := range typed {
			lowerKey := strings.ToLower(strings.TrimSpace(key))
			if _, ok := metadataFields[lowerKey]; ok {
				if text, textOK := item.(string); textOK {
					sanitized[key] = SanitizeText(text)
					continue
				}
			}
			sanitized[key] = SanitizeResult(item)
		}
		return sanitized
	case []any:
		sanitized := make([]any, 0, len(typed))
		for _, item := range typed {
			sanitized = append(sanitized, SanitizeResult(item))
		}
		return sanitized
	default:
		return value
	}
}

func ResolveOutputPath(ctx *ExecContext, rawPath string) (string, error) {
	if err := ValidateSafeText(rawPath, "输出路径"); err != nil {
		return "", err
	}
	return ctx.Writer.ResolvePath(rawPath)
}

func WriteOutputFile(ctx *ExecContext, rawPath string, data []byte, overwrite bool) (string, error) {
	if err := ValidateSafeText(rawPath, "输出路径"); err != nil {
		return "", err
	}
	return ctx.Writer.WriteFileAtomic(rawPath, data, 0o644, overwrite)
}

func ParamBool(params map[string]any, key string, defaultValue bool) (bool, error) {
	value, ok := params[key]
	if !ok {
		return defaultValue, nil
	}

	switch typed := value.(type) {
	case bool:
		return typed, nil
	case string:
		parsed, err := strconv.ParseBool(strings.TrimSpace(typed))
		if err != nil {
			return false, fmt.Errorf("%s 必须是布尔值", key)
		}
		return parsed, nil
	default:
		return false, fmt.Errorf("%s 必须是布尔值", key)
	}
}

func containsUnsafeRune(value string) bool {
	for _, r := range value {
		if isUnsafeRune(r) {
			return true
		}
	}
	return false
}

func isUnsafeRune(r rune) bool {
	switch {
	case r >= 0x00 && r <= 0x1F:
		return true
	case r == 0x7F:
		return true
	case r >= 0x200B && r <= 0x200F:
		return true
	case r >= 0x202A && r <= 0x202E:
		return true
	case r == 0x2028 || r == 0x2029:
		return true
	case r >= 0x2066 && r <= 0x2069:
		return true
	case r == 0xFEFF:
		return true
	default:
		return false
	}
}
