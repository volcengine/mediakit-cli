//go:build !mediakit_cloud_only

package commands

import (
	"errors"

	"mediakit-cli/internal/local"
)

func localDependencyErrorPayload(err error) (map[string]any, bool) {
	var dependencyError *local.DependencyError
	if !errors.As(err, &dependencyError) {
		return nil, false
	}
	return dependencyError.StructuredError(), true
}
