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
	// requestTimeout applies to JSON requests, but not to backups.
	requestTimeout = 30 * time.Second
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

// send sends a request to the public API, with `body` as JSON unless nil, and
// returns the response if it's successful. The caller must close its body.
func (c *client) send(ctx context.Context, method, path string, body any, accept string) (*http.Response, error) {
	var reqBody io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		reqBody = bytes.NewReader(data)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+publicApiPrefix+path, reqBody)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Accept", accept)
	req.Header.Set("User-Agent", "ugctl/"+version)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request to %s failed: %w", c.baseURL, err)
	}
	if resp.StatusCode < 300 {
		return resp, nil
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode < 400 {
		return nil, fmt.Errorf("%s redirected to %q; is it the URL of the admin host?",
			c.baseURL, resp.Header.Get("Location"))
	}
	data, _ := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes))
	var e api.PublicError
	if json.Unmarshal(data, &e) == nil && e.Error.Code != "" {
		return nil, &apiError{status: resp.StatusCode, code: e.Error.Code, message: e.Error.Message}
	}
	return nil, fmt.Errorf("unexpected response from %s: %s", c.baseURL, resp.Status)
}

// do sends a request to the public API. `body` is sent as JSON unless nil, and
// a successful response is decoded into `out` unless nil.
func (c *client) do(ctx context.Context, method, path string, body, out any) error {
	ctx, cancel := context.WithTimeout(ctx, requestTimeout)
	defer cancel()
	resp, err := c.send(ctx, method, path, body, "application/json")
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	data, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes))
	if err != nil {
		return fmt.Errorf("reading the response from %s failed: %w", c.baseURL, err)
	}
	if out != nil {
		if err := json.Unmarshal(data, out); err != nil {
			return fmt.Errorf("invalid response from %s: %w", c.baseURL, err)
		}
	}
	return nil
}

// downloadBackup writes a database backup to `w`. There's no overall timeout,
// as the database can be large.
func (c *client) downloadBackup(ctx context.Context, w io.Writer) (int64, error) {
	resp, err := c.send(ctx, http.MethodGet, "/backup", nil, "application/octet-stream")
	if err != nil {
		return 0, err
	}
	defer func() { _ = resp.Body.Close() }()
	if ct := resp.Header.Get("Content-Type"); ct != "application/octet-stream" {
		return 0, fmt.Errorf("unexpected response from %s: Content-Type %q", c.baseURL, ct)
	}
	// A body that is shorter than its Content-Length is reported as an error,
	// which is how the server signals a backup that failed midway.
	n, err := io.Copy(w, resp.Body)
	if err != nil {
		return n, fmt.Errorf("downloading the backup from %s failed: %w", c.baseURL, err)
	}
	return n, nil
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
