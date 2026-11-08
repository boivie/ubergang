package main

import (
	"boivie/ubergang/server/api"
	"boivie/ubergang/server/rest"
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"slices"
	"strings"
	"testing"
)

type testServer struct {
	url      string
	fixture  *rest.Fixture
	writeKey string
	readKey  string
}

// newTestServer serves the real API over HTTP, with a write and a read key.
func newTestServer(t *testing.T) *testServer {
	f := rest.CreateFixture(t)
	admin, _ := f.CreateAdmin("admin@example.com")
	write, rr := f.CreateApiKey(admin, "write", "write")
	if rr.Code != http.StatusOK {
		t.Fatalf("Creating API key: %d %s", rr.Code, rr.Body.String())
	}
	read, _ := f.CreateApiKey(admin, "read", "read")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.Host = "test.example.com"
		f.ServeHTTP(w, r)
	}))
	t.Cleanup(srv.Close)
	return &testServer{url: srv.URL, fixture: f, writeKey: write.Token, readKey: read.Token}
}

// isolate keeps tests from using the user's config and environment, and
// returns the path of the config file.
func isolate(t *testing.T) string {
	path := filepath.Join(t.TempDir(), "ugctl.yaml")
	t.Setenv(envConfig, path)
	t.Setenv(envURL, "")
	t.Setenv(envAPIKey, "")
	return path
}

// useServer isolates the test and points ugctl at `srv` with `key`.
func useServer(t *testing.T, srv *testServer, key string) {
	isolate(t)
	t.Setenv(envURL, srv.url)
	t.Setenv(envAPIKey, key)
}

type result struct {
	code   int
	stdout string
	stderr string
}

func ugctl(t *testing.T, stdin string, args ...string) result {
	t.Helper()
	var stdout, stderr bytes.Buffer
	code := run(context.Background(), args, strings.NewReader(stdin), &stdout, &stderr)
	return result{code, stdout.String(), stderr.String()}
}

// mustRun runs ugctl and fails the test unless it succeeds.
func mustRun(t *testing.T, stdin string, args ...string) string {
	t.Helper()
	res := ugctl(t, stdin, args...)
	if res.code != exitOK {
		t.Fatalf("ugctl %v: exit %d, stderr: %s", args, res.code, res.stderr)
	}
	return res.stdout
}

// expectExit runs ugctl and fails the test unless it exits with `code`.
func expectExit(t *testing.T, code int, args ...string) result {
	t.Helper()
	res := ugctl(t, "", args...)
	if res.code != code {
		t.Fatalf("ugctl %v: expected exit %d, got %d, stdout: %s, stderr: %s",
			args, code, res.code, res.stdout, res.stderr)
	}
	return res
}

func decode[T any](t *testing.T, s string) T {
	t.Helper()
	var v T
	if err := json.Unmarshal([]byte(s), &v); err != nil {
		t.Fatalf("Invalid JSON %q: %v", s, err)
	}
	return v
}

func TestMqttProfiles(t *testing.T) {
	srv := newTestServer(t)
	useServer(t, srv, srv.writeKey)

	out := mustRun(t, "", "mqtt", "profiles", "create", "sensors",
		"--publish", "sensors/$ID/#", "--publish", "rooms/$room/#")
	if !strings.Contains(out, "sensors/$ID/#") || !strings.Contains(out, "rooms/$room/#") {
		t.Errorf("Unexpected output: %s", out)
	}

	res := expectExit(t, exitConflict, "mqtt", "profiles", "create", "sensors")
	if !strings.Contains(res.stderr, "(conflict)") {
		t.Errorf("Unexpected error: %s", res.stderr)
	}

	list := decode[api.PublicListMqttProfilesResponse](t, mustRun(t, "", "-o", "json", "mqtt", "profiles", "list"))
	if len(list.Profiles) != 1 || len(list.Profiles[0].AllowPublish) != 2 {
		t.Errorf("Unexpected profiles: %+v", list)
	}

	updated := decode[api.PublicMqttProfile](t, mustRun(t, "",
		"mqtt", "profiles", "update", "sensors", "--clear-publish", "--subscribe", "commands/$ID", "-o", "json"))
	if len(updated.AllowPublish) != 0 || !slices.Equal(updated.AllowSubscribe, []string{"commands/$ID"}) {
		t.Errorf("Unexpected profile: %+v", updated)
	}

	out = mustRun(t, "", "mqtt", "profiles", "get", "sensors")
	if !regexp.MustCompile(`Publish:\s+-`).MatchString(out) || !strings.Contains(out, "commands/$ID") {
		t.Errorf("Unexpected output: %s", out)
	}

	res = expectExit(t, exitOK, "mqtt", "profiles", "rm", "sensors", "-o", "json")
	if res.stdout != "" {
		t.Errorf("Expected no output, got %q", res.stdout)
	}
	res = expectExit(t, exitNotFound, "mqtt", "profiles", "get", "sensors", "-o", "json")
	if e := decode[api.PublicError](t, res.stderr); e.Error.Code != "not_found" {
		t.Errorf("Unexpected error: %+v", e)
	}

	res = expectExit(t, exitOK, "mqtt", "profiles", "list")
	if res.stdout != "" || !strings.Contains(res.stderr, "No MQTT profiles") {
		t.Errorf("Unexpected output: %+v", res)
	}
}

