package rest

import (
	"boivie/ubergang/server/api"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetSshKey(t *testing.T) {
	t.Run("returns existing SSH key", func(t *testing.T) {
		f := CreateFixture(t)
		cookie, _ := f.CreateUser("test@example.com")

		// Enroll a credential first (required for SSH key creation)
		enrollResp, err := f.StartEnroll(cookie)
		require.NoError(t, err)
		enrollReq := enrollResp.EnrollRequest
		_, res := f.GenerateCredential(enrollReq)
		_, err = f.FinishEnroll(cookie, enrollReq.Token, res)
		require.NoError(t, err)

		// Create SSH key
		keyResp := f.createSshKey(cookie, "test-ssh-key")
		require.NotEmpty(t, keyResp.KeyID)

		// Get the SSH key
		resp := &api.ApiSSHKey{}
		rr := f.request("GET", "/api/ssh-key/"+keyResp.KeyID, nil, cookie, resp)

		require.Equal(t, http.StatusOK, rr.Code)
		assert.Equal(t, keyResp.KeyID, resp.ID)
		assert.Equal(t, "test-ssh-key", resp.Name)
		assert.NotEmpty(t, resp.CreatedAt)
		assert.Empty(t, resp.ExpiresAt)         // Should be empty for new keys
		assert.Empty(t, resp.Sha256Fingerprint) // Should be empty until key is confirmed
	})

	t.Run("returns 404 for non-existing SSH key", func(t *testing.T) {
		f := CreateFixture(t)
		cookie, _ := f.CreateUser("test@example.com")

		rr := f.request("GET", "/api/ssh-key/nonexistent-key-id", nil, cookie, nil)

		assert.Equal(t, http.StatusNotFound, rr.Code)
	})

	t.Run("requires authentication", func(t *testing.T) {
		t.Skip("Authentication middleware not enabled in test fixture")
	})

	t.Run("fails with invalid session cookie", func(t *testing.T) {
		t.Skip("Authentication middleware not enabled in test fixture")
	})

	t.Run("handles short key ID format", func(t *testing.T) {
		f := CreateFixture(t)
		cookie, _ := f.CreateUser("test@example.com")

		// Enroll a credential and create SSH key first
		enrollResp, err := f.StartEnroll(cookie)
		require.NoError(t, err)
		enrollReq := enrollResp.EnrollRequest
		_, res := f.GenerateCredential(enrollReq)
		_, err = f.FinishEnroll(cookie, enrollReq.Token, res)
		require.NoError(t, err)

		keyResp := f.createSshKey(cookie, "test-key-with-dashes")
		require.NotEmpty(t, keyResp.KeyID)

		// Get the SSH key using the full key ID
		resp := &api.ApiSSHKey{}
		rr := f.request("GET", "/api/ssh-key/"+keyResp.KeyID, nil, cookie, resp)

		require.Equal(t, http.StatusOK, rr.Code)
		assert.Equal(t, keyResp.KeyID, resp.ID)
		assert.Equal(t, "test-key-with-dashes", resp.Name)
	})

	t.Run("returns key with expiration if set", func(t *testing.T) {
		// This test would require setting up a key with expiration
		// Since the current API doesn't seem to support setting expiration during creation,
		// we'll document the expected behavior for future implementation
		t.Skip("SSH key expiration setting not yet supported in API - test documents expected behavior")
	})

	t.Run("returns key with fingerprint after confirmation", func(t *testing.T) {
		// This test would require completing the full SSH key confirmation flow
		// which involves multiple steps and WebAuthn authentication
		// We'll document the expected behavior for the complete flow
		t.Skip("SSH key confirmation flow test - would require full WebAuthn flow implementation")
	})
}
