package cloud

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	cliconfig "mediakit-cli/internal/config"
)

const (
	uploadCacheVersion = 1
	uploadCacheTTL     = 12 * time.Hour
	maxCacheEntries    = 1000
	uploadLockTimeout  = 5 * time.Second
	uploadLockStaleAge = 10 * time.Minute
)

type uploadCache struct {
	Version int                         `json:"version"`
	Entries map[string]uploadCacheEntry `json:"entries"`
	Scopes  map[string]*scopeBucket     `json:"scopes,omitempty"`
}

type scopeBucket struct {
	Entries map[string]uploadCacheEntry `json:"entries"`
}

type uploadCacheEntry struct {
	FileID        string `json:"file_id"`
	UploadedAt    string `json:"uploaded_at"`
	ExpiresAt     string `json:"expires_at"`
	Size          int64  `json:"size"`
	MTimeUnixNano int64  `json:"mtime_unix_nano"`
	Auth          string `json:"auth,omitempty"`
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

func computeUploadCacheAuth(endpoint string, authIdentity string) string {
	// upload-cache-v1 is a domain-separation prefix; NUL bytes delimit fields.
	material := "upload-cache-v1\x00" +
		normalizeUploadCacheEndpoint(endpoint) + "\x00" +
		strings.TrimSpace(authIdentity)
	sum := sha256.Sum256([]byte(material))
	return hex.EncodeToString(sum[:])
}

func normalizeUploadCacheEndpoint(endpoint string) string {
	endpoint = strings.TrimSpace(endpoint)
	endpoint = strings.TrimRight(endpoint, "/")
	return endpoint
}

func lookupUploadCache(home string, authKey string, identity fileIdentity, now time.Time) (string, error) {
	lock, err := acquireUploadCacheLock(home)
	if err != nil {
		return "", nil
	}
	defer lock.Release()

	cache, err := readUploadCache(home)
	if err != nil {
		return "", nil
	}
	changed := cache.normalizeFromLegacyFormats()
	if pruned := sweepUploadCacheEntries(&cache, authKey, now); pruned > 0 || changed {
		_ = writeUploadCache(home, cache)
	}

	entry, ok := cache.Entries[identity.AbsPath]
	if !ok || !entry.matches(identity, authKey, now) {
		return "", nil
	}
	return entry.FileID, nil
}

func storeUploadCache(home string, authKey string, identity fileIdentity, fileID string, now time.Time) (string, error) {
	lock, err := acquireUploadCacheLock(home)
	if err != nil {
		return fileID, nil
	}
	defer lock.Release()

	cache, err := readUploadCache(home)
	if err != nil {
		cache = uploadCache{
			Version: uploadCacheVersion,
			Entries: map[string]uploadCacheEntry{},
		}
	}
	cache.normalizeFromLegacyFormats()
	sweepUploadCacheEntries(&cache, authKey, now)

	if entry, ok := cache.Entries[identity.AbsPath]; ok && entry.matches(identity, authKey, now) {
		return entry.FileID, nil
	}

	if cache.Entries == nil {
		cache.Entries = map[string]uploadCacheEntry{}
	}
	cache.Entries[identity.AbsPath] = uploadCacheEntry{
		FileID:        fileID,
		UploadedAt:    now.Format(time.RFC3339),
		ExpiresAt:     now.Add(uploadCacheTTL).Format(time.RFC3339),
		Size:          identity.Size,
		MTimeUnixNano: identity.MTimeUnixNano,
		Auth:          authKey,
	}
	enforceEntryLimit(cache.Entries, maxCacheEntries)
	_ = writeUploadCache(home, cache)
	return fileID, nil
}

func (entry uploadCacheEntry) matches(identity fileIdentity, authKey string, now time.Time) bool {
	if entry.FileID == "" {
		return false
	}
	if strings.TrimSpace(entry.Auth) != authKey {
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

func (cache *uploadCache) normalizeFromLegacyFormats() bool {
	changed := false
	if len(cache.Scopes) > 0 {
		if cache.Entries == nil {
			cache.Entries = map[string]uploadCacheEntry{}
		}
		for scopeKey, bucket := range cache.Scopes {
			if bucket == nil {
				continue
			}
			for path, entry := range bucket.Entries {
				entry.Auth = scopeKey
				cache.Entries[path] = entry
			}
		}
		cache.Scopes = nil
		changed = true
	}
	if cache.Entries == nil {
		cache.Entries = map[string]uploadCacheEntry{}
		changed = true
	}
	if cache.Version == 0 {
		cache.Version = uploadCacheVersion
		changed = true
	}
	return changed
}

func sweepUploadCacheEntries(cache *uploadCache, authKey string, now time.Time) int {
	if cache.Entries == nil {
		return 0
	}
	pruned := 0
	for path, entry := range cache.Entries {
		if shouldRemoveCacheEntry(entry, authKey, now) {
			delete(cache.Entries, path)
			pruned++
		}
	}
	return pruned
}

func shouldRemoveCacheEntry(entry uploadCacheEntry, authKey string, now time.Time) bool {
	if strings.TrimSpace(entry.Auth) != authKey {
		return true
	}
	expiresAt, err := time.Parse(time.RFC3339, entry.ExpiresAt)
	if err != nil || !now.Before(expiresAt) {
		return true
	}
	return false
}

func readUploadCache(home string) (uploadCache, error) {
	cache := uploadCache{
		Version: uploadCacheVersion,
		Entries: map[string]uploadCacheEntry{},
	}
	path, err := cliconfig.UploadCacheFile(home)
	if err != nil {
		return cache, err
	}
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return cache, nil
	}
	if err != nil {
		return uploadCache{Version: uploadCacheVersion, Entries: map[string]uploadCacheEntry{}}, nil
	}
	if len(data) == 0 {
		return cache, nil
	}
	if err := json.Unmarshal(data, &cache); err != nil {
		return uploadCache{Version: uploadCacheVersion, Entries: map[string]uploadCacheEntry{}}, nil
	}
	if cache.Entries == nil {
		cache.Entries = map[string]uploadCacheEntry{}
	}
	return cache, nil
}

func writeUploadCache(home string, cache uploadCache) error {
	cache.normalizeFromLegacyFormats()
	cache.Version = uploadCacheVersion
	cache.Scopes = nil
	if cache.Entries == nil {
		cache.Entries = map[string]uploadCacheEntry{}
	}
	path, err := cliconfig.UploadCacheFile(home)
	if err != nil {
		return err
	}
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

func enforceEntryLimit(entries map[string]uploadCacheEntry, maxEntries int) {
	if len(entries) <= maxEntries {
		return
	}
	type entryTime struct {
		path       string
		uploadedAt time.Time
	}
	times := make([]entryTime, 0, len(entries))
	for path, entry := range entries {
		uploadedAt, err := time.Parse(time.RFC3339, entry.UploadedAt)
		if err != nil {
			uploadedAt = time.Time{}
		}
		times = append(times, entryTime{path: path, uploadedAt: uploadedAt})
	}
	sort.Slice(times, func(i, j int) bool {
		return times[i].uploadedAt.Before(times[j].uploadedAt)
	})
	for len(entries) > maxEntries {
		delete(entries, times[0].path)
		times = times[1:]
	}
}

func acquireUploadCacheLock(home string) (*cacheLock, error) {
	if err := cliconfig.EnsureCacheDir(home); err != nil {
		return nil, err
	}
	cacheFile, err := cliconfig.UploadCacheFile(home)
	if err != nil {
		return nil, err
	}
	lockPath := cacheFile + ".lock"
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