func TestMqttClients(t *testing.T) {
	srv := newTestServer(t)
	useServer(t, srv, srv.writeKey)
	mustRun(t, "", "mqtt", "profiles", "create", "sensors", "--publish", "sensors/$ID/#")

	created := decode[api.PublicMqttClient](t, mustRun(t, "s3cret\n",
		"mqtt", "clients", "create", "dev-1", "--profile", "sensors", "--password-stdin",
		"--value", "room=kitchen", "--value", "floor=1", "-o", "json"))
	if created.Password != "s3cret" || created.Values["room"] != "kitchen" || created.Values["floor"] != "1" {
		t.Errorf("Unexpected client: %+v", created)
	}

	res := expectExit(t, exitError, "mqtt", "clients", "create", "dev-2", "--profile", "missing", "--generate-password")
	if !strings.Contains(res.stderr, "(profile_not_found)") {
		t.Errorf("Unexpected error: %s", res.stderr)
	}

	out := mustRun(t, "", "mqtt", "clients", "get", "dev-1")
	if !regexp.MustCompile(`Password:\s+s3cret`).MatchString(out) || !strings.Contains(out, "floor=1") {
		t.Errorf("Unexpected output: %s", out)
	}

	out = mustRun(t, "", "mqtt", "clients", "list")
	if !strings.Contains(out, "dev-1") || !strings.Contains(out, "floor=1, room=kitchen") {
		t.Errorf("Unexpected output: %s", out)
	}
	if strings.Contains(out, "s3cret") {
		t.Errorf("The table shouldn't include passwords: %s", out)
	}

	updated := decode[api.PublicMqttClient](t, mustRun(t, "",
		"mqtt", "clients", "update", "dev-1", "--generate-password", "--value", "room=hall", "-o", "json"))
	if !regexp.MustCompile(`^[0-9a-f]{32}$`).MatchString(updated.Password) {
		t.Errorf("Unexpected generated password: %q", updated.Password)
	}
	if len(updated.Values) != 1 || updated.Values["room"] != "hall" {
		t.Errorf("Expected --value to replace the values: %+v", updated.Values)
	}
	if !slices.Contains(srv.fixture.Mqtt.Disconnected, "dev-1") {
		t.Errorf("Expected the client to be disconnected")
	}

	updated = decode[api.PublicMqttClient](t, mustRun(t, "",
		"mqtt", "clients", "update", "dev-1", "--clear-values", "-o", "json"))
	if len(updated.Values) != 0 {
		t.Errorf("Expected no values: %+v", updated.Values)
	}

	expectExit(t, exitConflict, "mqtt", "profiles", "delete", "sensors")
	res = expectExit(t, exitOK, "mqtt", "clients", "delete", "dev-1")
	if res.stdout != "Deleted MQTT client dev-1.\n" {
		t.Errorf("Unexpected output: %q", res.stdout)
	}
	expectExit(t, exitNotFound, "mqtt", "clients", "delete", "dev-1")
	expectExit(t, exitOK, "mqtt", "profiles", "delete", "sensors")
}

func TestIdsAreEscaped(t *testing.T) {
	srv := newTestServer(t)
	useServer(t, srv, srv.writeKey)
	mustRun(t, "", "mqtt", "profiles", "create", "p")
	mustRun(t, "", "mqtt", "clients", "create", "a b?c#d%2F", "--profile", "p", "--password", "x")

	cl := decode[api.PublicMqttClient](t, mustRun(t, "", "mqtt", "clients", "get", "a b?c#d%2F", "-o", "json"))
	if cl.Id != "a b?c#d%2F" {
		t.Errorf("Unexpected client: %+v", cl)
	}
}

