//go:build !mediakit_cloud_only

package local

import "mediakit-cli/internal/local/core"

// Handler defines the execution contract for a local capability implementation.
type Handler = core.Handler
