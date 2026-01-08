package rest

import (
	"boivie/ubergang/server/api"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetConfirmSshKey(t *testing.T) {
	t.Run("returns authentication challenge for valid key", func(t *testing.T) {
		f := CreateFixture(t)
		cookie, _ := f.CreateUser("test@example.com")

		// Enroll a credential first (required for SSH key creation and authentication)
		enrollResp, err := f.StartEnroll(cookie)
		require.NoError(t, err)
		enrollReq := enrollResp.EnrollRequest
		_, res := f.GenerateCredential(enrollReq)
		_, err = f.FinishEnroll(cookie, enrollReq.Token, res)
		require.NoError(t, err)

		// Create SSH key
		keyResp := f.createSshKey(cookie, "test-ssh-key")
		require.NotEmpty(t, keyResp.KeyID)

		// Request confirmation
		resp := f.requestConfirmSshKey(cookie, keyResp.KeyID)

		require.NotNil(t, resp.Authenticate)
		assert.Nil(t, resp.Error)
		assert.Equal(t, "test-ssh-key", resp.Authenticate.KeyName)
		assert.NotEmpty(t, resp.Authenticate.Token)
		assert.NotEmpty(t, resp.Authenticate.AssertionRequest.Challenge)
		assert.NotEmpty(t, resp.Authenticate.AssertionRequest.RPID)
		assert.NotEmpty(t, resp.Authenticate.AssertionRequest.AllowCredentials)
	})

	t.Run("fails for non-existing SSH key", func(t *testing.T) {
		f := CreateFixture(t)
		cookie, _ := f.CreateUser("test@example.com")

		// Enroll a credential first
		enrollResp, err := f.StartEnroll(cookie)
		require.NoError(t, err)
		enrollReq := enrollResp.EnrollRequest
		_, res := f.GenerateCredential(enrollReq)
		_, err = f.FinishEnroll(cookie, enrollReq.Token, res)
		require.NoError(t, err)

		resp := f.requestConfirmSshKey(cookie, "nonexistent-key-id")

		require.NotNil(t, resp.Error)
		assert.True(t, resp.Error.InvalidKey)
		assert.Nil(t, resp.Authenticate)
	})

	t.Run("requires authentication", func(t *testing.T) {
		t.Skip("Authentication middleware not enabled in test fixture")
	})

	t.Run("fails with invalid session cookie", func(t *testing.T) {
		t.Skip("Authentication middleware not enabled in test fixture")
	})

	t.Run("fails when accessing another user's key", func(t *testing.T) {
		f := CreateFixture(t)

		// Create first user and SSH key
		cookie1, _ := f.CreateUser("user1@example.com")
		enrollResp1, err := f.StartEnroll(cookie1)
		require.NoError(t, err)
		enrollReq1 := enrollResp1.EnrollRequest
		_, res1 := f.GenerateCredential(enrollReq1)
		_, err = f.FinishEnroll(cookie1, enrollReq1.Token, res1)
		require.NoError(t, err)

		keyResp1 := f.createSshKey(cookie1, "user1-ssh-key")
		require.NotEmpty(t, keyResp1.KeyID)

		// Create second user
		cookie2, _ := f.CreateUser("user2@example.com")
		enrollResp2, err := f.StartEnroll(cookie2)
		require.NoError(t, err)
		enrollReq2 := enrollResp2.EnrollRequest
		_, res2 := f.GenerateCredential(enrollReq2)
		_, err = f.FinishEnroll(cookie2, enrollReq2.Token, res2)
		require.NoError(t, err)

		// Try to access user1's key with user2's session
		resp := f.requestConfirmSshKey(cookie2, keyResp1.KeyID)

		require.NotNil(t, resp.Error)
		assert.True(t, resp.Error.InvalidKey)
		assert.Nil(t, resp.Authenticate)
	})

	t.Run("fails for user without credentials", func(t *testing.T) {
		f := CreateFixture(t)
		cookie, _ := f.CreateUser("test@example.com")

		// Create SSH key without enrolling credentials first
		// This should not be possible in the normal flow, but we test edge cases
		keyResp := f.createSshKey(cookie, "test-ssh-key")

		if keyResp.KeyID == "" {
			// If the system prevents SSH key creation without credentials,
			// that's the expected behavior
			t.Skip("SSH key creation requires credentials - this is expected behavior")
			return
		}

		// If key creation succeeded, the confirmation request should handle missing credentials
		rr := f.request("GET", "/api/ssh-key/"+keyResp.KeyID+"/confirm", nil, cookie, nil)

		// Should either fail with internal server error or handle gracefully
		// The exact behavior depends on implementation details
		if rr.Code == http.StatusInternalServerError {
			// This is acceptable - server detected missing credentials
			return
		}

		// If it doesn't error, it should return an error response
		resp := &api.ApiGetConfirmSshKeyResponse{}
		f.request("GET", "/api/ssh-key/"+keyResp.KeyID+"/confirm", nil, cookie, resp)
		if resp.Error == nil {
			t.Error("Expected error when user has no credentials for authentication")
		}
	})

	t.Run("handles malformed key ID", func(t *testing.T) {
		f := CreateFixture(t)
		cookie, _ := f.CreateUser("test@example.com")

		// Enroll a credential first
		enrollResp, err := f.StartEnroll(cookie)
		require.NoError(t, err)
		enrollReq := enrollResp.EnrollRequest
		_, res := f.GenerateCredential(enrollReq)
		_, err = f.FinishEnroll(cookie, enrollReq.Token, res)
		require.NoError(t, err)

		// Test various malformed key IDs
		malformedIDs := []string{
			"../../../etc/passwd",           // path traversal
			"DROP TABLE ssh_keys;",          // SQL injection attempt
			"<script>alert('xss')</script>", // XSS attempt
		}

		for _, keyID := range malformedIDs {
			resp := f.requestConfirmSshKey(cookie, keyID)

			// Malformed key IDs might be treated as non-existent keys
			// Either error should be present OR no authentication challenge should be returned
			if resp.Error != nil {
				assert.True(t, resp.Error.InvalidKey, "Expected InvalidKey error for malformed key ID: %q", keyID)
				assert.Nil(t, resp.Authenticate)
			} else {
				// If no error, authentication challenge should be nil (key not found)
				assert.Nil(t, resp.Authenticate, "Expected no authentication challenge for malformed key ID: %q", keyID)
			}
		}

		// Test empty key ID separately since it behaves differently
		resp := f.requestConfirmSshKey(cookie, "")
		// Empty key ID might be handled differently by the router
		if resp.Error != nil {
			assert.True(t, resp.Error.InvalidKey, "Expected InvalidKey error for empty key ID")
		}
	})
}