func TestReadKey(t *testing.T) {
	srv := newTestServer(t)
	useServer(t, srv, srv.writeKey)
	mustRun(t, "", "mqtt", "profiles", "create", "p")
	mustRun(t, "", "mqtt", "clients", "create", "dev-1", "--profile", "p", "--password", "s3cret")

	t.Setenv(envAPIKey, srv.readKey)
	out := mustRun(t, "", "mqtt", "clients", "get", "dev-1")
	if strings.Contains(out, "s3cret") || strings.Contains(out, "Password") {
		t.Errorf("A read key shouldn't see passwords: %s", out)
	}
	res := expectExit(t, exitUnauthorized, "mqtt", "clients", "delete", "dev-1")
	if !strings.Contains(res.stderr, "(insufficient_scope)") {
		t.Errorf("Unexpected error: %s", res.stderr)
	}
}

func TestInvalidKey(t *testing.T) {
	srv := newTestServer(t)
	useServer(t, srv, "ug_bogus")
	res := expectExit(t, exitUnauthorized, "mqtt", "profiles", "list", "-o", "json")
	if e := decode[api.PublicError](t, res.stderr); e.Error.Code != "unauthorized" {
		t.Errorf("Unexpected error: %+v", e)
	}
}

func TestUsageErrors(t *testing.T) {
	srv := newTestServer(t)
	useServer(t, srv, srv.writeKey)
	for _, args := range [][]string{
		{"nope"},
		{"mqtt", "nope"},
		{"mqtt", "clients", "get"},
		{"mqtt", "clients", "get", "a", "b"},
		{"mqtt", "clients", "get", "a/b"},
		{"mqtt", "clients", "get", ""},
		{"mqtt", "clients", "list", "--bogus"},
		{"mqtt", "clients", "list", "-o", "yaml"},
		{"mqtt", "clients", "create", "dev-1", "--password", "x"},
		{"mqtt", "clients", "create", "dev-1", "--profile", "p"},
		{"mqtt", "clients", "create", "dev-1", "--profile", "p", "--password", ""},
		{"mqtt", "clients", "create", "dev-1", "--profile", "p", "--password", "x", "--generate-password"},
		{"mqtt", "clients", "create", "dev-1", "--profile", "p", "--password", "x", "--value", "novalue"},
		{"mqtt", "clients", "create", "dev-1", "--profile", "p", "--password", "x", "--value", "=v"},
		{"mqtt", "clients", "create", "dev-1", "--profile", "p", "--password", "x", "--value", "k=1", "--value", "k=2"},
		{"mqtt", "clients", "update", "dev-1"},
		{"mqtt", "clients", "update", "dev-1", "--value", "k=v", "--clear-values"},
		{"mqtt", "profiles", "update", "p"},
		{"mqtt", "profiles", "update", "p", "--publish", "t", "--clear-publish"},
	} {
		res := expectExit(t, exitUsage, args...)
		if !strings.HasPrefix(res.stderr, "Error: ") {
			t.Errorf("ugctl %v: unexpected stderr: %s", args, res.stderr)
		}
	}
	res := expectExit(t, exitUsage, "mqtt", "clients", "get", "-o", "json")
	if e := decode[api.PublicError](t, res.stderr); e.Error.Code != "usage" {
		t.Errorf("Unexpected error: %+v", e)
	}
}

func TestNotConfigured(t *testing.T) {
	isolate(t)
	res := expectExit(t, exitError, "mqtt", "profiles", "list")
	if !strings.Contains(res.stderr, "ugctl login") || !strings.Contains(res.stderr, envURL) {
		t.Errorf("Unexpected error: %s", res.stderr)
	}

	t.Setenv(envURL, "admin.example.com")
	res = expectExit(t, exitError, "mqtt", "profiles", "list")
	if !strings.Contains(res.stderr, "no API key for https://admin.example.com") {
		t.Errorf("Unexpected error: %s", res.stderr)
	}
}

