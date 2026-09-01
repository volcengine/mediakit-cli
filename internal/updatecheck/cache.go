//go:build !mediakit_no_update

package updatecheck

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	cliconfig "mediakit-cli/internal/config"
)

const (
	CacheFileName = "update-check.json"
	CacheTTL      = 24 * time.Hour

	CacheStatusChecking = "checking"
	CacheStatusReady    = "ready"
	CacheStatusError    = "error"
)

type Cache struct {
	Current    string    `json:"current"`
	Latest     string    `json:"latest"`
	CheckedAt  time.Time `json:"checked_at"`
	HasUpdate  bool      `json:"has_update"`
	Status     string    `json:"status,omitempty"`
	Error      string    `json:"error,omitempty"`
	NotifiedAt time.Time `json:"notified_at,omitempty"`
}

func CacheFile(home string) (string, error) {
	cacheDir, err := cliconfig.CacheDir(home)
	if err != nil {
		return "", err
	}
	return filepath.Join(cacheDir, CacheFileName), nil
}

func LoadCache(home string) (*Cache, error) {
	path, err := CacheFile(home)
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	c := &Cache{}
	if err := json.Unmarshal(data, c); err != nil {
		return nil, err
	}
	return c, nil
}

func SaveCache(home string, c *Cache) error {
	if err := cliconfig.EnsureCacheDir(home); err != nil {
		return err
	}
	path, err := CacheFile(home)
	if err != nil {
		return err
	}
	return cliconfig.WriteJSONAtomic(path, c)
}

func IsCacheFresh(c *Cache, runningVersion string) bool {
	if c == nil {
		return false
	}
	if c.Current != runningVersion {
		return false
	}
	if time.Since(c.CheckedAt) > CacheTTL {
		return false
	}
	return true
}

func effectiveStatus(c *Cache) string {
	if c == nil {
		return ""
	}
	if c.Status != "" {
		return c.Status
	}
	if c.Latest != "" {
		return CacheStatusReady
	}
	return ""
}
