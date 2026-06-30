package updatecheck

import (
	"fmt"
	"io"
	"os"
)

// NoticePayload returns a map suitable to be embedded into the JSON stdout
// payload at the top-level `_notice.update` key, or nil if no update is needed.
func NoticePayload() map[string]any {
	r := GetResult()
	if r == nil || !r.HasUpdate {
		return nil
	}
	return map[string]any{
		"current": r.Current,
		"latest":  r.Latest,
		"command": fmt.Sprintf("npm install -g %s@latest", PackageName),
		"message": fmt.Sprintf("New %s release available: %s -> %s", PackageName, r.Current, r.Latest),
	}
}

// InjectNotice adds `_notice.update` to a top-level JSON object map if an
// update is available. It is a no-op when the result is nil or no update.
func InjectNotice(result map[string]any) {
	if result == nil {
		return
	}
	payload := NoticePayload()
	if payload == nil {
		return
	}
	notice, _ := result["_notice"].(map[string]any)
	if notice == nil {
		notice = map[string]any{}
	}
	notice["update"] = payload
	result["_notice"] = notice
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
	fmt.Fprintf(w, "\n[mediakit-cli] new version available: %s -> %s\n  run: npm install -g %s@latest\n",
		r.Current, r.Latest, PackageName)
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
