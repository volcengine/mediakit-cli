//go:build mediakit_cloud_only

package config

import (
	"errors"
	"os"
	"strings"

	"mediakit-cli/internal/buildenv"
)

const (
	DefaultMode     = "cloud-first"
	ModeLocalFirst  = "local-first"
	ModeCloudFirst  = "cloud-first"
	DefaultEndpoint = "https://mediakit.cn-beijing.volces.com"
)

var (
	EnvEndpoint = buildenv.CloudEndpoint
	EnvRuntime  = buildenv.Runtime
)

type Config struct {
	Mode     string `json:"mode"`
	Endpoint string `json:"endpoint,omitempty"`
	Runtime  string `json:"runtime,omitempty"`

	originalEndpoint string
	originalRuntime  string
}

type ResolvedConfig struct {
	Mode           string
	Endpoint       string
	EndpointSource string
	ConfigPath     string
	CacheDir       string
	Runtime        string
}

func DefaultConfig() Config {
	return Config{Mode: ModeCloudFirst}
}

func ResolveHomeDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return "", errors.New("unable to resolve user home directory")
	}
	return home, nil
}

func LoadConfig(home string) (Config, error) {
	cfg := DefaultConfig()
	raw, err := loadConfigObject(home)
	if err != nil {
		return Config{}, err
	}
	if value, ok := raw["endpoint"].(string); ok {
		cfg.Endpoint = strings.TrimSpace(value)
	}
	if value, ok := raw["runtime"].(string); ok {
		cfg.Runtime = strings.TrimSpace(value)
	}
	cfg.originalEndpoint = cfg.Endpoint
	cfg.originalRuntime = cfg.Runtime
	return cfg, nil
}

func SaveConfig(home string, cfg Config) error {
	raw, err := loadConfigObject(home)
	if err != nil {
		return err
	}
	changed := false
	if cfg.Endpoint != cfg.originalEndpoint {
		setOptionalString(raw, "endpoint", cfg.Endpoint)
		changed = true
	}
	if cfg.Runtime != cfg.originalRuntime {
		setOptionalString(raw, "runtime", cfg.Runtime)
		changed = true
	}
	if !changed {
		return nil
	}
	return WriteJSONAtomic(ConfigFile(home), raw)
}

func ResolveConfig(home string) (ResolvedConfig, error) {
	cfg, err := LoadConfig(home)
	if err != nil {
		return ResolvedConfig{}, err
	}
	cacheDir, err := CacheDir(home)
	if err != nil {
		return ResolvedConfig{}, err
	}
	resolved := ResolvedConfig{
		Mode:           ModeCloudFirst,
		Endpoint:       cfg.Endpoint,
		EndpointSource: "config",
		ConfigPath:     ConfigFile(home),
		CacheDir:       cacheDir,
		Runtime:        cfg.Runtime,
	}
	if resolved.Endpoint == "" {
		resolved.Endpoint = DefaultEndpoint
		resolved.EndpointSource = "default"
	}
	if value := strings.TrimSpace(os.Getenv(EnvEndpoint)); value != "" {
		resolved.Endpoint = value
		resolved.EndpointSource = "env"
	}
	if value := strings.TrimSpace(os.Getenv(EnvRuntime)); value != "" {
		resolved.Runtime = value
	}
	return resolved, nil
}
