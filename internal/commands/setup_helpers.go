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

func promptRequired(reader *bufio.Reader, out io.Writer, message string) (string, error) {
	for {
		value, err := promptLine(reader, out, message)
		if err != nil {
			return "", err
		}
		if value != "" {
			return value, nil
		}
		if _, err := fmt.Fprintln(out, "输入不能为空，请重试。"); err != nil {
			return "", err
		}
	}
}

func promptYesNo(reader *bufio.Reader, out io.Writer, message string, defaultYes bool) (bool, error) {
	suffix := " [y/N]: "
	if defaultYes {
		suffix = " [Y/n]: "
	}
	for {
		value, err := promptLine(reader, out, message+suffix)
		if err != nil {
			return false, err
		}
		value = strings.ToLower(strings.TrimSpace(value))
		if value == "" {
			return defaultYes, nil
		}
		switch value {
		case "y", "yes":
			return true, nil
		case "n", "no":
			return false, nil
		}
		if _, err := fmt.Fprintln(out, "请输入 y 或 n。"); err != nil {
			return false, err
		}
	}
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

func promptCredentialStore(reader *bufio.Reader, out io.Writer) (string, error) {
	if _, err := fmt.Fprintln(out, "请选择 API Key 保存方式："); err != nil {
		return "", err
	}
	if _, err := fmt.Fprintln(out, "1. 写入 ~/.mediakit/config.json"); err != nil {
		return "", err
	}
	if _, err := fmt.Fprintln(out, "2. 追加到 shell 配置文件（~/.zshrc / ~/.bashrc）"); err != nil {
		return "", err
	}
	if _, err := fmt.Fprintln(out, "3. 仅输出 export 命令，由我手动执行"); err != nil {
		return "", err
	}
	for {
		choice, err := promptLine(reader, out, "请输入选项 [1/2/3]: ")
		if err != nil {
			return "", err
		}
		switch choice {
		case "1":
			return "config", nil
		case "2":
			return "shell", nil
		case "3":
			return "export", nil
		}
		if _, err := fmt.Fprintln(out, "无效选项，请输入 1、2 或 3。"); err != nil {
			return "", err
		}
	}
}

func formatExportLines(apiKey, endpoint, surface, runtime string) string {
	lines := []string{fmt.Sprintf("export %s=%q", cliconfig.EnvAPIKey, apiKey)}
	if endpoint != "" {
		lines = append(lines, fmt.Sprintf("export %s=%q", cliconfig.EnvEndpoint, endpoint))
	}
	if surface != "" {
		lines = append(lines, fmt.Sprintf("export %s=%q", cliconfig.EnvSurface, surface))
	}
	if runtime != "" {
		lines = append(lines, fmt.Sprintf("export %s=%q", cliconfig.EnvRuntime, runtime))
	}
	return strings.Join(lines, "\n")
}
