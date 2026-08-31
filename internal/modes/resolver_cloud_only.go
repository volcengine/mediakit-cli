//go:build mediakit_cloud_only

package modes

import (
	"github.com/spf13/cobra"

	"mediakit-cli/internal/auth"
	"mediakit-cli/internal/cloud"
	cliconfig "mediakit-cli/internal/config"
)

type CapabilityRuntimeMeta struct {
	Name                   string
	Domain                 string
	Description            string
	CloudOnly              bool
	LocalSupported         bool
	LocalSource            string
	LocalDeps              []string
	LocalUnsupportedParams []string
}

type Decision struct {
	Mode    string
	Warning string
}

type Resolver struct{}

func LocalSurfaceVisible() bool {
	return false
}

func SchemaMode(
	cmd *cobra.Command,
	meta CapabilityRuntimeMeta,
) (string, error) {
	return "cloud", nil
}

func Dispatch(cmd *cobra.Command, meta CapabilityRuntimeMeta, params map[string]any) error {
	home, err := cliconfig.ResolveHomeDir()
	if err != nil {
		return err
	}
	resolved, err := cliconfig.ResolveConfig(home)
	if err != nil {
		return err
	}
	authContext, err := auth.Resolve()
	if err != nil {
		return err
	}
	return cloud.Execute(
		cmd,
		meta.Name,
		params,
		authContext,
		resolved.Endpoint,
		resolved.Runtime,
	)
}

func ModeLabel(CapabilityRuntimeMeta) string {
	return ""
}

func ApplyRuntimeConstraints(meta CapabilityRuntimeMeta) CapabilityRuntimeMeta {
	meta.CloudOnly = true
	meta.LocalSupported = false
	meta.LocalSource = ""
	meta.LocalDeps = nil
	meta.LocalUnsupportedParams = nil
	return meta
}
