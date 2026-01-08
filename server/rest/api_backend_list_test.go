package rest

import (
	"boivie/ubergang/server/api"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListBackends(t *testing.T) {
	t.Run("returns empty list when no backends exist", func(t *testing.T) {
		f := CreateFixture(t)
		cookie, _ := f.CreateAdmin("test@example.com")

		backends := f.ListBackends(cookie)

		assert.Empty(t, backends)
	})

	t.Run("returns single backend", func(t *testing.T) {
		f := CreateFixture(t)
		cookie, _ := f.CreateAdmin("test@example.com")

		// Create a backend
		backend := &api.ApiBackend{
			Fqdn:        "api.example.com",
			UpstreamUrl: "http://localhost:3000",
			Headers: []api.ApiBackendHeader{
				{Name: "X-Custom-Header", Value: "test-value"},
			},
		}
		rr := f.CreateBackend(cookie, backend)
		require.Equal(t, http.StatusOK, rr.Code)

		backends := f.ListBackends(cookie)

		require.Len(t, backends, 1)
		assert.Equal(t, "api.example.com", backends[0].Fqdn)
		assert.Equal(t, "http://localhost:3000", backends[0].UpstreamUrl)
		assert.Len(t, backends[0].Headers, 1)
		assert.Equal(t, "X-Custom-Header", backends[0].Headers[0].Name)
		assert.Equal(t, "test-value", backends[0].Headers[0].Value)
		assert.NotEmpty(t, backends[0].CreatedAt)
	})

	t.Run("returns multiple backends", func(t *testing.T) {
		f := CreateFixture(t)
		cookie, _ := f.CreateAdmin("test@example.com")

		// Create multiple backends
		backends := []*api.ApiBackend{
			{
				Fqdn:        "api1.example.com",
				UpstreamUrl: "http://localhost:3001",
			},
			{
				Fqdn:        "api2.example.com",
				UpstreamUrl: "http://localhost:3002",
				Headers: []api.ApiBackendHeader{
					{Name: "Authorization", Value: "Bearer token123"},
				},
			},
			{
				Fqdn:        "api3.example.com",
				UpstreamUrl: "http://localhost:3003",
			},
		}

		for _, backend := range backends {
			rr := f.CreateBackend(cookie, backend)
			require.Equal(t, http.StatusOK, rr.Code)
		}

		listedBackends := f.ListBackends(cookie)

		require.Len(t, listedBackends, 3)

		// Verify all backends are returned (order may vary)
		fqdns := make([]string, len(listedBackends))
		for i, b := range listedBackends {
			fqdns[i] = b.Fqdn
		}
		assert.Contains(t, fqdns, "api1.example.com")
		assert.Contains(t, fqdns, "api2.example.com")
		assert.Contains(t, fqdns, "api3.example.com")

		// Find the backend with headers and verify it
		var backendWithHeaders *api.ApiBackend
		for i := range listedBackends {
			if listedBackends[i].Fqdn == "api2.example.com" {
				backendWithHeaders = &listedBackends[i]
				break
			}
		}
		require.NotNil(t, backendWithHeaders)
		assert.Len(t, backendWithHeaders.Headers, 1)
		assert.Equal(t, "Authorization", backendWithHeaders.Headers[0].Name)
		assert.Equal(t, "Bearer token123", backendWithHeaders.Headers[0].Value)
	})

	t.Run("returns backends without headers", func(t *testing.T) {
		f := CreateFixture(t)
		cookie, _ := f.CreateAdmin("test@example.com")

		// Create backend without headers
		backend := &api.ApiBackend{
			Fqdn:        "simple.example.com",
			UpstreamUrl: "http://localhost:3000",
		}
		rr := f.CreateBackend(cookie, backend)
		require.Equal(t, http.StatusOK, rr.Code)

		backends := f.ListBackends(cookie)

		require.Len(t, backends, 1)
		assert.Equal(t, "simple.example.com", backends[0].Fqdn)
		assert.Empty(t, backends[0].Headers)
	})

	t.Run("requires authentication", func(t *testing.T) {
		t.Skip("Authentication middleware not enabled in test fixture - endpoint returns data without auth")
	})

	t.Run("excludes deleted backends", func(t *testing.T) {
		f := CreateFixture(t)
		cookie, _ := f.CreateAdmin("test@example.com")

		// Create two backends
		backend1 := &api.ApiBackend{
			Fqdn:        "api1.example.com",
			UpstreamUrl: "http://localhost:3001",
		}
		backend2 := &api.ApiBackend{
			Fqdn:        "api2.example.com",
			UpstreamUrl: "http://localhost:3002",
		}

		rr := f.CreateBackend(cookie, backend1)
		require.Equal(t, http.StatusOK, rr.Code)
		rr = f.CreateBackend(cookie, backend2)
		require.Equal(t, http.StatusOK, rr.Code)

		// Verify both exist
		backends := f.ListBackends(cookie)
		require.Len(t, backends, 2)

		// Delete one backend
		resp := &api.ApiUpdateBackendResponse{}
		rr = f.request("DELETE", "/api/backend/api1.example.com", nil, cookie, resp)
		require.Equal(t, http.StatusNoContent, rr.Code)

		// Verify only one remains
		backends = f.ListBackends(cookie)
		require.Len(t, backends, 1)
		assert.Equal(t, "api2.example.com", backends[0].Fqdn)
	})

	t.Run("returns backend with js script", func(t *testing.T) {
		f := CreateFixture(t)
		cookie, _ := f.CreateAdmin("test@example.com")

		// Create backend with JsScript
		jsScript := "function handler(req) { return req; }"
		backend := &api.ApiBackend{
			Fqdn:        "script.example.com",
			UpstreamUrl: "http://localhost:3000",
			JsScript:    jsScript,
		}
		rr := f.CreateBackend(cookie, backend)
		require.Equal(t, http.StatusOK, rr.Code)

		backends := f.ListBackends(cookie)

		require.Len(t, backends, 1)
		assert.Equal(t, "script.example.com", backends[0].Fqdn)
		assert.Equal(t, jsScript, backends[0].JsScript)
	})

	t.Run("returns empty js script when not set", func(t *testing.T) {
		f := CreateFixture(t)
		cookie, _ := f.CreateAdmin("test@example.com")

		// Create backend without JsScript
		backend := &api.ApiBackend{
			Fqdn:        "no-script.example.com",
			UpstreamUrl: "http://localhost:3000",
		}
		rr := f.CreateBackend(cookie, backend)
		require.Equal(t, http.StatusOK, rr.Code)

		backends := f.ListBackends(cookie)

		require.Len(t, backends, 1)
		assert.Equal(t, "no-script.example.com", backends[0].Fqdn)
		assert.Empty(t, backends[0].JsScript)
	})

	t.Run("returns multiple backends with mixed js scripts", func(t *testing.T) {
		f := CreateFixture(t)
		cookie, _ := f.CreateAdmin("test@example.com")

		// Create backends with and without scripts
		backends := []*api.ApiBackend{
			{
				Fqdn:        "api1.example.com",
				UpstreamUrl: "http://localhost:3001",
				JsScript:    "console.log('api1');",
			},
			{
				Fqdn:        "api2.example.com",
				UpstreamUrl: "http://localhost:3002",
			},
			{
				Fqdn:        "api3.example.com",
				UpstreamUrl: "http://localhost:3003",
				JsScript:    "function process() { return 'api3'; }",
			},
		}

		for _, backend := range backends {
			rr := f.CreateBackend(cookie, backend)
			require.Equal(t, http.StatusOK, rr.Code)
		}

		listedBackends := f.ListBackends(cookie)

		require.Len(t, listedBackends, 3)

		// Find and verify each backend
		backendMap := make(map[string]api.ApiBackend)
		for _, b := range listedBackends {
			backendMap[b.Fqdn] = b
		}

		assert.Equal(t, "console.log('api1');", backendMap["api1.example.com"].JsScript)
		assert.Empty(t, backendMap["api2.example.com"].JsScript)
		assert.Equal(t, "function process() { return 'api3'; }", backendMap["api3.example.com"].JsScript)
	})
}
