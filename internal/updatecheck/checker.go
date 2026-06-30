package updatecheck

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync/atomic"
	"time"

	"mediakit-cli/internal/build"
	cliconfig "mediakit-cli/internal/config"
)

const (
	NpmRegistryURL = "https://registry.npmjs.org/@volcengine/mediakit-cli/latest"
	PackageName    = "@volcengine/mediakit-cli"
	EnvDisable     = "MEDIAKIT_DISABLE_UPDATE_CHECK"
	EnvCI          = "CI"

	// InternalRefreshCommand is the hidden subcommand the detached refresh
	// subprocess runs. It only fetches the latest version and writes the cache.
	InternalRefreshCommand = "__update-refresh"
	// envInternalRefresh marks the detached subprocess so it never re-spawns.
	envInternalRefresh = "MEDIAKIT_INTERNAL_REFRESH"

	// refreshFetchTimeout bounds the registry fetch inside the detached
	// subprocess. It can be generous since it never blocks a user command.
	refreshFetchTimeout = 5 * time.Second
)

type Result struct {
	HasUpdate bool
	Current   string
	Latest    string
	Err       error
}

var checkResult atomic.Value // *Result

// StartAsync wires the non-blocking update notice for normal commands.
// It only reads the local cache synchronously: a fresh real result is stored
// for the stdout/stderr notice path. When the cache is missing or stale it
// claims a refresh (placeholder debounce) and spawns a detached subprocess to
// fetch + persist the latest version, then returns immediately. The current
// command never waits on the network.
func StartAsync() {
	if isInternalRefresh() || shouldSkip() {
		return
	}
	running := strings.TrimSpace(build.Version)
	if running == "" {
		return
	}
	home, err := cliconfig.ResolveHomeDir()
	if err != nil {
		return
	}
	if r := loadCachedResult(home, running); r != nil {
		checkResult.Store(r)
		return
	}
	if !claimRefresh(home, running) {
		return
	}
	spawnRefresh()
}

// RunRefresh is the detached subprocess entry: fetch the latest version and
// persist the cache. Failures leave the placeholder in place so the next
// attempt only happens after the TTL elapses (no per-command hammering).
func RunRefresh() {
	if shouldSkip() {
		return
	}
	running := strings.TrimSpace(build.Version)
	if running == "" {
		return
	}
	latest, err := fetchLatestVersion(refreshFetchTimeout)
	if err != nil {
		return
	}
	latest = strings.TrimSpace(latest)
	if latest == "" {
		return
	}
	home, err := cliconfig.ResolveHomeDir()
	if err != nil {
		return
	}
	_ = SaveCache(home, &Cache{
		Current:   running,
		Latest:    latest,
		CheckedAt: time.Now(),
		HasUpdate: compareVersions(running, latest) < 0,
	})
}

// CheckNow performs a synchronous check for explicit commands
// (`version --check`, `update`). It returns a fresh cached result when
// available, otherwise fetches from the registry within the timeout and
// persists the cache. Returns nil only when update checks are disabled.
func CheckNow(timeout time.Duration) *Result {
	if shouldSkip() {
		return nil
	}
	running := strings.TrimSpace(build.Version)
	if running == "" {
		return nil
	}
	if home, err := cliconfig.ResolveHomeDir(); err == nil {
		if r := loadCachedResult(home, running); r != nil {
			return r
		}
	}
	latest, err := fetchLatestVersion(timeout)
	if err != nil {
		return &Result{Current: running, Err: err}
	}
	latest = strings.TrimSpace(latest)
	if latest == "" {
		return &Result{Current: running, Err: fmt.Errorf("empty latest version")}
	}
	hasUpdate := compareVersions(running, latest) < 0
	if home, herr := cliconfig.ResolveHomeDir(); herr == nil {
		_ = SaveCache(home, &Cache{
			Current:   running,
			Latest:    latest,
			CheckedAt: time.Now(),
			HasUpdate: hasUpdate,
		})
	}
	return &Result{HasUpdate: hasUpdate, Current: running, Latest: latest}
}

// loadCachedResult returns a displayable result only when the cache is fresh
// and carries a real latest version. A debounce placeholder (empty Latest) is
// treated as "nothing to show".
func loadCachedResult(home, running string) *Result {
	cached, _ := LoadCache(home)
	if cached == nil || !IsCacheFresh(cached, running) {
		return nil
	}
	if strings.TrimSpace(cached.Latest) == "" {
		return nil
	}
	return &Result{
		HasUpdate: cached.HasUpdate,
		Current:   cached.Current,
		Latest:    cached.Latest,
	}
}

// claimRefresh debounces detached refreshes. It returns false when a cache
// entry for the running version is still within the TTL (a real result or a
// recent placeholder), meaning a refresh is unnecessary. Otherwise it writes a
// placeholder (CheckedAt=now, empty Latest) and returns true so the caller
// spawns exactly one refresh subprocess.
func claimRefresh(home, running string) bool {
	if cached, _ := LoadCache(home); cached != nil &&
		cached.Current == running && time.Since(cached.CheckedAt) <= CacheTTL {
		return false
	}
	_ = SaveCache(home, &Cache{Current: running, CheckedAt: time.Now()})
	return true
}

func GetResult() *Result {
	v := checkResult.Load()
	if v == nil {
		return nil
	}
	r, ok := v.(*Result)
	if !ok {
		return nil
	}
	return r
}

func isInternalRefresh() bool {
	return strings.TrimSpace(os.Getenv(envInternalRefresh)) == "1"
}

func shouldSkip() bool {
	if v := strings.TrimSpace(os.Getenv(EnvDisable)); v != "" && v != "0" && strings.ToLower(v) != "false" {
		return true
	}
	if v := strings.TrimSpace(os.Getenv(EnvCI)); v != "" && v != "0" && strings.ToLower(v) != "false" {
		return true
	}
	if strings.TrimSpace(build.Version) == "dev" {
		return true
	}
	return false
}

func fetchLatestVersion(timeout time.Duration) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, NpmRegistryURL, nil)
	if err != nil {
		return "", err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("registry status %d", resp.StatusCode)
	}
	var payload struct {
		Version string `json:"version"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return "", err
	}
	return payload.Version, nil
}

// compareVersions returns -1 if a<b, 0 if equal, 1 if a>b. Best-effort semver.
func compareVersions(a, b string) int {
	a = strings.TrimPrefix(strings.TrimSpace(a), "v")
	b = strings.TrimPrefix(strings.TrimSpace(b), "v")
	if a == b {
		return 0
	}
	pa := strings.SplitN(a, "-", 2)
	pb := strings.SplitN(b, "-", 2)
	ca := splitNums(pa[0])
	cb := splitNums(pb[0])
	for i := 0; i < 3; i++ {
		var va, vb int
		if i < len(ca) {
			va = ca[i]
		}
		if i < len(cb) {
			vb = cb[i]
		}
		if va < vb {
			return -1
		}
		if va > vb {
			return 1
		}
	}
	// Equal core. Pre-release < release.
	if len(pa) == 2 && len(pb) == 1 {
		return -1
	}
	if len(pa) == 1 && len(pb) == 2 {
		return 1
	}
	if len(pa) == 2 && len(pb) == 2 {
		return strings.Compare(pa[1], pb[1])
	}
	return 0
}

func splitNums(s string) []int {
	parts := strings.Split(s, ".")
	out := make([]int, 0, len(parts))
	for _, p := range parts {
		n := 0
		for _, r := range p {
			if r < '0' || r > '9' {
				break
			}
			n = n*10 + int(r-'0')
		}
		out = append(out, n)
	}
	return out
}
