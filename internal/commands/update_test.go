package commands

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	buildinfo "mediakit-cli/internal/build"
	"mediakit-cli/internal/skillstate"
	"mediakit-cli/internal/updatecheck"

	"github.com/spf13/cobra"
)

func TestUpdateCheckDoesNotInstallOrSyncSkills(t *testing.T) {
	restore := stubUpdateDeps(t, &updatecheck.Result{Current: "0.1.7", Latest: "0.1.8", HasUpdate: true})
	defer restore()

	var npmCalled bool
	runNpmInstall = func(*cobra.Command) error {
		npmCalled = true
		return nil
	}
	var skillsCalled bool
	runSkillsInstall = func(*cobra.Command) error {
		skillsCalled = true
		return nil
	}

	cmd := newUpdateCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"--check", "--json"})

	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if npmCalled {
		t.Fatal("npm install called during update --check")
	}
	if skillsCalled {
		t.Fatal("skills install called during update --check")
	}
}

func TestUpdateCheckJsonIncludesSkillsStatus(t *testing.T) {
	restore := stubUpdateDeps(t, &updatecheck.Result{Current: "0.1.7", Latest: "0.1.7", HasUpdate: false})
	defer restore()
	home := mustHome(t)
	if err := skillstate.Save(home, &skillstate.State{
		PackageName: "@volcengine/mediakit-cli",
		Version:     "0.1.6",
		SkillsDir:   "/tmp/old-skills",
		InstalledAt: time.Date(
			2026, 7, 3, 12, 0, 0, 0, time.UTC,
		),
	}); err != nil {
		t.Fatal(err)
	}

	cmd := newUpdateCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"--check", "--json"})

	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	var payload map[string]any
	if err := json.Unmarshal(out.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	status, ok := payload["skills_status"].(map[string]any)
	if !ok {
		t.Fatalf("payload missing skills_status: %#v", payload)
	}
	if status["current"] != "0.1.6" || status["target"] != "0.1.7" || status["in_sync"] != false {
		t.Fatalf("skills_status = %#v", status)
	}
	if status["command"] != "mediakit-cli update --force" {
		t.Fatalf("skills_status = %#v", status)
	}
}

func TestUpdateCheckDefaultOutputMatchesHumanNotice(t *testing.T) {
	restore := stubUpdateDeps(t, &updatecheck.Result{Current: "0.1.7", Latest: "0.1.8", HasUpdate: true})
	defer restore()

	cmd := newUpdateCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"--check"})

	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	got := out.String()
	if !strings.Contains(got, "Update available: 0.1.7 → 0.1.8") {
		t.Fatalf("output = %q", got)
	}
	if !strings.Contains(got, "Run `mediakit-cli update` to install.") {
		t.Fatalf("output = %q", got)
	}
	if strings.Contains(got, "{") {
		t.Fatalf("default check output should not be JSON: %q", got)
	}
}

func TestUpdateDoesNotInstallSkillsWhenAlreadyLatest(t *testing.T) {
	restore := stubUpdateDeps(t, &updatecheck.Result{Current: "0.1.7", Latest: "0.1.7", HasUpdate: false})
	defer restore()
	home := mustHome(t)
	if err := skillstate.Save(home, &skillstate.State{
		PackageName: "@volcengine/mediakit-cli",
		Version:     "0.1.7",
	}); err != nil {
		t.Fatal(err)
	}
	var skillsCalled bool
	runSkillsInstall = func(*cobra.Command) error {
		skillsCalled = true
		return nil
	}

	cmd := newUpdateCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"--json"})

	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	var payload map[string]any
	if err := json.Unmarshal(out.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if _, ok := payload["skills_action"]; ok {
		t.Fatalf("payload = %#v", payload)
	}
	if skillsCalled {
		t.Fatal("skills install called when CLI is already latest")
	}
}

func TestUpdateInstallsSkillsWhenAlreadyLatestButSkillsOutOfSync(t *testing.T) {
	restore := stubUpdateDeps(t, &updatecheck.Result{Current: "0.1.7", Latest: "0.1.7", HasUpdate: false})
	defer restore()
	home := mustHome(t)
	if err := skillstate.Save(home, &skillstate.State{
		PackageName: "@volcengine/mediakit-cli",
		Version:     "0.1.6",
	}); err != nil {
		t.Fatal(err)
	}
	var skillsCalled bool
	runSkillsInstall = func(*cobra.Command) error {
		skillsCalled = true
		return nil
	}

	cmd := newUpdateCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&bytes.Buffer{})

	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	got := out.String()
	if !strings.Contains(got, "mediakit-cli 0.1.7 is already up to date") {
		t.Fatalf("output = %q", got)
	}
	if !strings.Contains(got, "Updating skills ...") || !strings.Contains(got, "✓ Skills updated") {
		t.Fatalf("output = %q", got)
	}
	if !skillsCalled {
		t.Fatal("skills install not called when skills are out of sync")
	}
}

func TestUpdateForceInstallsSkillsWhenAlreadyLatest(t *testing.T) {
	restore := stubUpdateDeps(t, &updatecheck.Result{Current: "0.1.7", Latest: "0.1.7", HasUpdate: false})
	defer restore()
	var skillsCalled bool
	runSkillsInstall = func(*cobra.Command) error {
		skillsCalled = true
		return nil
	}

	cmd := newUpdateCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"--force"})

	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	got := out.String()
	if !strings.Contains(got, "mediakit-cli 0.1.7 is already up to date") {
		t.Fatalf("output = %q", got)
	}
	if !strings.Contains(got, "Updating skills ...") || !strings.Contains(got, "✓ Skills updated") {
		t.Fatalf("output = %q", got)
	}
	if strings.Contains(got, "{") {
		t.Fatalf("default output should not be JSON: %q", got)
	}
	if !skillsCalled {
		t.Fatal("skills install not called under update --force")
	}
}

