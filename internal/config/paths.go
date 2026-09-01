package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode"

	"mediakit-cli/internal/buildenv"
)

const (
	ConfigDirName   = ".mediakit"
	ConfigFileName  = "config.json"
	UploadCacheName = "upload_cache.json"
)

func ConfigDir(home string) string {
	return filepath.Join(home, ConfigDirName)
}

func ConfigFile(home string) string {
	return filepath.Join(ConfigDir(home), ConfigFileName)
}

// CacheDir resolves the non-secret cache directory compiled into the binary.
// It intentionally has no environment, flag, config, XDG, migration or
// fallback path.
func CacheDir(home string) (string, error) {
	literal := buildenv.CacheDir
	if err := validateCacheDirLiteral(literal); err != nil {
		return "", err
	}
	if strings.HasPrefix(literal, "~/") {
		if home == "" || !filepath.IsAbs(home) {
			return "", fmt.Errorf("unable to resolve compiled cache directory: invalid home")
		}
		return filepath.Join(
			filepath.Clean(home),
			filepath.FromSlash(strings.TrimPrefix(literal, "~/")),
		), nil
	}
	resolved := filepath.Clean(filepath.FromSlash(literal))
	if !filepath.IsAbs(resolved) {
		return "", fmt.Errorf(
			"compiled cache directory %q is not absolute on this platform",
			literal,
		)
	}
	return resolved, nil
}

func UploadCacheFile(home string) (string, error) {
	cacheDir, err := CacheDir(home)
	if err != nil {
		return "", err
	}
	return filepath.Join(cacheDir, UploadCacheName), nil
}

func EnsureConfigDir(home string) error {
	return os.MkdirAll(ConfigDir(home), 0o755)
}

func EnsureCacheDir(home string) error {
	cacheDir, err := CacheDir(home)
	if err != nil {
		return err
	}
	return os.MkdirAll(cacheDir, 0o755)
}

func validateCacheDirLiteral(literal string) error {
	if literal == "" || literal != strings.TrimSpace(literal) {
		return fmt.Errorf("compiled cache directory must be a normalized non-empty path")
	}
	for _, character := range literal {
		if unicode.IsSpace(character) ||
			character < 0x21 ||
			character == 0x7f ||
			strings.ContainsRune("\"'`\\=", character) {
			return fmt.Errorf("compiled cache directory contains unsafe characters")
		}
	}

	var remainder string
	switch {
	case strings.HasPrefix(literal, "~/"):
		remainder = strings.TrimPrefix(literal, "~/")
	case strings.HasPrefix(literal, "/"):
		remainder = strings.TrimPrefix(literal, "/")
	case len(literal) >= 3 &&
		((literal[0] >= 'A' && literal[0] <= 'Z') ||
			(literal[0] >= 'a' && literal[0] <= 'z')) &&
		literal[1] == ':' &&
		literal[2] == '/':
		remainder = literal[3:]
	default:
		return fmt.Errorf(
			"compiled cache directory must use ~/..., a POSIX absolute path, or a drive-qualified absolute path",
		)
	}
	if remainder == "" {
		return fmt.Errorf("compiled cache directory cannot be a filesystem root")
	}
	for _, part := range strings.Split(remainder, "/") {
		if part == "" || part == "." || part == ".." {
			return fmt.Errorf(
				"compiled cache directory contains an empty, dot, or parent segment",
			)
		}
	}
	return nil
}
