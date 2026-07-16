package commands

import "testing"

func TestRootCommandDoesNotExposeSkillsCommand(t *testing.T) {
	cmd := newRootCmd()
	for _, child := range cmd.Commands() {
		if child.Name() == "skills" {
			t.Fatal("root command exposes deprecated skills command")
		}
	}
}
