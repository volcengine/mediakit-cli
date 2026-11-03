//go:build !mediakit_no_update

package updatecheck

import (
	"os"
	"os/exec"
	"strings"

	"mediakit-cli/internal/buildenv"
)

func refreshEnvironment(environ []string) []string {
	denied := map[string]struct{}{
		buildenv.CloudAPIKey:        {},
		"MEDIAKIT_API_KEY":          {},
		"MEDIAKIT_BYTEPLUS_API_KEY": {},
	}
	filtered := make([]string, 0, len(environ)+1)
	for _, entry := range environ {
		name, _, found := strings.Cut(entry, "=")
		if found {
			if _, blocked := denied[name]; blocked {
				continue
			}
		}
		filtered = append(filtered, entry)
	}
	return append(filtered, envInternalRefresh+"=1")
}

// spawnRefresh starts a detached subprocess that runs the hidden
// `__update-refresh` command to fetch the latest version and persist the
// cache. It returns immediately; the child outlives this process so the cache
// is ready for the next invocation. All failures are silently ignored — update
// checks must never disrupt a user command.
func spawnRefresh() {
	exe, err := os.Executable()
	if err != nil {
		return
	}
	devNull, err := os.OpenFile(os.DevNull, os.O_RDWR, 0)
	if err != nil {
		return
	}
	defer devNull.Close()

	cmd := exec.Command(exe, InternalRefreshCommand)
	cmd.Stdin = devNull
	cmd.Stdout = devNull
	cmd.Stderr = devNull
	cmd.Env = refreshEnvironment(os.Environ())
	cmd.SysProcAttr = detachSysProcAttr()

	if err := cmd.Start(); err != nil {
		return
	}
	// Release so this process does not wait on or signal the child.
	_ = cmd.Process.Release()
}
