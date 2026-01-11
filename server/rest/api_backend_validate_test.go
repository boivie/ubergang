package rest

import (
	"boivie/ubergang/server/api"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateBackend(t *testing.T) {
	t.Run("returns error for invalid upstream URL", func(t *testing.T) {
		f := CreateFixture(t)
		cookie, _ := f.CreateAdmin("test@example.com")

		UpstreamUrl := "://invalid-url"

		resp := &api.ApiValidateBackendResponse{}
		rr := f.request("POST", "/api/backend/validate?upstream_url="+UpstreamUrl, nil, cookie, resp)

		require.Equal(t, http.StatusOK, rr.Code)
		assert.False(t, resp.Reachable)
		assert.Contains(t, resp.Error, "Invalid upstream URL")
	})

	t.Run("validates reachable HTTP backend", func(t *testing.T) {
		f := CreateFixture(t)
		cookie, _ := f.CreateAdmin("test@example.com")

		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		defer ts.Close()

		resp := &api.ApiValidateBackendResponse{}
		// ts.URL looks like "http://127.0.0.1:12345"
		rr := f.request("POST", "/api/backend/validate?upstream_url="+url.QueryEscape(ts.URL), nil, cookie, resp)

		require.Equal(t, http.StatusOK, rr.Code)
		assert.True(t, resp.Reachable)
		assert.False(t, resp.TLS)
		assert.False(t, resp.ValidCertificate)
		assert.Empty(t, resp.Error)
	})

	t.Run("validates reachable HTTPS backend with self-signed cert", func(t *testing.T) {
		f := CreateFixture(t)
		cookie, _ := f.CreateAdmin("test@example.com")

		ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		defer ts.Close()

		resp := &api.ApiValidateBackendResponse{}
		// ts.URL looks like "https://127.0.0.1:12345"
		rr := f.request("POST", "/api/backend/validate?upstream_url="+url.QueryEscape(ts.URL), nil, cookie, resp)

		require.Equal(t, http.StatusOK, rr.Code)
		assert.True(t, resp.Reachable)
		assert.True(t, resp.TLS)
		// Should be false because httptest uses a self-signed cert not in system roots
		assert.False(t, resp.ValidCertificate)
		assert.Empty(t, resp.Error)
		assert.NotEmpty(t, resp.Certificates)
	})
}
