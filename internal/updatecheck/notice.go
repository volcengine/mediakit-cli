//go:build !mediakit_no_update

package updatecheck

import (
	"fmt"
	"io"
	"os"
	"strings"

	"mediakit-cli/internal/build"
	cliconfig "mediakit-cli/internal/config"
	"mediakit-cli/internal/skillstate"
)

// NoticePayload returns a map suitable to be embedded into the JSON stdout
// payload at the top-level `_notice.update` key, or nil if no update is needed.
func NoticePayload() map[string]any {
	if currentCacheChecking() {
		return nil
	}
	r := GetResult()
	if r == nil || !r.HasUpdate {
		return nil
	}
	return map[string]any{
		"current": r.Current,
		"latest":  r.Latest,
		"command": "mediakit-cli update",
		"message": fmt.Sprintf("New %s release available: %s -> %s, run mediakit-cli update to update CLI and skills", PackageName, r.Current, r.Latest),
	}
}

// InjectNotice adds `_notice.update` to a top-level JSON object map if an
// update is available. It is a no-op when the result is nil or no update.
func InjectNotice(result map[string]any) {
	if result == nil {
		return
	}
	notice, _ := result["_notice"].(map[string]any)
	if notice == nil {
		notice = map[string]any{}
	}
	if payload := NoticePayload(); payload != nil {
		notice["update"] = payload
	}
	if payload := SkillsNoticePayload(); payload != nil {
		notice["skills"] = payload
	}
	if len(notice) == 0 {
		return
	}
	result["_notice"] = notice
}

func SkillsNoticePayload() map[string]any {
	if currentCacheChecking() {
		return nil
	}
	r := GetResult()
	if r == nil || r.HasUpdate || strings.TrimSpace(r.Latest) == "" {
		return nil
	}
	home, err := cliconfig.ResolveHomeDir()
	if err != nil {
		return nil
	}
	status, err := skillstate.ReadStatus(home, r.Current)
	if err != nil || status == nil || status.InSync {
		return nil
	}
	status.Command = "mediakit-cli update --force"
	return map[string]any{
		"current": status.Current,
		"target":  status.Target,
		"command": status.Command,
		"message": fmt.Sprintf("MediaKit skills are not synced with current CLI: current %s, target %s, run: %s",
			displayVersion(status.Current), status.Target, status.Command),
	}
}

func currentCacheChecking() bool {
	home, err := cliconfig.ResolveHomeDir()
	if err != nil {
		return false
	}
	cached, err := LoadCache(home)
	if err != nil || !IsCacheFresh(cached, build.Version) {
		return false
	}
	return effectiveStatus(cached) == CacheStatusChecking
}

func displayVersion(version string) string {
	if version == "" {
		return "missing"
	}
	return version
}

// PrintStderrNag prints an unobtrusive update hint to stderr when an update is
// available and stderr is a TTY (character device).
func PrintStderrNag(w io.Writer) {
	r := GetResult()
	if r == nil || !r.HasUpdate {
		return
	}
	f, ok := w.(*os.File)
	if !ok {
		return
	}
	if !isCharDevice(f) {
		return
	}
	fmt.Fprintf(w, "\n[mediakit-cli] new version available: %s -> %s\n  run: mediakit-cli update\n",
		r.Current, r.Latest)
}

func isCharDevice(f *os.File) bool {
	if f == nil {
		return false
	}
	info, err := f.Stat()
	if err != nil {
		return false
	}
	return (info.Mode() & os.ModeCharDevice) != 0
}
