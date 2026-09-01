//go:build !mediakit_cloud_only

package commands

import (
	"bufio"
	"fmt"
	"io"
	"strings"

	cliconfig "mediakit-cli/internal/config"
)

func promptLine(reader *bufio.Reader, out io.Writer, message string) (string, error) {
	if _, err := fmt.Fprint(out, message); err != nil {
		return "", err
	}
	text, err := reader.ReadString('\n')
	if err != nil && len(text) == 0 {
		return "", err
	}
	return strings.TrimSpace(text), nil
}

func promptMode(reader *bufio.Reader, out io.Writer) (string, error) {
	if _, err := fmt.Fprintln(out, "请选择默认执行模式："); err != nil {
		return "", err
	}
	if _, err := fmt.Fprintln(out, "1. local-first  本地模式优先，本地工具异常调用 cloud 工具"); err != nil {
		return "", err
	}
	if _, err := fmt.Fprintln(out, "2. cloud-first  云端模式优先，云端不可用时再尝试本地能力"); err != nil {
		return "", err
	}
	for {
		choice, err := promptLine(reader, out, "请输入选项 [1/2]: ")
		if err != nil {
			return "", err
		}
		switch choice {
		case "1":
			return cliconfig.ModeLocalFirst, nil
		case "2":
			return cliconfig.ModeCloudFirst, nil
		}
		if _, err := fmt.Fprintln(out, "无效选项，请输入 1 或 2。"); err != nil {
			return "", err
		}
	}
}
