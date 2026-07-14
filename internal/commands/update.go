package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	buildinfo "mediakit-cli/internal/build"
	cliconfig "mediakit-cli/internal/config"
	"mediakit-cli/internal/skillstate"
	"mediakit-cli/internal/updatecheck"

	"github.com/spf13/cobra"
)

var (
	checkNowFresh = func() *updatecheck.Result {
		return updatecheck.CheckNow(3*time.Second, true)
	}
	runNpmInstall              = runNpmInstallLatest
	runSkillsInstall           = runSkillsInstallFromPackage
	runSkillsInstallForVersion = runSkillsInstallFromPackageVersion
)

func newUpdateCmd() *cobra.Command {
	var checkOnly bool
	var force bool
	var asJSON bool
	cmd := &cobra.Command{
		Use:               "update",
		Short:             "Update mediakit-cli",
		Long:              "Update @volcengine/mediakit-cli from npm.",
		Args:              cobra.NoArgs,
		DisableAutoGenTag: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			r := checkNowFresh()
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
			payload["upgrade_command"] = "mediakit-cli update"
			if checkOnly {
				payload["action"] = "check"
				applySkillsStatus(payload, r)
				if !asJSON {
					return writeUpdateCheckText(cmd, r)
				}
				return writeUpdatePayload(cmd, payload)
			}

			if r.HasUpdate {
				payload["action"] = "install"
				if !asJSON {
					fmt.Fprintf(cmd.OutOrStdout(), "Updating mediakit-cli %s → %s via npm ...\n", r.Current, r.Latest)
				}
				if err := runNpmInstall(cmd); err != nil {
					payload["install_status"] = "failed"
					payload["error"] = err.Error()
					if asJSON {
						_ = writeUpdatePayload(cmd, payload)
					}
					return err
				}
				payload["install_status"] = "ok"
				if err := runSkillsInstallForVersion(cmd, r.Latest); err != nil {
					payload["skills_action"] = "failed"
					payload["skills_error"] = err.Error()
					if asJSON {
						return writeUpdatePayload(cmd, payload)
					}
					fmt.Fprintf(cmd.ErrOrStderr(), "Warning: skills update failed: %v\nRun: mediakit-cli update --force\n", err)
					return nil
				}
				payload["skills_action"] = "installed"
			} else {
				payload["action"] = "noop"
			}

			shouldInstallSkills := !r.HasUpdate && shouldInstallSkillsForVersion(r.Current, force)
			if shouldInstallSkills {
				if err := runSkillsInstall(cmd); err != nil {
					payload["skills_action"] = "failed"
					payload["skills_error"] = err.Error()
					if asJSON {
						_ = writeUpdatePayload(cmd, payload)
					}
					return err
				}
				payload["skills_action"] = "installed"
			}
			if !asJSON {
				return writeUpdateText(cmd, r, shouldInstallSkills)
			}
			return writeUpdatePayload(cmd, payload)
		},
	}
	cmd.Flags().BoolVar(&checkOnly, "check", false, "Only check for updates; do not install")
	cmd.Flags().BoolVar(&force, "force", false, "Force reinstall skills from the current npm package even when CLI is already up to date")
	cmd.Flags().BoolVar(&asJSON, "json", false, "Output structured JSON")
	return cmd
}

func writeUpdatePayload(cmd *cobra.Command, payload map[string]any) error {
	encoder := json.NewEncoder(cmd.OutOrStdout())
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")
	return encoder.Encode(payload)
}

func writeUpdateCheckText(cmd *cobra.Command, r *updatecheck.Result) error {
	if r.HasUpdate {
		_, err := fmt.Fprintf(cmd.OutOrStdout(),
			"Update available: %s → %s\n  Release:   https://www.npmjs.com/package/%s/v/%s\n\nRun `mediakit-cli update` to install.\n",
			r.Current, r.Latest, updatecheck.PackageName, r.Latest)
		return err
	}
	_, err := fmt.Fprintf(cmd.OutOrStdout(), "mediakit-cli %s is already up to date\n", r.Current)
	return err
}

