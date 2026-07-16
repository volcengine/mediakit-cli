package cloud

import (
	"fmt"
	"net/http"
	"strings"
	"time"
)

const defaultEndpoint = "https://mediakit.cn-beijing.volces.com/"

type Client struct {
	Endpoint   string
	APIKey     string
	Surface    string
	Runtime    string
	HTTPClient *http.Client
}

func NewClient(apiKey string, endpoint string, surface string, runtime string) *Client {
	endpoint = strings.TrimSpace(endpoint)
	if endpoint == "" {
		endpoint = defaultEndpoint
	}
	return &Client{
		Endpoint: endpoint,
		APIKey:   strings.TrimSpace(apiKey),
		Surface:  strings.TrimSpace(surface),
		Runtime:  strings.TrimSpace(runtime),
		HTTPClient: &http.Client{
			Timeout: 10 * time.Minute,
		},
	}
}

func (c *Client) Call(apiName string, args map[string]any) (map[string]any, error) {
	api, ok := apiInfoRegistry[apiName]
	if !ok {
		return nil, fmt.Errorf("unknown api: %s", apiName)
	}

	payload := cloneParams(args)
	path := api.Path
	for key, value := range payload {
		placeholder := "{" + key + "}"
		if strings.Contains(path, placeholder) {
			path = strings.Replace(path, placeholder, fmt.Sprint(value), 1)
			delete(payload, key)
		}
	}

	method := strings.ToUpper(api.Method)
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
