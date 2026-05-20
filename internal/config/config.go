package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

const (
	DefaultMode          = "cloud-first"
	DefaultSurface       = "cli"
	ModeLocalFirst       = "local-first"
	ModeCloudFirst       = "cloud-first"
	ConfigDirName        = ".mediakit"
	ConfigFileName       = "config.json"
	EnvCacheName         = "env_cache.json"
	DefaultOutputDirName = "temp"
	DefaultEndpoint      = "https://amk.cn-beijing.volces.com"
	EnvAPIKey            = "MEDIAKIT_API_KEY"
	EnvEndpoint          = "MEDIAKIT_ENDPOINT"
	EnvOutputPath        = "MEDIAKIT_OUTPUT_PATH"
	EnvSurface           = "MEDIAKIT_SURFACE"
	EnvRuntime           = "MEDIAKIT_RUNTIME"
)

type Config struct {
	Mode            string `json:"mode"`
	APIKey          string `json:"api_key,omitempty"`
	Endpoint        string `json:"endpoint,omitempty"`
	CredentialStore string `json:"credential_store,omitempty"`
	OutputPath      string `json:"output_path,omitempty"`
	Surface         string `json:"surface,omitempty"`
	Runtime         string `json:"runtime,omitempty"`
}

type ResolvedConfig struct {
	Mode             string
	APIKey           string
	Endpoint         string
	OutputPath       string
	APIKeySource     string
	EndpointSource   string
	OutputPathSource string
	ConfigPath       string
	EnvCachePath     string
	CredentialStore  string
	Surface          string
	Runtime          string
}

type ToolStatus struct {
	Available bool   `json:"available"`
	Version   string `json:"version,omitempty"`
	Reason    string `json:"reason,omitempty"`
}

type EnvCache struct {
	CheckedAt   string                `json:"checked_at"`
	Platform    string                `json:"platform"`
	DefaultMode string                `json:"default_mode"`
	CloudReady  bool                  `json:"cloud_ready"`
	LocalReady  bool                  `json:"local_ready"`
	Tools       map[string]ToolStatus `json:"tools"`
}

func DefaultConfig() Config {
	return Config{
		Mode:    DefaultMode,
		Surface: DefaultSurface,
	}
}

func ConfigDir(home string) string {
	return filepath.Join(home, ConfigDirName)
}

func ConfigFile(home string) string {
	return filepath.Join(ConfigDir(home), ConfigFileName)
}

func EnvCacheFile(home string) string {
	return filepath.Join(ConfigDir(home), EnvCacheName)
}

func DefaultOutputPath(home string) string {
	return filepath.Join(ConfigDir(home), DefaultOutputDirName)
}

func ResolveHomeDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return "", errors.New("unable to resolve user home directory")
	}
	return home, nil
}

func EnsureConfigDir(home string) error {
	return os.MkdirAll(ConfigDir(home), 0o755)
}

func EnsureOutputDir(path string) error {
	return os.MkdirAll(path, 0o755)
}

func ResolveOutputPath(home string) (string, string, error) {
	// Priority: env > config > default
	if value := strings.TrimSpace(os.Getenv(EnvOutputPath)); value != "" {
		outputPath, err := expandUserPath(value, home)
		if err != nil {
			return "", "", err
		}
		return outputPath, "env", nil
	}
	cfg, err := LoadConfig(home)
	if err == nil && strings.TrimSpace(cfg.OutputPath) != "" {
		outputPath, err := expandUserPath(cfg.OutputPath, home)
		if err != nil {
			return "", "", err
		}
		return outputPath, "config", nil
	}
	return DefaultOutputPath(home), "default", nil
}

func ValidateMode(mode string) error {
	switch mode {
	case ModeLocalFirst, ModeCloudFirst:
		return nil
	default:
		return fmt.Errorf("unsupported mode: %s", mode)
	}
}

func LoadConfig(home string) (Config, error) {
	cfg := DefaultConfig()

	data, err := os.ReadFile(ConfigFile(home))
	if errors.Is(err, os.ErrNotExist) {
		return cfg, nil
	}
	if err != nil {
		return Config{}, err
	}
	if len(data) == 0 {
		return cfg, nil
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return Config{}, err
	}
	if cfg.Mode == "" {
		cfg.Mode = DefaultMode
	}
	if cfg.Surface == "" {
		cfg.Surface = DefaultSurface
	}
	if err := ValidateMode(cfg.Mode); err != nil {
		cfg.Mode = DefaultMode
	}
	return cfg, nil
}

