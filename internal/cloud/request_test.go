package cloud

import (
	"testing"

	cliconfig "mediakit-cli/internal/config"
)

func TestDetectClientEnvFrom(t *testing.T) {
	tests := []struct {
		name    string
		environ []string
		want    string
	}{
		{
			name:    "claude code via CLAUDECODE",
			environ: []string{"CLAUDECODE=1", "TERM_PROGRAM=vscode"},
			want:    "claude-code",
		},
		{
			name:    "claude code via entrypoint",
			environ: []string{"CLAUDE_CODE_ENTRYPOINT=cli"},
			want:    "claude-code",
		},
		{
			name:    "jetbrains via terminal emulator",
			environ: []string{"TERMINAL_EMULATOR=JetBrains-JediTerm"},
			want:    "jetbrains",
		},
		{
			name:    "warp via term program",
			environ: []string{"TERM_PROGRAM=WarpTerminal"},
			want:    "warp",
		},
		{
			name:    "cursor wins over vscode term program",
			environ: []string{"CURSOR_TRACE_ID=abc", "TERM_PROGRAM=vscode"},
			want:    "cursor",
		},
		{
			name:    "trae wins over vscode term program",
			environ: []string{"TRAE_BRAND_NAME=Trae", "TERM_PROGRAM=vscode"},
			want:    "trae",
		},
		{
			name:    "windsurf wins over vscode term program",
			environ: []string{"WINDSURF_SESSION=1", "TERM_PROGRAM=vscode"},
			want:    "windsurf",
		},
		{
			name:    "plain vscode",
			environ: []string{"TERM_PROGRAM=vscode"},
			want:    "vscode",
		},
		{
			name:    "other terminal lowercased",
			environ: []string{"TERM_PROGRAM=iTerm.app"},
			want:    "iterm.app",
		},
		{
			name:    "no client signal",
			environ: []string{"PATH=/usr/bin", "HOME=/home/user"},
			want:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := detectClientEnvFrom(tt.environ); got != tt.want {
				t.Fatalf("detectClientEnvFrom() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestResolveRuntimeFromPriority(t *testing.T) {
	tests := []struct {
		name          string
		configRuntime string
		environ       []string
		want          string
	}{
		{
			name:          "env var wins over everything",
			configRuntime: "config-runtime",
			environ:       []string{cliconfig.EnvRuntime + "=explicit-runtime", "TRAE_BRAND_NAME=Trae"},
			want:          "explicit-runtime",
		},
		{
			name:          "config wins over auto detection",
			configRuntime: "config-runtime",
			environ:       []string{"TRAE_BRAND_NAME=Trae"},
			want:          "config-runtime",
		},
		{
			name:          "auto detection wins over service marker",
			configRuntime: "",
			environ:       []string{"CLAUDECODE=1", envIdentityName + "=svc"},
			want:          "claude-code",
		},
		{
			name:          "service marker fallback",
			configRuntime: "",
			environ:       []string{envIdentityName + "=svc", envOpenClawServiceMarker + "=marker"},
			want:          "svc/marker",
		},
		{
			name:          "unknown when no signal",
			configRuntime: "",
			environ:       []string{"PATH=/usr/bin"},
			want:          "unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := resolveRuntimeFrom(tt.configRuntime, tt.environ); got != tt.want {
				t.Fatalf("resolveRuntimeFrom() = %q, want %q", got, tt.want)
			}
		})
	}
}
