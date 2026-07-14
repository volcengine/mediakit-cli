package cloud

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"mediakit-cli/internal/build"
	cliconfig "mediakit-cli/internal/config"
)

const (
	envIdentityName          = "IDENTITY_NAME"
	envOpenClawServiceMarker = "OPENCLAW_SERVICE_MARKER"
	envTermProgram           = "TERM_PROGRAM"
	envTerminalEmulator      = "TERMINAL_EMULATOR"
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
	return resolveRuntimeFrom(configRuntime, os.Environ())
}

func resolveRuntimeFrom(configRuntime string, environ []string) string {
	env := parseEnviron(environ)

	if value := strings.TrimSpace(env[cliconfig.EnvRuntime]); value != "" {
		return normalizeRuntime(value)
	}
	if value := strings.TrimSpace(configRuntime); value != "" {
		return normalizeRuntime(value)
	}

	workspace := detectWorkspaceRuntimeFrom(environ)
	client := detectClientEnvFrom(environ)
	if workspace != "" && client != "" {
		if workspace == client {
			return workspace
		}
		return workspace + "," + client
	}
	if workspace != "" {
		return workspace
	}
	if client != "" {
		return client
	}

	if hasNonEmptyEnvPrefix(env, "OPENCLAW_") {
		return "openclaw"
	}
	identity := strings.TrimSpace(env[envIdentityName])
	if identity != "" {
		return identity
	}
	return "unknown"
}

// detectWorkspaceRuntimeFrom identifies an outer workspace/sandbox host. This is
// kept separate from detectClientEnvFrom so nested executions can report both
// layers, for example "openclaw,claude-code".
func detectWorkspaceRuntimeFrom(environ []string) string {
	env := parseEnviron(environ)

	if hasNonEmptyEnvPrefix(env, "OPENCLAW_") {
		return "openclaw"
	}
	if hasNonEmptyEnvPrefix(env, "ARKCLAW_") ||
		strings.Contains(strings.ToLower(strings.TrimSpace(env[envIdentityName])), "arkclaw") {
		return "arkclaw"
	}
	if hasNonEmptyEnvPrefix(env, "HERMES_") {
		return "hermes"
	}
	return ""
}

// detectClientEnvFrom identifies the calling IDE/Agent host from environment
// signals. Ordering matters: dedicated strong signals are checked before the
// shared TERM_PROGRAM=vscode signal, because VS Code forks (Cursor, Trae,
// Windsurf) all report TERM_PROGRAM=vscode and would otherwise be misdetected.
// Returns an empty string when no client signal is present.
func detectClientEnvFrom(environ []string) string {
	env := parseEnviron(environ)

	present := func(name string) bool { return strings.TrimSpace(env[name]) != "" }
	containsValue := func(name string, needle string) bool {
		return strings.Contains(strings.ToLower(strings.TrimSpace(env[name])), needle)
	}

	// First tier: strong, dedicated signals.
	if strings.TrimSpace(env["CLAUDECODE"]) == "1" || present("CLAUDE_CODE_ENTRYPOINT") {
		return "claude-code"
	}
	if hasNonEmptyEnvPrefix(env, "CODEX_") {
		return "codex"
	}
	if strings.Contains(strings.ToLower(env[envTerminalEmulator]), "jetbrains") {
		return "jetbrains"
	}
	if strings.EqualFold(strings.TrimSpace(env[envTermProgram]), "WarpTerminal") {
		return "warp"
	}

	// Second tier: VS Code fork combination check. Dedicated prefixes win over
	// the shared TERM_PROGRAM=vscode fallback.
	if present("CURSOR_TRACE_ID") || hasNonEmptyEnvPrefix(env, "CURSOR_") {
		return "cursor"
	}
	if hasNonEmptyEnvPrefix(env, "TRAE_") {
		return "trae"
	}
	if hasNonEmptyEnvPrefix(env, "WINDSURF_") ||
		containsValue("VSCODE_GIT_ASKPASS_MAIN", "windsurf") ||
		containsValue("VSCODE_GIT_ASKPASS_NODE", "windsurf") ||
		containsValue("__CFBundleIdentifier", "windsurf") {
		return "windsurf"
	}

	term := strings.TrimSpace(env[envTermProgram])
	if strings.EqualFold(term, "vscode") {
		return "vscode"
	}
	if term != "" {
		return strings.ToLower(term)
	}

	return ""
}

// normalizeRuntime maps a user-provided runtime identifier to its canonical
// form, mirroring the keyword matching in detectClientEnvFrom so that injected
// values (e.g. MEDIAKIT_RUNTIME=claude) produce the same result as auto-
// detection (claude-code).
func normalizeRuntime(value string) string {
	lower := strings.ToLower(strings.TrimSpace(value))
	if lower == "" {
		return ""
	}
	if strings.Contains(lower, "claude") {
		return "claude-code"
	}
	if strings.Contains(lower, "codex") {
		return "codex"
	}
	if strings.Contains(lower, "openclaw") {
		return "openclaw"
	}
	if strings.Contains(lower, "arkclaw") {
		return "arkclaw"
	}
	if strings.Contains(lower, "jetbrains") {
		return "jetbrains"
	}
	if strings.Contains(lower, "warp") {
		return "warp"
	}
	if strings.Contains(lower, "cursor") {
		return "cursor"
	}
	if strings.Contains(lower, "trae") {
		return "trae"
	}
	if strings.Contains(lower, "windsurf") {
		return "windsurf"
	}
	if strings.Contains(lower, "vscode") {
		return "vscode"
	}
	return lower
}

func PreviewHeaders(surface string, runtime string) map[string]string {
	return map[string]string{
		"Accept":            "application/json",
		"Content-Type":      "application/json",
		"x-surface":         resolveSurface(surface),
		"X-Amk-Cli-Runtime": resolveRuntime(runtime),
		"X-Amk-Task-Source": "cli",
		"X-Amk-Cli-Version": build.Version,
	}
}

func parseEnviron(environ []string) map[string]string {
	env := make(map[string]string, len(environ))
	for _, entry := range environ {
		key, value, found := strings.Cut(entry, "=")
		if !found {
			continue
		}
		env[key] = value
	}
	return env
}

func hasNonEmptyEnvPrefix(env map[string]string, prefix string) bool {
	for key, value := range env {
		if strings.HasPrefix(key, prefix) && strings.TrimSpace(value) != "" {
			return true
		}
	}
	return false
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
	req.Header.Set("X-Amk-Cli-Runtime", resolveRuntime(c.Runtime))
	req.Header.Set("X-Amk-Task-Source", "cli")
	req.Header.Set("X-Amk-Cli-Version", build.Version)
	if c.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.APIKey)
	}
	return req, nil
}
