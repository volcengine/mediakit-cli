package auth

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"os"
	"strings"

	"mediakit-cli/internal/buildenv"
)

type Kind string

const (
	KindAPIKey Kind = "api_key"
)

// Context is immutable for one logical Cloud execution. The same value is
// reused for submit and poll requests; object-storage uploads do not consume it.
type Context struct {
	kind  Kind
	value string
	logID string
}

type Resolver struct {
	LookupEnv func(string) (string, bool)
}

func Resolve() (Context, error) {
	return Resolver{LookupEnv: os.LookupEnv}.Resolve()
}

func (r Resolver) Resolve() (Context, error) {
	lookup := r.LookupEnv
	if lookup == nil {
		lookup = os.LookupEnv
	}

	logID := lookupTrimmed(lookup, buildenv.CloudLogID)
	if err := validateHeaderValue(buildenv.CloudLogID, logID); err != nil {
		return Context{}, err
	}

	apiKey := lookupTrimmed(lookup, buildenv.CloudAPIKey)
	if apiKey == "" {
		return Context{}, fmt.Errorf(
			"未配置云端鉴权：请设置 %s",
			buildenv.CloudAPIKey,
		)
	}
	if err := validateHeaderValue(buildenv.CloudAPIKey, apiKey); err != nil {
		return Context{}, err
	}
	return Context{
		kind:  KindAPIKey,
		value: apiKey,
		logID: logID,
	}, nil
}

func (c Context) Apply(req *http.Request) {
	if req == nil {
		return
	}
	req.Header.Set("Authorization", "Bearer "+c.value)
	if c.logID != "" {
		req.Header.Set("x-tt-logid", c.logID)
	}
}

func (c Context) Kind() Kind {
	return c.kind
}

// UploadCacheScopeIdentity returns a non-secret bucket identity for upload cache.
func (c Context) UploadCacheScopeIdentity() string {
	sum := sha256.Sum256([]byte(c.value))
	return hex.EncodeToString(sum[:])
}

func lookupTrimmed(lookup func(string) (string, bool), name string) string {
	value, _ := lookup(name)
	return strings.TrimSpace(value)
}

func validateHeaderValue(name string, value string) error {
	if strings.ContainsAny(value, "\r\n\x00") {
		return fmt.Errorf("%s 包含不允许的控制字符", name)
	}
	return nil
}
