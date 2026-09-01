//go:build !windows && !mediakit_no_update

package updatecheck

import "syscall"

// detachSysProcAttr puts the refresh subprocess in its own session so it
// survives the parent's exit (no controlling terminal, no signal propagation).
func detachSysProcAttr() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{Setsid: true}
}