func SaveConfig(home string, cfg Config) error {
	if cfg.Mode == "" {
		cfg.Mode = DefaultMode
	}
	if cfg.Surface == "" {
		cfg.Surface = DefaultSurface
	}
	if err := ValidateMode(cfg.Mode); err != nil {
		return err
	}
	if err := EnsureConfigDir(home); err != nil {
		return err
	}
	return writeJSONAtomic(ConfigFile(home), cfg)
}

func ResolveConfig(home string) (ResolvedConfig, error) {
	fileCfg, err := LoadConfig(home)
	if err != nil {
		return ResolvedConfig{}, err
	}

	resolved := ResolvedConfig{
		Mode:             fileCfg.Mode,
		APIKey:           fileCfg.APIKey,
		Endpoint:         fileCfg.Endpoint,
		OutputPath:       DefaultOutputPath(home),
		APIKeySource:     "config",
		EndpointSource:   "config",
		OutputPathSource: "default",
		ConfigPath:       ConfigFile(home),
		EnvCachePath:     EnvCacheFile(home),
		CredentialStore:  fileCfg.CredentialStore,
		Surface:          fileCfg.Surface,
		Runtime:          fileCfg.Runtime,
	}
	if resolved.Mode == "" {
		resolved.Mode = DefaultMode
	}
	if resolved.Endpoint == "" {
		resolved.Endpoint = DefaultEndpoint
		resolved.EndpointSource = "default"
	}

	if value := strings.TrimSpace(os.Getenv(EnvAPIKey)); value != "" {
		resolved.APIKey = value
		resolved.APIKeySource = "env"
	}
	if value := strings.TrimSpace(os.Getenv(EnvEndpoint)); value != "" {
		resolved.Endpoint = value
		resolved.EndpointSource = "env"
	}
	if value := strings.TrimSpace(os.Getenv(EnvSurface)); value != "" {
		resolved.Surface = value
	}
	if value := strings.TrimSpace(os.Getenv(EnvRuntime)); value != "" {
		resolved.Runtime = value
	}
	outputPath, outputSource, err := ResolveOutputPath(home)
	if err != nil {
		return ResolvedConfig{}, err
	}
	resolved.OutputPath = outputPath
	resolved.OutputPathSource = outputSource
	return resolved, nil
}

func LoadEnvCache(home string) (EnvCache, error) {
	var cache EnvCache
	data, err := os.ReadFile(EnvCacheFile(home))
	if errors.Is(err, os.ErrNotExist) {
		return EnvCache{}, nil
	}
	if err != nil {
		return EnvCache{}, err
	}
	if len(data) == 0 {
		return EnvCache{}, nil
	}
	if err := json.Unmarshal(data, &cache); err != nil {
		return EnvCache{}, err
	}
	return cache, nil
}

func SaveEnvCache(home string, cache EnvCache) error {
	if err := EnsureConfigDir(home); err != nil {
		return err
	}
	return writeJSONAtomic(EnvCacheFile(home), cache)
}

func NewEnvCache(mode string) EnvCache {
	return EnvCache{
		CheckedAt:   time.Now().UTC().Format(time.RFC3339),
		Platform:    fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH),
		DefaultMode: mode,
		Tools:       map[string]ToolStatus{},
	}
}

func MaskSecret(value string) string {
	if value == "" {
		return ""
	}
	if len(value) <= 6 {
		return "***"
	}
	return value[:3] + strings.Repeat("*", len(value)-6) + value[len(value)-3:]
}

func expandUserPath(value string, home string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", fmt.Errorf("%s 不能为空", EnvOutputPath)
	}
	if value == "~" {
		value = home
	} else if strings.HasPrefix(value, "~"+string(filepath.Separator)) || strings.HasPrefix(value, "~/") {
		value = filepath.Join(home, strings.TrimPrefix(strings.TrimPrefix(value, "~"+string(filepath.Separator)), "~/"))
	}
	return filepath.Abs(filepath.Clean(value))
}

func writeJSONAtomic(path string, value any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".tmp-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpName, path)
}
