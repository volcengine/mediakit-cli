//go:build !mediakit_cloud_only

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

	"mediakit-cli/internal/buildenv"
)

const (
	DefaultMode          = "cloud-first"
	ModeLocalFirst       = "local-first"
	ModeCloudFirst       = "cloud-first"
	EnvCacheName         = "env_cache.json"
	DefaultOutputDirName = "temp"
	DefaultEndpoint      = "https://mediakit.cn-beijing.volces.com"
)

var (
	EnvEndpoint   = buildenv.CloudEndpoint
	EnvOutputPath = buildenv.OutputPath
	EnvRuntime    = buildenv.Runtime
)

type Config struct {
	Mode               string `json:"mode"`
	Endpoint           string `json:"endpoint,omitempty"`
	OutputPath         string `json:"output_path,omitempty"`
	Runtime            string `json:"runtime,omitempty"`
	DisableUpdateCheck bool   `json:"disable_update_check,omitempty"`
}

type ResolvedConfig struct {
	Mode             string
	Endpoint         string
	OutputPath       string
	EndpointSource   string
	OutputPathSource string
	ConfigPath       string
	CacheDir         string
	EnvCachePath     string
	Runtime          string

	DisableUpdateCheck bool
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
		Mode: DefaultMode,
	}
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

func EnsureOutputDir(path string) error {
	return os.MkdirAll(path, 0o755)
}

func EnvCacheFile(home string) (string, error) {
	cacheDir, err := CacheDir(home)
	if err != nil {
		return "", err
	}
	return filepath.Join(cacheDir, EnvCacheName), nil
}

func ResolveOutputPath(home string) (string, string, error) {
	cfg, err := LoadConfig(home)
	if err != nil {
		return "", "", err
	}
	return ResolveOutputPathFromConfig(home, cfg)
}

func ResolveOutputPathFromConfig(
	home string,
	cfg Config,
) (string, string, error) {
	// Priority: env > provided config > default
	if value := strings.TrimSpace(os.Getenv(EnvOutputPath)); value != "" {
		outputPath, err := expandUserPath(value, home)
		if err != nil {
			return "", "", err
		}
		return outputPath, "env", nil
	}
	if strings.TrimSpace(cfg.OutputPath) != "" {
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
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		return Config{}, err
	}
	if value, ok := raw["mode"].(string); ok {
		cfg.Mode = strings.TrimSpace(value)
	}
	if value, ok := raw["endpoint"].(string); ok {
		cfg.Endpoint = strings.TrimSpace(value)
	}
	if value, ok := raw["output_path"].(string); ok {
		cfg.OutputPath = strings.TrimSpace(value)
	}
	if value, ok := raw["runtime"].(string); ok {
		cfg.Runtime = strings.TrimSpace(value)
	}
	if value, ok := raw["disable_update_check"].(bool); ok {
		cfg.DisableUpdateCheck = value
	}
	if cfg.Mode == "" {
		cfg.Mode = DefaultMode
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
	if err := ValidateMode(cfg.Mode); err != nil {
		return err
	}
	if err := EnsureConfigDir(home); err != nil {
		return err
	}
	raw, err := loadConfigObject(home)
	if err != nil {
		return err
	}
	raw["mode"] = cfg.Mode
	setOptionalString(raw, "endpoint", cfg.Endpoint)
	setOptionalString(raw, "output_path", cfg.OutputPath)
	setOptionalString(raw, "runtime", cfg.Runtime)
	if cfg.DisableUpdateCheck {
		raw["disable_update_check"] = true
	} else {
		delete(raw, "disable_update_check")
	}
	return WriteJSONAtomic(ConfigFile(home), raw)
}

func ResolveConfig(home string) (ResolvedConfig, error) {
	fileCfg, err := LoadConfig(home)
	if err != nil {
		return ResolvedConfig{}, err
	}
	disableUpdateCheck, err := resolveDisableUpdateCheck(
		fileCfg.DisableUpdateCheck,
	)
	if err != nil {
		return ResolvedConfig{}, err
	}
	cacheDir, err := CacheDir(home)
	if err != nil {
		return ResolvedConfig{}, err
	}
	envCachePath, err := EnvCacheFile(home)
	if err != nil {
		return ResolvedConfig{}, err
	}

	resolved := ResolvedConfig{
		Mode:             fileCfg.Mode,
		Endpoint:         fileCfg.Endpoint,
		OutputPath:       DefaultOutputPath(home),
		EndpointSource:   "config",
		OutputPathSource: "default",
		ConfigPath:       ConfigFile(home),
		CacheDir:         cacheDir,
		EnvCachePath:     envCachePath,
		Runtime:          fileCfg.Runtime,

		DisableUpdateCheck: disableUpdateCheck,
	}
	if resolved.Mode == "" {
		resolved.Mode = DefaultMode
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
	outputPath, outputSource, err := ResolveOutputPathFromConfig(home, fileCfg)
	if err != nil {
		return ResolvedConfig{}, err
	}
	resolved.OutputPath = outputPath
	resolved.OutputPathSource = outputSource
	return resolved, nil
}

func resolveDisableUpdateCheck(configured bool) (bool, error) {
	value, exists := os.LookupEnv(buildenv.DisableUpdateCheck)
	value = strings.TrimSpace(value)
	if !exists || value == "" {
		return configured, nil
	}
	switch strings.ToLower(value) {
	case "true":
		return true, nil
	case "false":
		return false, nil
	default:
		return false, fmt.Errorf(
			"%s 必须是 true 或 false",
			buildenv.DisableUpdateCheck,
		)
	}
}

// ResolveUpdateCheckDisabled applies the runtime environment override to the
// persisted setting. It is shared by config display and update execution.
func ResolveUpdateCheckDisabled(home string) (bool, error) {
	cfg, err := LoadConfig(home)
	if err != nil {
		return false, err
	}
	return resolveDisableUpdateCheck(cfg.DisableUpdateCheck)
}

// UpdateCheckDisabledByConfig reports whether the persisted config disables the
// version update check. Missing or unreadable config is treated as "enabled".
func UpdateCheckDisabledByConfig(home string) bool {
	cfg, err := LoadConfig(home)
	if err != nil {
		return false
	}
	return cfg.DisableUpdateCheck
}

func LoadEnvCache(home string) (EnvCache, error) {
	var cache EnvCache
	path, err := EnvCacheFile(home)
	if err != nil {
		return EnvCache{}, err
	}
	data, err := os.ReadFile(path)
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
	if err := EnsureCacheDir(home); err != nil {
		return err
	}
	path, err := EnvCacheFile(home)
	if err != nil {
		return err
	}
	return WriteJSONAtomic(path, cache)
}

func NewEnvCache(mode string) EnvCache {
	return EnvCache{
		CheckedAt:   time.Now().UTC().Format(time.RFC3339),
		Platform:    fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH),
		DefaultMode: mode,
		Tools:       map[string]ToolStatus{},
	}
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
