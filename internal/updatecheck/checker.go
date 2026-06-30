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

	cliconfig "mediakit-cli/internal/config"
	"mediakit-cli/internal/build"
)

const (
	NpmRegistryURL = "https://registry.npmjs.org/@volcengine/mediakit-cli/latest"
	PackageName    = "@volcengine/mediakit-cli"
	EnvDisable     = "MEDIAKIT_DISABLE_UPDATE_CHECK"
	EnvCI          = "CI"
)

type Result struct {
	HasUpdate bool
	Current   string
	Latest    string
	Err       error
}

var checkResult atomic.Value // *Result

func StartAsync() {
	if shouldSkip() {
		return
	}
	if r := loadCachedResult(); r != nil {
		checkResult.Store(r)
		return
	}
	go func() {
		r := runCheck()
		if r != nil {
			checkResult.Store(r)
		}
	}()
}

func loadCachedResult() *Result {
	running := strings.TrimSpace(build.Version)
	if running == "" {
		return nil
	}
	home, err := cliconfig.ResolveHomeDir()
	if err != nil {
		return nil
	}
	cached, _ := LoadCache(home)
	if cached == nil || !IsCacheFresh(cached, running) {
		return nil
	}
	return &Result{
		HasUpdate: cached.HasUpdate,
		Current:   cached.Current,
		Latest:    cached.Latest,
	}
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

func WaitForResult(timeout time.Duration) *Result {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if r := GetResult(); r != nil {
			return r
		}
		time.Sleep(10 * time.Millisecond)
	}
	return GetResult()
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

func runCheck() *Result {
	running := strings.TrimSpace(build.Version)
	if running == "" {
		return nil
	}

	home, err := cliconfig.ResolveHomeDir()
	if err == nil {
		if cached, _ := LoadCache(home); cached != nil && IsCacheFresh(cached, running) {
			return &Result{
				HasUpdate: cached.HasUpdate,
				Current:   cached.Current,
				Latest:    cached.Latest,
			}
		}
	}

	latest, err := fetchLatestVersion()
	if err != nil {
		return &Result{Current: running, Err: err}
	}
	latest = strings.TrimSpace(latest)
	if latest == "" {
		return &Result{Current: running, Err: fmt.Errorf("empty latest version")}
	}

	hasUpdate := compareVersions(running, latest) < 0
	cache := &Cache{
		Current:   running,
		Latest:    latest,
		CheckedAt: time.Now(),
		HasUpdate: hasUpdate,
	}
	if home, herr := cliconfig.ResolveHomeDir(); herr == nil {
		_ = SaveCache(home, cache)
	}

	return &Result{
		HasUpdate: hasUpdate,
		Current:   running,
		Latest:    latest,
	}
}

func fetchLatestVersion() (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 1500*time.Millisecond)
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
