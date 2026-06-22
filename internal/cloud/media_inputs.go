package cloud

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	cliconfig "mediakit-cli/internal/config"
)

const mediaUploadCommand = "request-media-upload-url"

var mediaInputNames = map[string]bool{
	"video_url":     true,
	"video_urls":    true,
	"audio_url":     true,
	"audio_urls":    true,
	"image_url":     true,
	"image_urls":    true,
	"subtitle_url":  true,
	"subtitle_urls": true,
	"sub_image_url": true,
}

func materializeCloudMediaInputs(client *Client, command string, params map[string]any) (map[string]any, error) {
	if command == queryTaskCommand || command == mediaUploadCommand || len(params) == 0 {
		return params, nil
	}
	home, err := cliconfig.ResolveHomeDir()
	if err != nil {
		return nil, err
	}
	materialized, err := materializeCloudValue(client, home, command, "", params, false)
	if err != nil {
		return nil, err
	}
	next, ok := materialized.(map[string]any)
	if !ok {
		return params, nil
	}
	return next, nil
}

func materializeCloudValue(client *Client, home string, command string, key string, value any, mediaContext bool) (any, error) {
	switch typed := value.(type) {
	case map[string]any:
		next := make(map[string]any, len(typed))
		for childKey, childValue := range typed {
			childMediaContext := mediaContext || isMediaInputField(childKey)
			materialized, err := materializeCloudValue(client, home, command, childKey, childValue, childMediaContext)
			if err != nil {
				return nil, err
			}
			next[childKey] = materialized
		}
		return next, nil
	case []any:
		next := make([]any, len(typed))
		childMediaContext := mediaContext || isMediaInputField(key)
		for i, childValue := range typed {
			materialized, err := materializeCloudValue(client, home, command, key, childValue, childMediaContext)
			if err != nil {
				return nil, err
			}
			next[i] = materialized
		}
		return next, nil
	case []string:
		if !mediaContext && !isMediaInputField(key) {
			return typed, nil
		}
		next := make([]string, len(typed))
		for i, childValue := range typed {
			materialized, err := materializeCloudMediaString(client, home, command, childValue)
			if err != nil {
				return nil, err
			}
			next[i] = materialized
		}
		return next, nil
	case string:
		if !mediaContext && !isMediaInputField(key) {
			return typed, nil
		}
		return materializeCloudMediaString(client, home, command, typed)
	default:
		return value, nil
	}
}

func materializeCloudMediaString(client *Client, home string, command string, value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" || isRemoteOrMediaKitURL(value) {
		return value, nil
	}

	identity, ok, err := resolveLocalMediaIdentity(value)
	if err != nil {
		return "", err
	}
	if !ok {
		return value, nil
	}

	now := time.Now().UTC()
	if fileID, err := lookupUploadCache(home, identity, now); err != nil {
		return "", err
	} else if fileID != "" {
		return fileID, nil
	}

	fileID, err := client.uploadLocalMediaFile(command, identity.AbsPath)
	if err != nil {
		return "", err
	}
	return storeUploadCache(home, identity, fileID, now)
}

func resolveLocalMediaIdentity(value string) (fileIdentity, bool, error) {
	path := expandUserPath(value)
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) && looksLikeLocalPath(value) {
			return fileIdentity{}, false, fmt.Errorf("本地媒体文件不存在: %s", value)
		}
		return fileIdentity{}, false, nil
	}
	if info.IsDir() {
		return fileIdentity{}, false, fmt.Errorf("本地媒体输入不能是目录: %s", value)
	}
	absPath, err := filepath.Abs(path)
	if err != nil {
		return fileIdentity{}, false, err
	}
	return fileIdentity{
		AbsPath:       absPath,
		Size:          info.Size(),
		MTimeUnixNano: info.ModTime().UnixNano(),
	}, true, nil
}

func isMediaInputField(name string) bool {
	normalized := strings.ToLower(strings.ReplaceAll(strings.TrimSpace(name), "-", "_"))
	if mediaInputNames[normalized] {
		return true
	}
	if strings.HasSuffix(normalized, "_url") || strings.HasSuffix(normalized, "_urls") {
		return strings.Contains(normalized, "video") ||
			strings.Contains(normalized, "audio") ||
			strings.Contains(normalized, "image") ||
			strings.Contains(normalized, "subtitle")
	}
	return false
}

func isRemoteOrMediaKitURL(value string) bool {
	lower := strings.ToLower(value)
	return strings.HasPrefix(lower, "http://") ||
		strings.HasPrefix(lower, "https://") ||
		strings.HasPrefix(lower, "mediakit://")
}

func looksLikeLocalPath(value string) bool {
	if filepath.IsAbs(value) {
		return true
	}
	if strings.HasPrefix(value, "./") || strings.HasPrefix(value, "../") ||
		strings.HasPrefix(value, "~/") || strings.HasPrefix(value, ".\\") ||
		strings.HasPrefix(value, "..\\") || strings.HasPrefix(value, "~\\") {
		return true
	}
	if strings.Contains(value, "/") || strings.Contains(value, "\\") {
		return true
	}
	switch strings.ToLower(filepath.Ext(value)) {
	case ".mp4", ".mov", ".m4v", ".avi", ".mkv", ".webm", ".mp3", ".m4a", ".wav", ".aac", ".flac", ".jpg", ".jpeg", ".png", ".webp", ".gif", ".srt", ".ass", ".vtt":
		return true
	default:
		return false
	}
}

func expandUserPath(value string) string {
	if value == "~" {
		if home, err := os.UserHomeDir(); err == nil {
			return home
		}
	}
	if strings.HasPrefix(value, "~/") || strings.HasPrefix(value, "~\\") {
		if home, err := os.UserHomeDir(); err == nil {
			return filepath.Join(home, value[2:])
		}
	}
	return value
}
