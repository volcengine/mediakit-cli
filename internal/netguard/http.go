package netguard

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const maxRedirects = 10

// ValidateRemoteURL accepts HTTP and HTTPS URLs with a valid host. Private and
// reserved destinations are allowed because product runtimes may resolve
// customer-accessible media URLs through an internal network.
func ValidateRemoteURL(_ context.Context, rawURL string) (*url.URL, error) {
	parsed, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil {
		return nil, fmt.Errorf("invalid remote URL: %w", err)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return nil, fmt.Errorf("remote URL must use HTTP or HTTPS")
	}
	if parsed.Hostname() == "" {
		return nil, fmt.Errorf("remote URL host is required")
	}
	if parsed.User != nil {
		return nil, fmt.Errorf("remote URL must not contain user information")
	}
	return parsed, nil
}

// NewHTTPClient returns a client that revalidates every redirect. Environment
// proxies remain disabled so reachability checks reflect the resolved target.
func NewHTTPClient(timeout time.Duration) *http.Client {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.Proxy = nil
	return &http.Client{
		Timeout:   timeout,
		Transport: transport,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= maxRedirects {
				return fmt.Errorf("stopped after %d redirects", maxRedirects)
			}
			_, err := ValidateRemoteURL(req.Context(), req.URL.String())
			return err
		},
	}
}
