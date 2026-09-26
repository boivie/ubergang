package rest

import (
	"boivie/ubergang/server/api"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBackendAccessRules(t *testing.T) {
	rules := []api.ApiAccessRule{
		{Predicate: `request.path == "/health"`, Action: "PUBLIC"},
		{Predicate: `request.path.startsWith("/admin")`, Action: "DENY"},
		{Predicate: `true`, Action: "AUTHENTICATED"},
	}

	t.Run("create with rules", func(t *testing.T) {
		f, cookie := setupBackendTest(t)
		rr := f.CreateBackend(cookie, &api.ApiBackend{
			Fqdn:        "test.example.com",
			UpstreamUrl: "http://localhost:8080",
			AccessRules: rules,
		})
		require.Equal(t, http.StatusOK, rr.Code, rr.Body.String())

		backends := f.ListBackends(cookie)
		require.Len(t, backends, 1)
		assert.Equal(t, rules, backends[0].AccessRules)
	})

	t.Run("no rules lists as empty", func(t *testing.T) {
		f, cookie := setupBackendTest(t)
		rr := f.CreateBackend(cookie, &api.ApiBackend{
			Fqdn:        "test.example.com",
			UpstreamUrl: "http://localhost:8080",
		})
		require.Equal(t, http.StatusOK, rr.Code, rr.Body.String())

		backends := f.ListBackends(cookie)
		require.Len(t, backends, 1)
		assert.NotNil(t, backends[0].AccessRules)
		assert.Empty(t, backends[0].AccessRules)
	})

	t.Run("update without rules keeps them", func(t *testing.T) {
		f, cookie := setupBackendTest(t)
		rr := f.CreateBackend(cookie, &api.ApiBackend{
			Fqdn:        "test.example.com",
			UpstreamUrl: "http://localhost:8080",
			AccessRules: rules,
		})
		require.Equal(t, http.StatusOK, rr.Code, rr.Body.String())

		updatedURL := "http://localhost:9090"
		rr = f.request("POST", "/api/backend/test.example.com",
			&api.ApiUpdateBackendRequest{UpstreamUrl: &updatedURL}, cookie, &api.ApiUpdateBackendResponse{})
		require.Equal(t, http.StatusOK, rr.Code, rr.Body.String())

		backends := f.ListBackends(cookie)
		require.Len(t, backends, 1)
		assert.Equal(t, rules, backends[0].AccessRules)
	})

	t.Run("update with empty list clears them", func(t *testing.T) {
		f, cookie := setupBackendTest(t)
		rr := f.CreateBackend(cookie, &api.ApiBackend{
			Fqdn:        "test.example.com",
			UpstreamUrl: "http://localhost:8080",
			AccessRules: rules,
		})
		require.Equal(t, http.StatusOK, rr.Code, rr.Body.String())

		empty := []api.ApiAccessRule{}
		rr = f.request("POST", "/api/backend/test.example.com",
			&api.ApiUpdateBackendRequest{AccessRules: &empty}, cookie, &api.ApiUpdateBackendResponse{})
		require.Equal(t, http.StatusOK, rr.Code, rr.Body.String())

		backends := f.ListBackends(cookie)
		require.Len(t, backends, 1)
		assert.Empty(t, backends[0].AccessRules)
	})

	invalid := []struct {
		name string
		rule api.ApiAccessRule
	}{
		{"invalid action", api.ApiAccessRule{Predicate: `true`, Action: "ALLOW"}},
		{"missing action", api.ApiAccessRule{Predicate: `true`}},
		{"syntax error", api.ApiAccessRule{Predicate: `request.path ==`, Action: "PUBLIC"}},
		{"unknown field", api.ApiAccessRule{Predicate: `request.pth == "/"`, Action: "PUBLIC"}},
		{"not a bool", api.ApiAccessRule{Predicate: `request.path`, Action: "PUBLIC"}},
	}
	for _, tt := range invalid {
		t.Run("rejects "+tt.name, func(t *testing.T) {
			f, cookie := setupBackendTest(t)
			rr := f.CreateBackend(cookie, &api.ApiBackend{
				Fqdn:        "test.example.com",
				UpstreamUrl: "http://localhost:8080",
				AccessRules: []api.ApiAccessRule{{Predicate: `true`, Action: "PUBLIC"}, tt.rule},
			})
			assert.Equal(t, http.StatusBadRequest, rr.Code)
			assert.Contains(t, rr.Body.String(), "rule 2")
			assert.Empty(t, f.ListBackends(cookie))
		})
	}
}
