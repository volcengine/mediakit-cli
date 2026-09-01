//go:build !mediakit_cloud_only

package commands

import (
	"github.com/spf13/cobra"
)

func configureModeFlags(cmd *cobra.Command) {
	strictBoolVar(cmd.PersistentFlags(), &forceLocal, "local", false, "仅对当前命令生效：强制按 local-first 策略执行，不修改全局配置")
	strictBoolVar(cmd.PersistentFlags(), &forceCloud, "cloud", false, "仅对当前命令生效：强制按 cloud-first 策略执行，不修改全局配置")
}

func validateModeArguments([]string) error {
	return nil
}
