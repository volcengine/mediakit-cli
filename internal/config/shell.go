//go:build !mediakit_cloud_only

package config

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

func DetectShellProfile(home string) (string, error) {
	if runtime.GOOS == "windows" {
		return "", errors.New("automatic shell profile updates are not supported on windows")
	}

	shell := strings.ToLower(os.Getenv("SHELL"))
	switch {
	case strings.Contains(shell, "zsh"):
		return filepath.Join(home, ".zshrc"), nil
	case strings.Contains(shell, "bash"):
		return filepath.Join(home, ".bashrc"), nil
	default:
		return filepath.Join(home, ".zshrc"), nil
	}
}
