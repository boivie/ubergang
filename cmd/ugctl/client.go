package main

import (
	"boivie/ubergang/server/api"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	publicApiPrefix  = "/api/v1"
	maxResponseBytes = 16 << 20
)

// apiError is an error response from the public API.
type apiError struct {
	status  int
	code    string
	message string
}

func (e *apiError) Error() string { return e.message }

// client calls the public API, see docs/public-api.md.
type client struct {
	baseURL string
	apiKey  string
	http    *http.Client
}

func newClient(baseURL, apiKey string) *client {
	return &client{
		baseURL: baseURL,
		apiKey:  apiKey,
		http: &http.Client{
			Timeout: 30 * time.Second,
			// Never follow redirects, as the API doesn't use them. A redirect
			// means that the URL isn't the admin host, and following it
			// could send the API key elsewhere.
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
	}
}

// normalizeURL validates the URL of a server's admin host and returns it as
// "scheme://host[:port]". The scheme defaults to https, and plain http is only
// allowed to loopback addresses so that API keys aren't sent in the clear.
func normalizeURL(raw string) (string, error) {
	s := strings.TrimSpace(raw)
	if s == "" {
		return "", errors.New("the URL is empty")
	}
	if !strings.Contains(s, "://") {
		s = "https://" + s
	}
	u, err := url.Parse(s)
	if err != nil {
		return "", fmt.Errorf("invalid URL %q: %w", raw, err)
	}
	switch {
	case u.Scheme != "https" && u.Scheme != "http":
		return "", fmt.Errorf("invalid URL %q: the scheme must be https", raw)
	case u.Host == "" || u.Hostname() == "":
		return "", fmt.Errorf("invalid URL %q: no host", raw)
	case u.User != nil:
		return "", fmt.Errorf("invalid URL %q: credentials aren't allowed in the URL", raw)
	case strings.TrimSuffix(u.Path, "/") != "" || u.RawQuery != "" || u.Fragment != "":
		return "", fmt.Errorf("invalid URL %q: use only the admin host, such as https://admin.example.com", raw)
	case u.Scheme == "http" && !isLoopback(u.Hostname()):
		return "", fmt.Errorf("invalid URL %q: refusing to send the API key over plain http; use https", raw)
	}
	return u.Scheme + "://" + u.Host, nil
}

func isLoopback(host string) bool {
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

// do sends a request to the public API. `body` is sent as JSON unless nil, and
// a successful response is decoded into `out` unless nil.
func (c *client) do(ctx context.Context, method, path string, body, out any) error {
	var reqBody io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return err
		}
		reqBody = bytes.NewReader(data)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+publicApiPrefix+path, reqBody)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "ugctl/"+version)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("request to %s failed: %w", c.baseURL, err)
	}
	defer func() { _ = resp.Body.Close() }()
	data, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes))
	if err != nil {
		return fmt.Errorf("reading the response from %s failed: %w", c.baseURL, err)
	}

	switch {
	case resp.StatusCode >= 300 && resp.StatusCode < 400:
		return fmt.Errorf("%s redirected to %q; is it the URL of the admin host?",
			c.baseURL, resp.Header.Get("Location"))
	case resp.StatusCode >= 400:
		var e api.PublicError
		if json.Unmarshal(data, &e) == nil && e.Error.Code != "" {
			return &apiError{status: resp.StatusCode, code: e.Error.Code, message: e.Error.Message}
		}
		return fmt.Errorf("unexpected response from %s: %s", c.baseURL, resp.Status)
	}
	if out != nil {
		if err := json.Unmarshal(data, out); err != nil {
			return fmt.Errorf("invalid response from %s: %w", c.baseURL, err)
		}
	}
	return nil
}

func resourcePath(collection, id string) string {
	return collection + "/" + url.PathEscape(id)
}

// call sends a request and decodes the response as a T.
func call[T any](ctx context.Context, c *client, method, path string, body any) (*T, error) {
	var res T
	if err := c.do(ctx, method, path, body, &res); err != nil {
		return nil, err
	}
	return &res, nil
}

func (c *client) listMqttProfiles(ctx context.Context) (*api.PublicListMqttProfilesResponse, error) {
	return call[api.PublicListMqttProfilesResponse](ctx, c, http.MethodGet, "/mqtt/profiles", nil)
}

func (c *client) getMqttProfile(ctx context.Context, id string) (*api.PublicMqttProfile, error) {
	return call[api.PublicMqttProfile](ctx, c, http.MethodGet, resourcePath("/mqtt/profiles", id), nil)
}

func (c *client) createMqttProfile(ctx context.Context, req *api.PublicCreateMqttProfileRequest) (*api.PublicMqttProfile, error) {
	return call[api.PublicMqttProfile](ctx, c, http.MethodPost, "/mqtt/profiles", req)
}

func (c *client) updateMqttProfile(ctx context.Context, id string, req *api.PublicUpdateMqttProfileRequest) (*api.PublicMqttProfile, error) {
	return call[api.PublicMqttProfile](ctx, c, http.MethodPatch, resourcePath("/mqtt/profiles", id), req)
}

func (c *client) deleteMqttProfile(ctx context.Context, id string) error {
	return c.do(ctx, http.MethodDelete, resourcePath("/mqtt/profiles", id), nil, nil)
}

func (c *client) listMqttClients(ctx context.Context) (*api.PublicListMqttClientsResponse, error) {
	return call[api.PublicListMqttClientsResponse](ctx, c, http.MethodGet, "/mqtt/clients", nil)
}

func (c *client) getMqttClient(ctx context.Context, id string) (*api.PublicMqttClient, error) {
	return call[api.PublicMqttClient](ctx, c, http.MethodGet, resourcePath("/mqtt/clients", id), nil)
}

func (c *client) createMqttClient(ctx context.Context, req *api.PublicCreateMqttClientRequest) (*api.PublicMqttClient, error) {
	return call[api.PublicMqttClient](ctx, c, http.MethodPost, "/mqtt/clients", req)
}

func (c *client) updateMqttClient(ctx context.Context, id string, req *api.PublicUpdateMqttClientRequest) (*api.PublicMqttClient, error) {
	return call[api.PublicMqttClient](ctx, c, http.MethodPatch, resourcePath("/mqtt/clients", id), req)
}

func (c *client) deleteMqttClient(ctx context.Context, id string) error {
	return c.do(ctx, http.MethodDelete, resourcePath("/mqtt/clients", id), nil, nil)
}