func writeUpdateText(cmd *cobra.Command, r *updatecheck.Result, skillsInstalled bool) error {
	if r.HasUpdate {
		fmt.Fprintf(cmd.OutOrStdout(), "\n✓ Successfully updated mediakit-cli from %s to %s\n\n", r.Current, r.Latest)
		fmt.Fprintln(cmd.OutOrStdout(), "Updating skills ...")
		fmt.Fprintln(cmd.OutOrStdout(), "✓ Skills updated")
	} else {
		fmt.Fprintf(cmd.OutOrStdout(), "mediakit-cli %s is already up to date\n", r.Current)
	}
	if skillsInstalled {
		fmt.Fprintln(cmd.OutOrStdout(), "\nUpdating skills ...")
		_, err := fmt.Fprintln(cmd.OutOrStdout(), "✓ Skills updated")
		return err
	}
	return nil
}

func runNpmInstallLatest(cmd *cobra.Command) error {
	target := fmt.Sprintf("%s@latest", updatecheck.PackageName)
	c := exec.Command("npm", "install", "-g", target)
	c.Stdout = os.Stderr
	c.Stderr = os.Stderr
	return c.Run()
}

func runSkillsInstallFromPackage(cmd *cobra.Command) error {
	return runSkillsInstallFromPackageVersion(cmd, buildinfo.Version)
}

func runSkillsInstallFromPackageVersion(cmd *cobra.Command, version string) error {
	c := exec.Command("npx", "-y", packageSpecForVersion(version), "install", "--skills-only", "-y")
	c.Stdout = os.Stderr
	c.Stderr = os.Stderr
	return c.Run()
}

func currentPackageSpec() string {
	return packageSpecForVersion(buildinfo.Version)
}

func packageSpecForVersion(version string) string {
	version = normalizeVersion(version)
	if version == "" {
		return updatecheck.PackageName
	}
	return fmt.Sprintf("%s@%s", updatecheck.PackageName, version)
}

func shouldInstallSkillsForVersion(version string, force bool) bool {
	if force {
		return true
	}
	home, err := cliconfig.ResolveHomeDir()
	if err != nil {
		return false
	}
	return !skillstate.InSync(home, version)
}

func applySkillsStatus(payload map[string]any, r *updatecheck.Result) {
	home, err := cliconfig.ResolveHomeDir()
	if err != nil {
		return
	}
	target := skillsTargetVersion(r)
	command := "mediakit-cli update --force"
	if r.HasUpdate {
		command = "mediakit-cli update"
	}
	status, err := skillstate.ReadStatus(home, target)
	if err != nil || status == nil {
		return
	}
	status.Command = command
	payload["skills_status"] = map[string]any{
		"current": status.Current,
		"target":  status.Target,
		"in_sync": status.InSync,
		"missing": status.Missing,
		"command": status.Command,
	}
}

func skillsTargetVersion(r *updatecheck.Result) string {
	if r == nil {
		return ""
	}
	if r.HasUpdate && strings.TrimSpace(r.Latest) != "" {
		return r.Latest
	}
	return r.Current
}

func normalizeVersion(version string) string {
	version = strings.TrimSpace(version)
	version = strings.TrimPrefix(version, "v")
	return strings.TrimPrefix(version, "V")
}

// newUpdateRefreshCmd is the hidden entry the detached refresh subprocess runs.
// It fetches the latest version and writes the cache, with no output. The empty
// persistent hooks override the root's update-notice hooks so the child never
// re-spawns itself or prints a nag.
func newUpdateRefreshCmd() *cobra.Command {
	return &cobra.Command{
		Use:               "__update-refresh",
		Hidden:            true,
		Args:              cobra.NoArgs,
		DisableAutoGenTag: true,
		PersistentPreRun:  func(cmd *cobra.Command, args []string) {},
		PersistentPostRun: func(cmd *cobra.Command, args []string) {},
		RunE: func(cmd *cobra.Command, args []string) error {
			updatecheck.RunRefresh()
			return nil
		},
	}
}
