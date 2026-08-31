//go:build windows && !mediakit_no_update

package updatecheck

import "syscall"

const (
	// From Windows process creation flags.
	createNewProcessGroup = 0x00000200 // CREATE_NEW_PROCESS_GROUP
	detachedProcess       = 0x00000008 // DETACHED_PROCESS
)

// detachSysProcAttr detaches the refresh subprocess from the parent console so
// it survives the parent's exit without a window or signal propagation.
func detachSysProcAttr() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{CreationFlags: createNewProcessGroup | detachedProcess}
}