func TestCurrentPackageSpecUsesRunningVersionForSkillsReinstall(t *testing.T) {
	oldVersion := buildinfo.Version
	buildinfo.Version = "0.1.8-beta.2"
	t.Cleanup(func() {
		buildinfo.Version = oldVersion
	})

	if got, want := currentPackageSpec(), "@volcengine/mediakit-cli@0.1.8-beta.2"; got != want {
		t.Fatalf("currentPackageSpec() = %q, want %q", got, want)
	}
}

func TestUpdateReliesOnNpmPostinstallForSkillsAfterNpmUpdate(t *testing.T) {
	restore := stubUpdateDeps(t, &updatecheck.Result{Current: "0.1.7", Latest: "0.1.8", HasUpdate: true})
	defer restore()

	var calls []string
	runNpmInstall = func(*cobra.Command) error {
		calls = append(calls, "npm")
		return nil
	}
	runSkillsInstall = func(*cobra.Command) error {
		calls = append(calls, "skills")
		return nil
	}

	cmd := newUpdateCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"--json"})

	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if got, want := strings.Join(calls, ","), "npm"; got != want {
		t.Fatalf("calls = %q, want %q", got, want)
	}
	var payload map[string]any
	if err := json.Unmarshal(out.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if _, ok := payload["skills_action"]; ok {
		t.Fatalf("payload = %#v", payload)
	}
}

func TestUpdateDefaultOutputSeparatesCliAndSkillsStatus(t *testing.T) {
	restore := stubUpdateDeps(t, &updatecheck.Result{Current: "0.1.7", Latest: "0.1.8", HasUpdate: true})
	defer restore()

	cmd := newUpdateCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&bytes.Buffer{})

	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	got := out.String()
	for _, want := range []string{
		"Updating mediakit-cli 0.1.7 → 0.1.8 via npm ...",
		"✓ Successfully updated mediakit-cli from 0.1.7 to 0.1.8",
		"Updating skills ...",
		"✓ Skills updated",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("output missing %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, "Changelog:") {
		t.Fatalf("output should not include changelog:\n%s", got)
	}
}

func TestUpdateDoesNotRunSkillsInstallAfterNpmUpdate(t *testing.T) {
	restore := stubUpdateDeps(t, &updatecheck.Result{Current: "0.1.7", Latest: "0.1.8", HasUpdate: true})
	defer restore()

	runSkillsInstall = func(*cobra.Command) error { return errors.New("skills should not run") }

	cmd := newUpdateCmd()
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"--json"})

	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
}

func TestInjectNoticeIncludesSkillsNoticeWhenOutOfSync(t *testing.T) {
	restore := stubUpdateDeps(t, &updatecheck.Result{Current: "0.1.7", Latest: "0.1.7", HasUpdate: false})
	defer restore()
	home := mustHome(t)
	if err := skillstate.Save(home, &skillstate.State{
		PackageName: "@volcengine/mediakit-cli",
		Version:     "0.1.6",
	}); err != nil {
		t.Fatal(err)
	}

	payload := map[string]any{"ok": true}
	updatecheck.InjectNotice(payload)

	notice, ok := payload["_notice"].(map[string]any)
	if !ok {
		t.Fatalf("payload missing _notice: %#v", payload)
	}
	skills, ok := notice["skills"].(map[string]any)
	if !ok {
		t.Fatalf("notice missing skills: %#v", notice)
	}
	if skills["current"] != "0.1.6" || skills["target"] != buildinfo.Version {
		t.Fatalf("skills notice = %#v", skills)
	}
	if skills["command"] != "mediakit-cli update --force" {
		t.Fatalf("skills notice = %#v", skills)
	}
}

func TestUpdateDoesNotSyncWhenNpmInstallFails(t *testing.T) {
	restore := stubUpdateDeps(t, &updatecheck.Result{Current: "0.1.7", Latest: "0.1.8", HasUpdate: true})
	defer restore()

	runNpmInstall = func(*cobra.Command) error { return errors.New("npm failed") }
	var skillsCalled bool
	runSkillsInstall = func(*cobra.Command) error {
		skillsCalled = true
		return nil
	}

	cmd := newUpdateCmd()
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})

	if err := cmd.Execute(); err == nil {
		t.Fatal("Execute returned nil error, want npm failure")
	}
	if skillsCalled {
		t.Fatal("skills install called after npm failure")
	}
}

func stubUpdateDeps(t *testing.T, result *updatecheck.Result) func() {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	oldCheck := checkNow
	oldNpm := runNpmInstall
	oldSkills := runSkillsInstall
	checkNow = func() *updatecheck.Result { return result }
	runNpmInstall = func(*cobra.Command) error { return nil }
	runSkillsInstall = func(*cobra.Command) error { return nil }
	return func() {
		checkNow = oldCheck
		runNpmInstall = oldNpm
		runSkillsInstall = oldSkills
	}
}

func mustHome(t *testing.T) string {
	t.Helper()
	home := os.Getenv("HOME")
	if home == "" {
		t.Fatal("HOME not set")
	}
	return filepath.Clean(home)
}
