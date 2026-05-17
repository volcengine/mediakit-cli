package manual

import "mediakit-cli/internal/local/core"

// Registrations returns all hand-maintained local handlers.
func Registrations() []core.Registration {
	return []core.Registration{
		// __MANUAL_LOCAL_HANDLERS__
	}
}
