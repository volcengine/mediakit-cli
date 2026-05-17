package local

import (
	"strings"

	"mediakit-cli/internal/local/core"
	"mediakit-cli/internal/local/generated"
	"mediakit-cli/internal/local/manual"
)

var registeredHandlers = loadRegistrations()

// Registry will map capability names to local handlers in later stages.
type Registry struct{}

func Resolve(command string) (core.Registration, bool) {
	command = normalizeCommand(command)
	registration, ok := registeredHandlers[command]
	return registration, ok
}

func Has(command string) bool {
	_, ok := Resolve(command)
	return ok
}

func loadRegistrations() map[string]core.Registration {
	registrations := map[string]core.Registration{}
	mergeRegistrations(registrations, generated.Registrations(), false)
	mergeRegistrations(registrations, manual.Registrations(), true)
	return registrations
}

func mergeRegistrations(target map[string]core.Registration, entries []core.Registration, preferOverride bool) {
	for _, entry := range entries {
		key := normalizeCommand(entry.Command)
		if key == "" || entry.Handler == nil {
			continue
		}
		entry.Command = key
		entry.Dependencies = cloneStrings(entry.Dependencies)
		if _, exists := target[key]; exists && !preferOverride {
			continue
		}
		target[key] = entry
	}
}

func normalizeCommand(command string) string {
	command = strings.TrimSpace(command)
	command = strings.ReplaceAll(command, "_", "-")
	return strings.ToLower(command)
}

func cloneStrings(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	cloned := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		cloned = append(cloned, value)
	}
	return cloned
}
