package local

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/spf13/cobra"

	cliconfig "mediakit-cli/internal/config"
	"mediakit-cli/internal/local/core"
	"mediakit-cli/internal/output"
)

// Executor coordinates local capability execution in later stages.
type Executor struct{}

func Execute(cmd *cobra.Command, command string, params map[string]any) error {
	command = normalizeCommand(command)
	registration, ok := Resolve(command)
	if !ok {
		return fmt.Errorf("%s 的本地处理器未实现", command)
	}

	// 执行前检查该命令所需的依赖是否可用
	if missing := checkCommandDependencies(registration); len(missing) > 0 {
		return &DependencyError{
			Command: command,
			Missing: missing,
		}
	}

	workDir, err := os.Getwd()
	if err != nil {
		return err
	}
	home, err := cliconfig.ResolveHomeDir()
	if err != nil {
		return err
	}

	// Output path: --output-path flag > env > config > default
	// 如果 flag 值带文件扩展名，视为完整文件路径；否则视为目录
	var outputDir string
	var outputFile string
	if flagOutputPath, flagErr := cmd.Flags().GetString("output-path"); flagErr == nil && flagOutputPath != "" {
		if looksLikeFilePath(flagOutputPath) {
			outputFile = flagOutputPath
			outputDir = filepath.Dir(flagOutputPath)
		} else {
			outputDir = flagOutputPath
		}
	} else {
		resolved, _, resolveErr := cliconfig.ResolveOutputPath(home)
		if resolveErr != nil {
			return resolveErr
		}
		outputDir = resolved
	}
	if err := cliconfig.EnsureOutputDir(outputDir); err != nil {
		return err
	}
	writer, err := output.NewWriter(workDir)
	if err != nil {
		return err
	}
	tempDir, err := os.MkdirTemp("", "mediakit-cli-local-*")
	if err != nil {
		return err
	}
	stopCleanup := watchCleanupSignals(tempDir)
	defer stopCleanup()
	defer os.RemoveAll(tempDir)

	normalizedParams, _ := normalizeValueKeys(params).(map[string]any)
	if normalizedParams == nil {
		normalizedParams = map[string]any{}
	}
	if validateErr := core.ValidateParams(normalizedParams); validateErr != nil {
		return validateErr
	}

	ctx := &core.ExecContext{
		Command:    command,
		Params:     cloneParams(normalizedParams),
		WorkDir:    workDir,
		TempDir:    tempDir,
		OutputDir:  outputDir,
		OutputFile: outputFile,
		CommandIO:  cmd,
		Writer:     writer,
		Limits:     core.DefaultResourceLimits(),
	}

	result, err := registration.Handler.Execute(ctx)
	if err != nil {
		return fmt.Errorf("local 执行失败(%s/%s): %w", command, registration.Source, err)
	}
	if sanitized, ok := core.SanitizeResult(result).(map[string]any); ok {
		result = sanitized
	}
	if normalizedResult, ok := normalizeValueKeys(result).(map[string]any); ok {
		result = normalizedResult
	}

	return writeJSON(cmd.OutOrStdout(), result)
}

func writeJSON(output io.Writer, value any) error {
	encoder := json.NewEncoder(output)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")
	return encoder.Encode(value)
}

func cloneParams(params map[string]any) map[string]any {
	if len(params) == 0 {
		return map[string]any{}
	}

	cloned := make(map[string]any, len(params))
	for key, value := range params {
		cloned[key] = value
	}
	return cloned
}

func normalizeValueKeys(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		normalized := make(map[string]any, len(typed))
		for key, item := range typed {
			normalized[normalizeParamKey(key)] = normalizeValueKeys(item)
		}
		return normalized
	case []any:
		normalized := make([]any, 0, len(typed))
		for _, item := range typed {
			normalized = append(normalized, normalizeValueKeys(item))
		}
		return normalized
	default:
		return value
	}
}

func normalizeParamKey(key string) string {
	return strings.ReplaceAll(strings.TrimSpace(key), "-", "_")
}

func watchCleanupSignals(tempDir string) func() {
	signals := make(chan os.Signal, 2)
	done := make(chan struct{})
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM)

	go func() {
		select {
		case <-signals:
			_ = os.RemoveAll(tempDir)
		case <-done:
		}
	}()

	return func() {
		close(done)
		signal.Stop(signals)
		close(signals)
	}
}

// looksLikeFilePath 判断路径是否像一个文件路径（带媒体扩展名）
func looksLikeFilePath(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".mp4", ".mov", ".avi", ".mkv", ".flv", ".ts", ".wmv",
		".mp3", ".m4a", ".aac", ".wav", ".ogg", ".flac",
		".jpg", ".jpeg", ".png", ".gif", ".webp", ".bmp", ".svg":
		return true
	default:
		return false
	}
}

// DependencyError 表示本地依赖缺失错误，携带安装指引
type DependencyError struct {
	Command string
	Missing []string
}

func (e *DependencyError) Error() string {
	return fmt.Sprintf("本地依赖缺失: %s", strings.Join(e.Missing, ", "))
}

// StructuredError 返回结构化错误（含 install_guide），供 JSON 输出使用
func (e *DependencyError) StructuredError() map[string]any {
	return map[string]any{
		"error": map[string]any{
			"type":          "environment_error",
			"code":          "dependency_missing",
			"message":       fmt.Sprintf("命令 %s 所需本地依赖缺失: %s", e.Command, strings.Join(e.Missing, ", ")),
			"install_guide": cliconfig.InstallGuide(e.Missing),
		},
	}
}

// checkCommandDependencies 检查命令所需依赖是否可用
func checkCommandDependencies(reg core.Registration) []string {
	if len(reg.Dependencies) == 0 {
		return nil
	}
	home, err := cliconfig.ResolveHomeDir()
	if err != nil {
		return nil
	}
	cache, err := cliconfig.RefreshEnvCache(home)
	if err != nil {
		return nil
	}
	var missing []string
	for _, dep := range reg.Dependencies {
		status, ok := cache.Tools[dep]
		if !ok || !status.Available {
			missing = append(missing, dep)
		}
	}
	return missing
}
