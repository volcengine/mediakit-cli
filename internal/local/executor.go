package local

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/signal"
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

	workDir, err := os.Getwd()
	if err != nil {
		return err
	}
	home, err := cliconfig.ResolveHomeDir()
	if err != nil {
		return err
	}
	outputDir, _, err := cliconfig.ResolveOutputPath(home)
	if err != nil {
		return err
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
		Command:   command,
		Params:    cloneParams(normalizedParams),
		WorkDir:   workDir,
		TempDir:   tempDir,
		OutputDir: outputDir,
		CommandIO: cmd,
		Writer:    writer,
		Limits:    core.DefaultResourceLimits(),
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
