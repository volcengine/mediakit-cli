package cloud

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	cliconfig "mediakit-cli/internal/config"
)

const (
	envIdentityName          = "IDENTITY_NAME"
	envOpenClawServiceMarker = "OPENCLAW_SERVICE_MARKER"
)

func resolveHeaderValue(envName string, configValue string, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(envName)); value != "" {
		return value
	}
	if value := strings.TrimSpace(configValue); value != "" {
		return value
	}
	return fallback
}

func resolveSurface(configSurface string) string {
	const productSurface = "cli"
	value := resolveHeaderValue(cliconfig.EnvSurface, configSurface, productSurface)
	if value == "" || value == productSurface {
		return productSurface
	}
	if strings.HasPrefix(value, productSurface+"/") {
		return value
	}
	return productSurface + "/" + value
}

func resolveRuntime(configRuntime string) string {
	if value := strings.TrimSpace(os.Getenv(cliconfig.EnvRuntime)); value != "" {
		return value
	}
	if value := strings.TrimSpace(configRuntime); value != "" {
		return value
	}

	parts := make([]string, 0, 2)
	for _, envName := range []string{envIdentityName, envOpenClawServiceMarker} {
		if value := strings.TrimSpace(os.Getenv(envName)); value != "" {
			parts = append(parts, value)
		}
	}

	if len(parts) == 0 {
		return "unknown"
	}
	return strings.Join(parts, "/")
}

func (c *Client) newRequest(method string, path string, query map[string]any, body map[string]any) (*http.Request, error) {
	url := strings.TrimRight(c.Endpoint, "/") + "/" + strings.TrimLeft(path, "/")
	if len(query) > 0 {
		pairs := make([]string, 0, len(query))
		for key, value := range query {
			pairs = append(pairs, fmt.Sprintf("%s=%v", key, value))
		}
		url += "?" + strings.Join(pairs, "&")
	}

	var bodyReader io.Reader
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		bodyReader = bytes.NewReader(encoded)
	}

	req, err := http.NewRequest(method, url, bodyReader)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-surface", resolveSurface(c.Surface))
	req.Header.Set("x-runtime", resolveRuntime(c.Runtime))
	if c.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.APIKey)
	}
	return req, nil
}
