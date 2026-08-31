//go:build mediakit_cloud_only

package commands

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

func configureModeFlags(cmd *cobra.Command) {
	// A Cloud-only binary has no mode selector surface.
}

func validateModeArguments(args []string) error {
	for _, arg := range args {
		if isModeArgument(arg, "local") {
			return fmt.Errorf("当前构建不支持 Local 模式；请移除 --local 后使用 Cloud 模式重试")
		}
	}
	for _, arg := range args {
		if isModeArgument(arg, "cloud") {
			return fmt.Errorf("当前构建仅支持 Cloud，无需 --cloud 参数；请移除 --cloud 后重试")
		}
	}
	return nil
}

func isModeArgument(arg string, mode string) bool {
	return arg == "--"+mode || strings.HasPrefix(arg, "--"+mode+"=")
}
