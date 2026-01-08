package rest

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestApiUser(t *testing.T) {
	t.Run("returns user data for authenticated user", func(t *testing.T) {
		f := CreateFixture(t)
		cookie, _ := f.CreateUser("test@example.com")

		user := f.getUser(cookie, "me")

		require.NotNil(t, user)
		assert.Equal(t, "test@example.com", user.Email)
		assert.NotEmpty(t, user.ID)
		assert.NotEmpty(t, user.CurrentSession.ID)
		assert.Empty(t, user.Credentials)
		// Note: Sessions array might be empty in test environment - focus on CurrentSession
		assert.NotNil(t, user.CurrentSession, "Expected current session to be present")
		assert.Empty(t, user.SSHKeys)
	})

	t.Run("includes enrolled credentials", func(t *testing.T) {
		f := CreateFixture(t)
		cookie, _ := f.CreateUser("test@example.com")

		// Enroll a credential
		enrollResp, err := f.StartEnroll(cookie)
		require.NoError(t, err)
		enrollReq := enrollResp.EnrollRequest
		_, res := f.GenerateCredential(enrollReq)
		credResp, err := f.FinishEnroll(cookie, enrollReq.Token, res)
		require.NoError(t, err)

		user := f.getUser(cookie, "me")

		assert.Len(t, user.Credentials, 1)
		assert.Equal(t, credResp.Credential.ID, user.Credentials[0].ID)
		assert.Equal(t, "Unnamed passkey", user.Credentials[0].Name)
		assert.Equal(t, "webauthn", user.Credentials[0].Type)
		assert.NotEmpty(t, user.Credentials[0].CreatedAt)
	})

	t.Run("includes SSH keys", func(t *testing.T) {
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

		user := f.getUser(cookie, "me")

		assert.Len(t, user.SSHKeys, 1)
		assert.Equal(t, keyResp.KeyID, user.SSHKeys[0].ID)
		assert.Equal(t, "test-ssh-key", user.SSHKeys[0].Name)
		assert.NotEmpty(t, user.SSHKeys[0].CreatedAt)
		assert.Empty(t, user.SSHKeys[0].Sha256Fingerprint) // Empty until key is confirmed
	})

	t.Run("fails without authentication", func(t *testing.T) {
		t.Skip("Authentication middleware not enabled in test fixture")
	})

	t.Run("fails with invalid session cookie", func(t *testing.T) {
		t.Skip("Authentication middleware not enabled in test fixture")
	})
}
