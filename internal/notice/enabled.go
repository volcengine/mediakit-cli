//go:build !mediakit_no_notice

package notice

import "mediakit-cli/internal/updatecheck"

func Inject(payload map[string]any) {
	updatecheck.InjectNotice(payload)
}
