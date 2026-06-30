package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"time"

	buildinfo "mediakit-cli/internal/build"
	"mediakit-cli/internal/updatecheck"

	"github.com/spf13/cobra"
)

func newUpdateCmd() *cobra.Command {
	var checkOnly bool
	cmd := &cobra.Command{
		Use:               "update",
		Short:             "Check and apply mediakit-cli updates from the npm registry",
		Long:              "Check the npm registry for the latest @volcengine/mediakit-cli release and optionally install it via npm install -g.",
		Args:              cobra.NoArgs,
		DisableAutoGenTag: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			updatecheck.StartAsync()
			r := updatecheck.WaitForResult(3 * time.Second)
			if r == nil {
				return writeUpdatePayload(cmd, map[string]any{
					"current": buildinfo.Version,
					"action":  "skipped",
					"reason":  "update check disabled or unavailable in this environment",
				})
			}
			payload := map[string]any{
				"current":    r.Current,
				"latest":     r.Latest,
				"has_update": r.HasUpdate,
			}
			if r.Err != nil {
				payload["error"] = r.Err.Error()
				payload["action"] = "skipped"
				return writeUpdatePayload(cmd, payload)
			}
			if !r.HasUpdate {
				payload["action"] = "noop"
				return writeUpdatePayload(cmd, payload)
			}
			payload["upgrade_command"] = fmt.Sprintf("npm install -g %s@latest", updatecheck.PackageName)
			if checkOnly {
				payload["action"] = "check"
				return writeUpdatePayload(cmd, payload)
			}
			payload["action"] = "install"
			if err := runNpmInstallLatest(cmd); err != nil {
				payload["install_status"] = "failed"
				payload["error"] = err.Error()
				_ = writeUpdatePayload(cmd, payload)
				return err
			}
			payload["install_status"] = "ok"
			return writeUpdatePayload(cmd, payload)
		},
	}
	cmd.Flags().BoolVar(&checkOnly, "check", false, "Only check for updates; do not install")
	return cmd
}

func writeUpdatePayload(cmd *cobra.Command, payload map[string]any) error {
	encoder := json.NewEncoder(cmd.OutOrStdout())
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")
	return encoder.Encode(payload)
}

func runNpmInstallLatest(cmd *cobra.Command) error {
	target := fmt.Sprintf("%s@latest", updatecheck.PackageName)
	c := exec.Command("npm", "install", "-g", target)
	c.Stdout = os.Stderr
	c.Stderr = os.Stderr
	return c.Run()
}
