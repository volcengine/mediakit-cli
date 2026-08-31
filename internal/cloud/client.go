package cloud

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"mediakit-cli/internal/auth"
	"mediakit-cli/internal/surface"
)

const defaultEndpoint = "https://mediakit.cn-beijing.volces.com"

type Client struct {
	Endpoint   string
	Auth       auth.Context
	Runtime    string
	HTTPClient *http.Client
}

func NewClient(authContext auth.Context, endpoint string, runtime string) *Client {
	endpoint = strings.TrimSpace(endpoint)
	if endpoint == "" {
		endpoint = defaultEndpoint
	}
	return &Client{
		Endpoint: endpoint,
		Auth:     authContext,
		Runtime:  strings.TrimSpace(runtime),
		HTTPClient: &http.Client{
			Timeout: 10 * time.Minute,
		},
	}
}

func (c *Client) Call(apiName string, args map[string]any) (map[string]any, error) {
	method := ""
	path := ""
	if apiName == queryTaskCommand {
		method = http.MethodGet
		path = "/api/v1/tasks/{task_id}"
	} else {
		capability, ok := surface.Lookup(apiName)
		if !ok {
			return nil, fmt.Errorf("unknown api: %s", apiName)
		}
		method = capability.Method
		path = capability.Path
	}

	payload := cloneParams(args)
	for key, value := range payload {
		placeholder := "{" + key + "}"
		if strings.Contains(path, placeholder) {
			path = strings.Replace(path, placeholder, fmt.Sprint(value), 1)
			delete(payload, key)
		}
	}

	method = strings.ToUpper(method)
	switch method {
	case http.MethodGet, http.MethodDelete:
		req, err := c.newRequest(method, path, payload, nil)
		if err != nil {
			return nil, err
		}
		return c.do(req)
	default:
		req, err := c.newRequest(method, path, nil, payload)
		if err != nil {
			return nil, err
		}
		return c.do(req)
	}
}
