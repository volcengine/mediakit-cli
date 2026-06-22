package cloud

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"time"

	cliconfig "mediakit-cli/internal/config"
)

const (
	uploadCacheVersion = 1
	uploadCacheTTL     = 30 * 24 * time.Hour
	uploadLockTimeout  = 5 * time.Second
	uploadLockStaleAge = 10 * time.Minute
)

type uploadCache struct {
	Version int                         `json:"version"`
	Entries map[string]uploadCacheEntry `json:"entries"`
}

type uploadCacheEntry struct {
	FileID        string `json:"file_id"`
	UploadedAt    string `json:"uploaded_at"`
	ExpiresAt     string `json:"expires_at"`
	Size          int64  `json:"size"`
	MTimeUnixNano int64  `json:"mtime_unix_nano"`
}

type fileIdentity struct {
	AbsPath       string
	Size          int64
	MTimeUnixNano int64
}

type cacheLock struct {
	path string
	file *os.File
}

func lookupUploadCache(home string, identity fileIdentity, now time.Time) (string, error) {
	lock, err := acquireUploadCacheLock(home)
	if err != nil {
		return "", err
	}
	defer lock.Release()

	cache, err := readUploadCache(home)
	if err != nil {
		return "", err
	}
	entry, ok := cache.Entries[identity.AbsPath]
	if !ok || !entry.matches(identity, now) {
		return "", nil
	}
	return entry.FileID, nil
}

func storeUploadCache(home string, identity fileIdentity, fileID string, now time.Time) (string, error) {
	lock, err := acquireUploadCacheLock(home)
	if err != nil {
		return "", err
	}
	defer lock.Release()

	cache, err := readUploadCache(home)
	if err != nil {
		return "", err
	}
	if entry, ok := cache.Entries[identity.AbsPath]; ok && entry.matches(identity, now) {
		return entry.FileID, nil
	}

	cache.Entries[identity.AbsPath] = uploadCacheEntry{
		FileID:        fileID,
		UploadedAt:    now.Format(time.RFC3339),
		ExpiresAt:     now.Add(uploadCacheTTL).Format(time.RFC3339),
		Size:          identity.Size,
		MTimeUnixNano: identity.MTimeUnixNano,
	}
	pruneExpiredUploadCache(cache, now)
	return fileID, writeUploadCache(home, cache)
}

func (entry uploadCacheEntry) matches(identity fileIdentity, now time.Time) bool {
	if entry.FileID == "" {
		return false
	}
	if entry.Size != identity.Size || entry.MTimeUnixNano != identity.MTimeUnixNano {
		return false
	}
	expiresAt, err := time.Parse(time.RFC3339, entry.ExpiresAt)
	if err != nil {
		return false
	}
	return now.Before(expiresAt)
}

func readUploadCache(home string) (uploadCache, error) {
	cache := uploadCache{
		Version: uploadCacheVersion,
		Entries: map[string]uploadCacheEntry{},
	}
	data, err := os.ReadFile(cliconfig.UploadCacheFile(home))
	if errors.Is(err, os.ErrNotExist) {
		return cache, nil
	}
	if err != nil {
		return cache, err
	}
	if len(data) == 0 {
		return cache, nil
	}
	if err := json.Unmarshal(data, &cache); err != nil {
		return uploadCache{Version: uploadCacheVersion, Entries: map[string]uploadCacheEntry{}}, nil
	}
	if cache.Version == 0 {
		cache.Version = uploadCacheVersion
	}
	if cache.Entries == nil {
		cache.Entries = map[string]uploadCacheEntry{}
	}
	return cache, nil
}

func writeUploadCache(home string, cache uploadCache) error {
	if cache.Version == 0 {
		cache.Version = uploadCacheVersion
	}
	if cache.Entries == nil {
		cache.Entries = map[string]uploadCacheEntry{}
	}
	path := cliconfig.UploadCacheFile(home)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(cache, "", "  ")
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".upload-cache-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpName, path)
}

func pruneExpiredUploadCache(cache uploadCache, now time.Time) {
	for path, entry := range cache.Entries {
		expiresAt, err := time.Parse(time.RFC3339, entry.ExpiresAt)
		if err != nil || !now.Before(expiresAt) {
			delete(cache.Entries, path)
		}
	}
}

func acquireUploadCacheLock(home string) (*cacheLock, error) {
	if err := cliconfig.EnsureConfigDir(home); err != nil {
		return nil, err
	}
	lockPath := cliconfig.UploadCacheFile(home) + ".lock"
	deadline := time.Now().Add(uploadLockTimeout)
	for {
		file, err := os.OpenFile(lockPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
		if err == nil {
			_, _ = file.WriteString(time.Now().UTC().Format(time.RFC3339Nano))
			return &cacheLock{path: lockPath, file: file}, nil
		}
		if !errors.Is(err, os.ErrExist) {
			return nil, err
		}
		if isStaleUploadLock(lockPath) {
			_ = os.Remove(lockPath)
			continue
		}
		if time.Now().After(deadline) {
			return nil, errors.New("等待上传缓存锁超时")
		}
		time.Sleep(50 * time.Millisecond)
	}
}

func isStaleUploadLock(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return time.Since(info.ModTime()) > uploadLockStaleAge
}

func (lock *cacheLock) Release() {
	if lock == nil {
		return
	}
	if lock.file != nil {
		_ = lock.file.Close()
	}
	if lock.path != "" {
		_ = os.Remove(lock.path)
	}
}
