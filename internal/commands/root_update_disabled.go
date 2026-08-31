//go:build mediakit_no_update

package commands

import "github.com/spf13/cobra"

func rootLongDescription() string {
	return "MediaKit CLI provides Cloud media-processing commands and domain navigation."
}

func configureUpdate(*cobra.Command) {}
