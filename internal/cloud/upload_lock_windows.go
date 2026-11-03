//go:build windows

package cloud

import (
	"errors"
	"os"
	"syscall"
	"unsafe"
)

const (
	lockfileExclusiveLock   = 0x00000002
	lockfileFailImmediately = 0x00000001
)

var (
	kernel32                       = syscall.NewLazyDLL("kernel32.dll")
	procLockFileEx                 = kernel32.NewProc("LockFileEx")
	procUnlockFileEx               = kernel32.NewProc("UnlockFileEx")
	errLockViolation syscall.Errno = 33
)

func lockUploadCacheFile(file *os.File) error {
	var overlapped syscall.Overlapped
	r1, _, err := procLockFileEx.Call(
		file.Fd(), lockfileExclusiveLock|lockfileFailImmediately, 0,
		0xffffffff, 0xffffffff, uintptr(unsafe.Pointer(&overlapped)),
	)
	if r1 == 0 {
		return err
	}
	return nil
}

func unlockUploadCacheFile(file *os.File) error {
	var overlapped syscall.Overlapped
	r1, _, err := procUnlockFileEx.Call(
		file.Fd(), 0, 0xffffffff, 0xffffffff, uintptr(unsafe.Pointer(&overlapped)),
	)
	if r1 == 0 {
		return err
	}
	return nil
}

func isUploadCacheLockBusy(err error) bool { return errors.Is(err, errLockViolation) }
