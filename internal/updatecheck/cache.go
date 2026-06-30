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
)

type Cache struct {
	Current    string    `json:"current"`
	Latest     string    `json:"latest"`
	CheckedAt  time.Time `json:"checked_at"`
	HasUpdate  bool      `json:"has_update"`
	NotifiedAt time.Time `json:"notified_at,omitempty"`
}

func CacheFile(home string) string {
	return filepath.Join(cliconfig.ConfigDir(home), CacheFileName)
}

func LoadCache(home string) (*Cache, error) {
	path := CacheFile(home)
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
	dir := cliconfig.ConfigDir(home)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(CacheFile(home), data, 0o644)
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
