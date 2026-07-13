package updatecheck

import (
	"os"
	"testing"
	"time"

	"mediakit-cli/internal/build"
	"mediakit-cli/internal/skillstate"
)

func TestCheckNowForceRefreshBypassesFreshCache(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	oldVersion := build.Version
	build.Version = "0.1.8"
	t.Cleanup(func() {
		build.Version = oldVersion
	})
	if err := SaveCache(home, &Cache{
		Current:   "0.1.8",
		Latest:    "0.1.8",
		CheckedAt: time.Now(),
		HasUpdate: false,
		Status:    CacheStatusReady,
	}); err != nil {
		t.Fatal(err)
	}
	var calls int
	oldFetch := fetchLatestVersionFn
	fetchLatestVersionFn = func(time.Duration) (string, error) {
		calls++
		return "0.1.9", nil
	}
	t.Cleanup(func() {
		fetchLatestVersionFn = oldFetch
	})

	result := CheckNow(time.Second, true)
	if result == nil {
		t.Fatal("CheckNow returned nil")
	}
	if calls != 1 {
		t.Fatalf("fetch calls = %d, want 1", calls)
	}
	if result.Latest != "0.1.9" || !result.HasUpdate {
		t.Fatalf("result = %#v", result)
	}
}

func TestCheckNowWritesCheckingStatusWhileFetching(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	oldVersion := build.Version
	build.Version = "0.1.8"
	t.Cleanup(func() {
		build.Version = oldVersion
	})
	oldFetch := fetchLatestVersionFn
	fetchLatestVersionFn = func(time.Duration) (string, error) {
		cached, err := LoadCache(home)
		if err != nil {
			t.Fatal(err)
		}
		if cached == nil || cached.Status != CacheStatusChecking {
			t.Fatalf("cache while fetching = %#v, want checking", cached)
		}
		return "0.1.9", nil
	}
	t.Cleanup(func() {
		fetchLatestVersionFn = oldFetch
	})

	result := CheckNow(time.Second, true)
	if result == nil || result.Latest != "0.1.9" {
		t.Fatalf("result = %#v", result)
	}
}

func TestCheckingCacheSuppressesAllNotices(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	oldVersion := build.Version
	build.Version = "0.1.8"
	t.Cleanup(func() {
		build.Version = oldVersion
	})
	checkResult.Store(&Result{})
	if err := SaveCache(home, &Cache{
		Current:   "0.1.8",
		CheckedAt: time.Now(),
		Status:    CacheStatusChecking,
	}); err != nil {
		t.Fatal(err)
	}
	if err := skillstate.Save(home, &skillstate.State{
		PackageName: PackageName,
		Version:     "0.1.7",
	}); err != nil {
		t.Fatal(err)
	}

	payload := map[string]any{"ok": true}
	InjectNotice(payload)
	if _, ok := payload["_notice"]; ok {
		t.Fatalf("checking cache injected _notice: %#v", payload)
	}
}

func TestInjectNoticePrefersCliUpdateOverSkillsNotice(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	oldVersion := build.Version
	build.Version = "0.1.8"
	t.Cleanup(func() {
		build.Version = oldVersion
	})
	checkResult.Store(&Result{Current: "0.1.8", Latest: "0.1.9", HasUpdate: true})
	if err := skillstate.Save(home, &skillstate.State{
		PackageName: PackageName,
		Version:     "0.1.7",
	}); err != nil {
		t.Fatal(err)
	}

	payload := map[string]any{"ok": true}
	InjectNotice(payload)
	notice, ok := payload["_notice"].(map[string]any)
	if !ok {
		t.Fatalf("payload missing _notice: %#v", payload)
	}
	if _, ok := notice["update"].(map[string]any); !ok {
		t.Fatalf("notice missing update: %#v", notice)
	}
	if _, ok := notice["skills"]; ok {
		t.Fatalf("notice should not include skills when CLI update exists: %#v", notice)
	}
}

func TestInjectNoticeReportsSkillsWhenCliIsLatest(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	oldVersion := build.Version
	build.Version = "0.1.9"
	t.Cleanup(func() {
		build.Version = oldVersion
	})
	checkResult.Store(&Result{Current: "0.1.9", Latest: "0.1.9", HasUpdate: false})
	if err := skillstate.Save(home, &skillstate.State{
		PackageName: PackageName,
		Version:     "0.1.8",
	}); err != nil {
		t.Fatal(err)
	}

	payload := map[string]any{"ok": true}
	InjectNotice(payload)
	notice, ok := payload["_notice"].(map[string]any)
	if !ok {
		t.Fatalf("payload missing _notice: %#v", payload)
	}
	skills, ok := notice["skills"].(map[string]any)
	if !ok {
		t.Fatalf("notice missing skills: %#v", notice)
	}
	if skills["target"] != "0.1.9" || skills["command"] != "mediakit-cli update --force" {
		t.Fatalf("skills notice = %#v", skills)
	}
}

func TestMain(m *testing.M) {
	checkResult.Store(&Result{})
	os.Exit(m.Run())
}
