//go:build !mediakit_cloud_only

package generated

import (
	"fmt"
	"os"
	"strings"

	"mediakit-cli/internal/local/core"
)

type FetchFileHandler struct{}

func (h *FetchFileHandler) Execute(ctx *core.ExecContext) (map[string]any, error) {
	source, err := requiredStringParam(ctx.Params, "url")
	if err != nil {
		return nil, err
	}

	if core.IsRemoteURL(source) {
		localPath, err := core.FetchRemoteFile(ctx.OutputDir, source)
		if err != nil {
			return nil, err
		}
		return map[string]any{
			"source":     source,
			"sourceType": "remote",
			"localPath":  localPath,
		}, nil
	}

	localPath, err := core.ResolveLocalPath(ctx, source)
	if err != nil {
		return nil, err
	}
	if _, err := os.Stat(localPath); err != nil {
		return nil, fmt.Errorf("本地文件不可用: %w", err)
	}
	return map[string]any{
		"source":     source,
		"sourceType": "local",
		"localPath":  strings.TrimSpace(localPath),
	}, nil
}
