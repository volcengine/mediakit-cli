package skillstate

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"time"

	cliconfig "mediakit-cli/internal/config"
)

const (
	FileName    = "skills-state.json"
	PackageName = "@volcengine/mediakit-cli"
)

type State struct {
	PackageName string    `json:"package_name"`
	Version     string    `json:"version"`
	SkillsDir   string    `json:"skills_dir,omitempty"`
	InstalledAt time.Time `json:"installed_at,omitempty"`
}

type Status struct {
	Current string `json:"current,omitempty"`
	Target  string `json:"target"`
	InSync  bool   `json:"in_sync"`
	Command string `json:"command"`
	Missing bool   `json:"missing,omitempty"`
}

func File(home string) string {
	return filepath.Join(cliconfig.ConfigDir(home), FileName)
}

func Load(home string) (*State, error) {
	data, err := os.ReadFile(File(home))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	if len(data) == 0 {
		return nil, nil
	}
	var state State
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, err
	}
	return &state, nil
}

func Save(home string, state *State) error {
	if state == nil {
		return nil
	}
	if state.PackageName == "" {
		state.PackageName = PackageName
	}
	return cliconfig.WriteJSONAtomic(File(home), state)
}

func ReadStatus(home, target string) (*Status, error) {
	target = normalizeVersion(target)
	state, err := Load(home)
	if err != nil {
		return nil, err
	}
	status := &Status{
		Target:  target,
		Command: "mediakit-cli update --force",
	}
	if state == nil || strings.TrimSpace(state.Version) == "" {
		status.Missing = true
		return status, nil
	}
	status.Current = normalizeVersion(state.Version)
	status.InSync = status.Current == target
	return status, nil
}

func InSync(home, target string) bool {
	status, err := ReadStatus(home, target)
	return err == nil && status != nil && status.InSync
}

func normalizeVersion(version string) string {
	version = strings.TrimSpace(version)
	version = strings.TrimPrefix(version, "v")
	return strings.TrimPrefix(version, "V")
}