func TestLoginAndLogout(t *testing.T) {
	srv := newTestServer(t)
	configPath := isolate(t)

	res := expectExit(t, exitError, "login", srv.url)
	if !strings.Contains(res.stderr, "no API key given") {
		t.Errorf("Unexpected error: %s", res.stderr)
	}
	if res := ugctl(t, "ug_bogus\n", "login", srv.url); res.code != exitUnauthorized {
		t.Errorf("Expected exit %d for an invalid key, got %d: %s", exitUnauthorized, res.code, res.stderr)
	}
	if _, err := os.Stat(configPath); err == nil {
		t.Errorf("A failed login shouldn't write the config")
	}

	out := mustRun(t, "  "+srv.writeKey+"\n", "login", srv.url+"/")
	if !strings.Contains(out, "Logged in to "+srv.url) {
		t.Errorf("Unexpected output: %s", out)
	}
	info, err := os.Stat(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if runtime.GOOS != "windows" && info.Mode().Perm() != 0o600 {
		t.Errorf("Expected the config to only be readable by the user, got %v", info.Mode().Perm())
	}

	mustRun(t, "", "mqtt", "profiles", "list")

	// The saved key must not be sent to another server.
	res = expectExit(t, exitError, "--url", "https://other.example.com", "mqtt", "profiles", "list")
	if !strings.Contains(res.stderr, "no API key for https://other.example.com") {
		t.Errorf("Unexpected error: %s", res.stderr)
	}

	// The environment takes precedence over the config.
	t.Setenv(envAPIKey, "ug_bogus")
	expectExit(t, exitUnauthorized, "mqtt", "profiles", "list")
	t.Setenv(envAPIKey, "")

	// Logging in again without a URL reuses the saved one.
	mustRun(t, srv.readKey, "login")

	out = mustRun(t, "", "logout")
	if !strings.Contains(out, "Removed") {
		t.Errorf("Unexpected output: %s", out)
	}
	if _, err := os.Stat(configPath); !os.IsNotExist(err) {
		t.Errorf("Expected the config to be removed")
	}
	out = mustRun(t, "", "logout")
	if !strings.Contains(out, "Not logged in") {
		t.Errorf("Unexpected output: %s", out)
	}
	expectExit(t, exitError, "mqtt", "profiles", "list")
}

func TestRedirectsAreNotFollowed(t *testing.T) {
	var gotAuth []string
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = append(gotAuth, r.Header.Get("Authorization"))
	}))
	defer target.Close()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, target.URL+"/signin", http.StatusFound)
	}))
	defer srv.Close()

	isolate(t)
	t.Setenv(envURL, srv.URL)
	t.Setenv(envAPIKey, "ug_secret")
	res := expectExit(t, exitError, "mqtt", "profiles", "list")
	if !strings.Contains(res.stderr, "admin host") {
		t.Errorf("Unexpected error: %s", res.stderr)
	}
	if len(gotAuth) != 0 {
		t.Errorf("The redirect was followed")
	}
}

func TestUnexpectedErrorResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "<html>Bad gateway</html>", http.StatusBadGateway)
	}))
	defer srv.Close()

	isolate(t)
	t.Setenv(envURL, srv.URL)
	t.Setenv(envAPIKey, "ug_secret")
	res := expectExit(t, exitError, "mqtt", "profiles", "list")
	if !strings.Contains(res.stderr, "502 Bad Gateway") {
		t.Errorf("Unexpected error: %s", res.stderr)
	}
}

func TestNormalizeURL(t *testing.T) {
	for _, tc := range []struct {
		in, want string
	}{
		{"admin.example.com", "https://admin.example.com"},
		{" https://admin.example.com/ ", "https://admin.example.com"},
		{"admin.example.com:10443", "https://admin.example.com:10443"},
		{"http://localhost:8080", "http://localhost:8080"},
		{"http://127.0.0.1:8080", "http://127.0.0.1:8080"},
		{"http://[::1]:8080", "http://[::1]:8080"},
	} {
		got, err := normalizeURL(tc.in)
		if err != nil || got != tc.want {
			t.Errorf("normalizeURL(%q) = %q, %v; want %q", tc.in, got, err, tc.want)
		}
	}
	for _, in := range []string{
		"",
		"http://admin.example.com",
		"ftp://admin.example.com",
		"https://admin.example.com/api/v1",
		"https://admin.example.com?x=1",
		"https://user:pass@admin.example.com",
		"https://",
	} {
		if got, err := normalizeURL(in); err == nil {
			t.Errorf("normalizeURL(%q) = %q; want an error", in, got)
		}
	}
}

func TestVersion(t *testing.T) {
	isolate(t)
	out := mustRun(t, "", "version")
	if !strings.HasPrefix(out, "ugctl dev") {
		t.Errorf("Unexpected output: %s", out)
	}
	v := decode[map[string]string](t, mustRun(t, "", "version", "-o", "json"))
	if v["version"] != "dev" {
		t.Errorf("Unexpected version: %+v", v)
	}
}
