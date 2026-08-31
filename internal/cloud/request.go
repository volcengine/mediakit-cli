package cloud

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strings"
	"unicode/utf8"

	"mediakit-cli/internal/build"
	"mediakit-cli/internal/buildenv"
)

const (
	envIdentityName          = "IDENTITY_NAME"
	envOpenClawServiceMarker = "OPENCLAW_SERVICE_MARKER"
	envTermProgram           = "TERM_PROGRAM"
	envTerminalEmulator      = "TERMINAL_EMULATOR"
	maxAiAgentNameLength     = 128
)

var aiAgentNamePattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]*$`)

func resolveRuntime(configRuntime string) string {
	return resolveRuntimeFrom(configRuntime, os.Environ())
}

func resolveTaskSource() (string, error) {
	const productSource = "cli"
	value := strings.TrimSpace(os.Getenv(buildenv.TaskSource))
	if strings.ContainsAny(value, "\r\n") {
		return "", fmt.Errorf(
			"%s 包含不允许的换行字符",
			buildenv.TaskSource,
		)
	}
	value = strings.Trim(value, "/")
	if strings.EqualFold(strings.TrimSpace(os.Getenv(buildenv.Surface)), "skill") {
		value = appendTaskSourceSegment(value, "skill")
	}
	if value == "" || value == productSource {
		return productSource, nil
	}
	if strings.HasPrefix(value, productSource+"/") {
		return value, nil
	}
	return productSource + "/" + value, nil
}

func appendTaskSourceSegment(value string, segment string) string {
	parts := strings.Split(strings.Trim(value, "/"), "/")
	for _, part := range parts {
		if part == segment {
			return value
		}
	}
	if value == "" {
		return segment
	}
	return strings.Trim(value, "/") + "/" + segment
}

func resolveRuntimeFrom(configRuntime string, environ []string) string {
	env := parseEnviron(environ)

	if value := strings.TrimSpace(env[buildenv.Runtime]); value != "" {
		return normalizeRuntime(value)
	}
	if value := strings.TrimSpace(configRuntime); value != "" {
		return normalizeRuntime(value)
	}

	if value := detectClientEnvFrom(environ); value != "" {
		return value
	}

	identity := strings.TrimSpace(env[envIdentityName])
	if strings.Contains(strings.ToLower(identity), "arkclaw") {
		return "arkclaw"
	}
	if hasNonEmptyEnvPrefix(env, "ARKCLAW_") {
		return "arkclaw"
	}
	if hasNonEmptyEnvPrefix(env, "OPENCLAW_") {
		return "openclaw"
	}
	if hasNonEmptyEnvPrefix(env, "HERMES_") {
		return "hermes"
	}
	if _, ok := env["COZE_CLAW_AGENT_ID"]; ok {
		return "coze-claw"
	}
	if identity != "" {
		return identity
	}
	return "unknown"
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
	trimmedEqualsIgnoreCase := func(name string, expected string) bool {
		return strings.EqualFold(strings.TrimSpace(env[name]), expected)
	}

	if trimmedEqualsIgnoreCase("AI_AGENT", "trae") {
		return "trae"
	}
	if customAiAgent := normalizeAiAgentName(env["AI_AGENT"]); customAiAgent != "" {
		return customAiAgent
	}

	if hasNonEmptyEnvPrefix(env, "DOUBAO_OFFICE_") {
		return "doubao"
	}

	hasClaudeCode := present("CLAUDECODE") || present("CLAUDE_CODE") || present("CLAUDE_CODE_ENTRYPOINT")
	if hasClaudeCode && present("CLAUDE_CODE_IS_COWORK") {
		return "claude-cowork"
	}
	if hasClaudeCode {
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
	if hasNonEmptyEnvPrefix(env, "TRAE_") ||
		present("COCO_PLUGIN_ROOT") ||
		trimmedEqualsIgnoreCase("ICUBE_PRODUCT_BRAND_NAME", "trae") {
		return "trae"
	}
	if hasNonEmptyEnvPrefix(env, "WINDSURF_") ||
		containsValue("VSCODE_GIT_ASKPASS_MAIN", "windsurf") ||
		containsValue("VSCODE_GIT_ASKPASS_NODE", "windsurf") ||
		containsValue("__CFBundleIdentifier", "windsurf") {
		return "windsurf"
	}

	if present("GEMINI_CLI") {
		return "gemini-cli"
	}
	if present("KIRO_SESSION_ID") || present("KIRO_AGENT_PATH") {
		return "kiro"
	}
	if present("OPENCODE") || present("OPENCODE_CLIENT") {
		return "opencode"
	}
	if present("ANTIGRAVITY_AGENT") {
		return "antigravity"
	}
	if present("COPILOT_CLI") || present("COPILOT_MODEL") || present("COPILOT_ALLOW_ALL") {
		return "github-copilot"
	}
	if present("CLINE_ACTIVE") {
		return "cline"
	}
	if present("AMP_CURRENT_THREAD_ID") {
		return "amp"
	}
	if env["PI_CODING_AGENT"] == "true" {
		return "pi"
	}
	if present("REPL_ID") {
		return "replit"
	}
	if present("AUGMENT_AGENT") {
		return "augment"
	}
	if present("QWEN_CODE") {
		return "qwen-code"
	}
	if present("COPILOT_GITHUB_TOKEN") {
		return "github-copilot"
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

func normalizeAiAgentName(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" || utf8.RuneCountInString(trimmed) > maxAiAgentNameLength {
		return ""
	}
	if !aiAgentNamePattern.MatchString(trimmed) {
		return ""
	}
	return trimmed
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
	if strings.Contains(lower, "claude-cowork") {
		return "claude-cowork"
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
	if strings.Contains(lower, "hermes") {
		return "hermes"
	}
	if strings.Contains(lower, "coze-claw") || strings.Contains(lower, "cozeclaw") {
		return "coze-claw"
	}
	if strings.Contains(lower, "doubao") {
		return "doubao"
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
	if strings.Contains(lower, "gemini") {
		return "gemini-cli"
	}
	if strings.Contains(lower, "kiro") {
		return "kiro"
	}
	if strings.Contains(lower, "opencode") {
		return "opencode"
	}
	if strings.Contains(lower, "antigravity") {
		return "antigravity"
	}
	if strings.Contains(lower, "copilot") {
		return "github-copilot"
	}
	if strings.Contains(lower, "cline") {
		return "cline"
	}
	if lower == "amp" || strings.HasPrefix(lower, "amp-") {
		return "amp"
	}
	if lower == "pi" || strings.HasPrefix(lower, "pi-") {
		return "pi"
	}
	if strings.Contains(lower, "replit") {
		return "replit"
	}
	if strings.Contains(lower, "augment") {
		return "augment"
	}
	if strings.Contains(lower, "qwen") {
		return "qwen-code"
	}
	if strings.Contains(lower, "vscode") {
		return "vscode"
	}
	return lower
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
	req.Header.Set("X-Amk-Cli-Runtime", resolveRuntime(c.Runtime))
	taskSource, err := resolveTaskSource()
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-Amk-Task-Source", taskSource)
	req.Header.Set("X-Amk-Cli-Version", build.Version)
	c.Auth.Apply(req)
	return req, nil
}
