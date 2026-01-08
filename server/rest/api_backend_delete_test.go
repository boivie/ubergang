package rest

import (
	"boivie/ubergang/server/api"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDeleteBackend(t *testing.T) {
	t.Run("deletes existing backend", func(t *testing.T) {
		f := CreateFixture(t)
		cookie, _ := f.CreateAdmin("test@example.com")

		// Create a backend
		backend := &api.ApiBackend{
			Fqdn:        "api.example.com",
			UpstreamUrl: "http://localhost:3000",
		}
		rr := f.CreateBackend(cookie, backend)
		require.Equal(t, http.StatusOK, rr.Code)

		// Verify it exists
		backends := f.ListBackends(cookie)
		require.Len(t, backends, 1)

		// Delete the backend
		rr = f.request("DELETE", "/api/backend/api.example.com", nil, cookie, nil)
		assert.Equal(t, http.StatusNoContent, rr.Code)

		// Verify it's gone
		backends = f.ListBackends(cookie)
		assert.Empty(t, backends)
	})

	t.Run("returns okey for non-existent backend", func(t *testing.T) {
		f := CreateFixture(t)
		cookie, _ := f.CreateAdmin("test@example.com")

		// Delete a non-existent backend
		rr := f.request("DELETE", "/api/backend/non-existent.example.com", nil, cookie, nil)
		assert.Equal(t, http.StatusNoContent, rr.Code)
	})

	t.Run("requires authentication", func(t *testing.T) {
		f := CreateFixture(t)
		// No user, no cookie

		// Delete a backend without authentication
		rr := f.request("DELETE", "/api/backend/any.example.com", nil, nil, nil)
		assert.Equal(t, http.StatusForbidden, rr.Code)
	})
}
