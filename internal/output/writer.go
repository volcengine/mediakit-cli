package output

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Writer validates output paths and writes files atomically inside the cwd.
type Writer struct {
	WorkDir string
}

func NewWriter(workDir string) (Writer, error) {
	if strings.TrimSpace(workDir) == "" {
		var err error
		workDir, err = os.Getwd()
		if err != nil {
			return Writer{}, err
		}
	}
	return Writer{WorkDir: workDir}, nil
}

func (w Writer) ResolvePath(rawPath string) (string, error) {
	rawPath = strings.TrimSpace(rawPath)
	if rawPath == "" {
		return "", fmt.Errorf("输出路径不能为空")
	}

	baseAbs, err := filepath.Abs(w.WorkDir)
	if err != nil {
		return "", err
	}
	baseReal, err := resolveFuturePath(baseAbs)
	if err != nil {
		return "", err
	}

	targetPath := rawPath
	if !filepath.IsAbs(targetPath) {
		targetPath = filepath.Join(baseAbs, targetPath)
	}
	targetReal, err := resolveFuturePath(targetPath)
	if err != nil {
		return "", err
	}

	rel, err := filepath.Rel(baseReal, targetReal)
	if err != nil {
		return "", err
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("输出路径必须位于当前工作目录之下")
	}
	return targetReal, nil
}

func (w Writer) WriteFileAtomic(rawPath string, data []byte, perm os.FileMode, overwrite bool) (string, error) {
	targetPath, err := w.ResolvePath(rawPath)
	if err != nil {
		return "", err
	}

	if _, statErr := os.Stat(targetPath); statErr == nil && !overwrite {
		return "", fmt.Errorf("输出文件已存在，请显式开启 overwrite")
	}

	parentDir := filepath.Dir(targetPath)
	if mkdirErr := os.MkdirAll(parentDir, 0o755); mkdirErr != nil {
		return "", mkdirErr
	}

	tempFile, err := os.CreateTemp(parentDir, ".mediakit-atomic-*")
	if err != nil {
		return "", err
	}
	tempPath := tempFile.Name()
	cleanup := true
	defer func() {
		if cleanup {
			_ = os.Remove(tempPath)
		}
	}()

	if _, err := tempFile.Write(data); err != nil {
		_ = tempFile.Close()
		return "", err
	}
	if err := tempFile.Chmod(perm); err != nil {
		_ = tempFile.Close()
		return "", err
	}
	if err := tempFile.Close(); err != nil {
		return "", err
	}
	if err := os.Rename(tempPath, targetPath); err != nil {
		return "", err
	}

	cleanup = false
	return targetPath, nil
}

func resolveFuturePath(value string) (string, error) {
	value, err := filepath.Abs(value)
	if err != nil {
		return "", err
	}
	value = filepath.Clean(value)

	current := value
	suffix := make([]string, 0, 4)
	for {
		if _, err := os.Lstat(current); err == nil {
			resolved, err := filepath.EvalSymlinks(current)
			if err != nil {
				return "", err
			}
			for i := len(suffix) - 1; i >= 0; i-- {
				resolved = filepath.Join(resolved, suffix[i])
			}
			return filepath.Clean(resolved), nil
		}

		parent := filepath.Dir(current)
		if parent == current {
			return "", fmt.Errorf("无法解析路径: %s", value)
		}
		suffix = append(suffix, filepath.Base(current))
		current = parent
	}
}
