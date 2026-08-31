//go:build !mediakit_cloud_only

package core

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"time"
)

// MaterializeInput converts a local path or remote URL into a local file path.
func MaterializeInput(ctx *ExecContext, value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", fmt.Errorf("input is required")
	}
	if err := ValidateSafeText(value, "输入路径"); err != nil {
		return "", err
	}

	if IsRemoteURL(value) {
		return downloadRemoteFile(ctx.TempDir, value)
	}

	if filepath.IsAbs(value) {
		return filepath.Clean(value), nil
	}

	return filepath.Join(ctx.WorkDir, value), nil
}

// RunBinary executes a local process with argument list semantics only.
func RunBinary(binary string, args ...string) ([]byte, error) {
	if err := ValidateSafeText(binary, "可执行文件"); err != nil {
		return nil, err
	}
	for _, arg := range args {
		if err := ValidateSafeText(arg, "命令参数"); err != nil {
			return nil, err
		}
	}

	cmd := exec.Command(binary, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		text := strings.TrimSpace(string(output))
		if text == "" {
			return nil, err
		}
		return output, fmt.Errorf("%w: %s", err, text)
	}
	return output, nil
}

// RunFFmpeg executes ffmpeg via os/exec without going through a shell.
func RunFFmpeg(args ...string) ([]byte, error) {
	if err := DefaultFFmpegPolicy().ValidateArgs(args); err != nil {
		return nil, err
	}
	return RunBinary("ffmpeg", args...)
}

// RunFFprobe executes ffprobe via os/exec without going through a shell.
func RunFFprobe(args ...string) ([]byte, error) {
	for _, arg := range args {
		if err := ValidateSafeText(arg, "ffprobe 参数"); err != nil {
			return nil, err
		}
		if IsRemoteURL(arg) {
			return nil, fmt.Errorf("禁止将远程 URL 直接传给 ffprobe")
		}
	}
	return RunBinary("ffprobe", args...)
}

func IsRemoteURL(value string) bool {
	parsed, err := url.Parse(value)
	if err != nil {
		return false
	}
	switch strings.ToLower(parsed.Scheme) {
	case "http", "https":
		return parsed.Host != ""
	default:
		return false
	}
}

func ResolveLocalPath(ctx *ExecContext, value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", fmt.Errorf("本地路径不能为空")
	}
	if err := ValidateSafeText(value, "本地路径"); err != nil {
		return "", err
	}
	if filepath.IsAbs(value) {
		return filepath.Clean(value), nil
	}
	return filepath.Join(ctx.WorkDir, value), nil
}

func FetchRemoteFile(outputDir string, source string) (string, error) {
	if !IsRemoteURL(source) {
		return "", fmt.Errorf("fetch-file 仅下载 http/https URL")
	}
	return downloadRemoteFile(outputDir, source)
}

func downloadRemoteFile(tempDir string, source string) (string, error) {
	if strings.TrimSpace(tempDir) == "" {
		return "", fmt.Errorf("temp dir is required for remote inputs")
	}
	if err := ValidateSafeText(source, "远程输入"); err != nil {
		return "", err
	}

	resp, err := (&http.Client{Timeout: 2 * time.Minute}).Get(source)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("download failed: HTTP %d", resp.StatusCode)
	}

	fileName := remoteFileName(source)
	targetPath := uniqueTargetPath(tempDir, fileName)
	file, err := os.OpenFile(targetPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
	if err != nil {
		return "", err
	}
	defer file.Close()

	if _, err := io.Copy(file, resp.Body); err != nil {
		return "", err
	}
	return targetPath, nil
}

func uniqueTargetPath(dir string, fileName string) string {
	targetPath := filepath.Join(dir, fileName)
	if _, err := os.Stat(targetPath); os.IsNotExist(err) {
		return targetPath
	}
	ext := filepath.Ext(fileName)
	base := strings.TrimSuffix(fileName, ext)
	if strings.TrimSpace(base) == "" {
		base = "remote-input"
	}
	return filepath.Join(dir, fmt.Sprintf("%s-%d%s", base, time.Now().UnixNano(), ext))
}

func remoteFileName(source string) string {
	parsed, err := url.Parse(source)
	if err != nil {
		return "remote-input.bin"
	}

	name := path.Base(parsed.Path)
	name = strings.TrimSpace(name)
	if name == "" || name == "." || name == "/" {
		return "remote-input.bin"
	}
	return name
}
