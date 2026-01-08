package rest

import (
	"boivie/ubergang/server/api"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupEnrollment creates a user and starts the enrollment process.
func setupEnrollment(t *testing.T) (*Fixture, *http.Cookie, *api.ApiEnrollRequest) {
	t.Helper()
	f := CreateFixture(t)
	cookie, _ := f.CreateUser("test")
	resp, err := f.StartEnroll(cookie)
	require.NoError(t, err)
	return f, cookie, resp.EnrollRequest
}

func TestFinishEnroll(t *testing.T) {
	t.Run("successfully enrolls a new credential", func(t *testing.T) {
		f, cookie, request := setupEnrollment(t)

		_, res := f.GenerateCredential(request)
		enrollResp, err := f.FinishEnroll(cookie, request.Token, res)
		require.NoError(t, err)
		assert.NotEmpty(t, enrollResp.Credential.ID, "Expected a credential ID, but got none")

		user := f.getUser(cookie, "me")
		assert.Len(t, user.Credentials, 1, "Expected user to have one credential")
		assert.Equal(t, enrollResp.Credential.ID, user.Credentials[0].ID)
	})

	t.Run("fails with invalid token", func(t *testing.T) {
		f, cookie, request := setupEnrollment(t)

		_, res := f.GenerateCredential(request)
		_, err := f.FinishEnroll(cookie, "invalid-token", res)
		require.Error(t, err)
		// A more specific error check could be added here
		// if the API provides distinct error types.
	})

	t.Run("fails if token is already used", func(t *testing.T) {
		f, cookie, request := setupEnrollment(t)

		_, res1 := f.GenerateCredential(request)
		_, err := f.FinishEnroll(cookie, request.Token, res1)
		require.NoError(t, err)

		// Try to use the same token again
		_, res2 := f.GenerateCredential(request)
		_, err = f.FinishEnroll(cookie, request.Token, res2)
		require.Error(t, err, "Expected error when reusing an enrollment token")
	})

	t.Run("fails with token from another session", func(t *testing.T) {
		f, _, request1 := setupEnrollment(t)

		// Create a second user and session
		cookie2, _ := f.CreateUser("test2")

		// Try to finish enrollment for user 2 using token from user 1
		_, res := f.GenerateCredential(request1)
		_, err := f.FinishEnroll(cookie2, request1.Token, res)
		require.Error(t, err, "Expected error when using a token from a different session")
	})

	t.Run("fails if not logged in", func(t *testing.T) {
		f, _, request := setupEnrollment(t)

		_, res := f.GenerateCredential(request)
		// Make request without a session cookie
		_, err := f.FinishEnroll(nil, request.Token, res)
		require.Error(t, err, "Expected error when trying to enroll without being logged in")
	})
}
