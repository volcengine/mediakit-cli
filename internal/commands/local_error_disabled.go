//go:build mediakit_cloud_only

package commands

func localDependencyErrorPayload(error) (map[string]any, bool) {
	return nil, false
}
