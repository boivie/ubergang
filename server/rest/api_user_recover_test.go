package rest

import (
	"boivie/ubergang/server/api"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUserRecover(t *testing.T) {
	t.Run("creates recovery URL for existing user as admin", func(t *testing.T) {
		f := CreateFixture(t)
		adminCookie, _ := f.CreateAdmin("admin@example.com")
		_, targetUserId := f.CreateUserGetId("user@example.com")

		var resp api.ApiUserRecoverResponse
		rr := f.request("POST", "/api/user/"+targetUserId+"/recover", nil, adminCookie, &resp)
		assert.Equal(t, http.StatusOK, rr.Code)
		assert.NotEmpty(t, resp.RecoveryUrl)
		assert.True(t, strings.HasPrefix(resp.RecoveryUrl, "https://test.example.com/signin/"))

		// Verify the recovery URL has a valid token format
		parts := strings.Split(resp.RecoveryUrl, "/signin/")
		require.Len(t, parts, 2)
		token := parts[1]
		assert.NotEmpty(t, token)
		assert.Greater(t, len(token), 10, "Token should be reasonably long")

		// Verify the user object was updated with the signin request
		targetUser, err := f.Db.GetUserById(targetUserId)
		require.NoError(t, err)
		require.NotEmpty(t, targetUser.SigninRequests)

		// Find the signin request with the token
		var foundSigninRequest bool
		for _, sr := range targetUser.SigninRequests {
			if sr.Id == token {
				assert.True(t, sr.Confirmed, "SigninRequest should be confirmed")
				assert.NotNil(t, sr.ExpiresAt, "SigninRequest should have expiry")
				foundSigninRequest = true
				break
			}
		}
		assert.True(t, foundSigninRequest, "Should find signin request with the token")
	})

	t.Run("returns not found for non-existent user", func(t *testing.T) {
		f := CreateFixture(t)
		adminCookie, _ := f.CreateAdmin("admin@example.com")

		rr := f.request("POST", "/api/user/non-existent-id/recover", nil, adminCookie, nil)
		assert.Equal(t, http.StatusNotFound, rr.Code)
	})

	t.Run("is forbidden for non-admin user", func(t *testing.T) {
		f := CreateFixture(t)
		_, _ = f.CreateAdmin("admin@example.com")
		regularCookie, _ := f.CreateUser("user@example.com")
		_, targetUserId := f.CreateUserGetId("targetuser@example.com")

		rr := f.request("POST", "/api/user/"+targetUserId+"/recover", nil, regularCookie, nil)
		assert.Equal(t, http.StatusForbidden, rr.Code)
	})

	t.Run("is forbidden for unauthenticated user", func(t *testing.T) {
		f := CreateFixture(t)
		_, targetUserId := f.CreateAdminGetId("user@example.com")

		rr := f.request("POST", "/api/user/"+targetUserId+"/recover", nil, nil, nil)
		assert.Equal(t, http.StatusForbidden, rr.Code)
	})

	t.Run("allows admin to create recovery URL for another admin", func(t *testing.T) {
		f := CreateFixture(t)
		adminCookie, _ := f.CreateAdmin("admin@example.com")
		_, targetAdminId := f.CreateAdminGetId("targetadmin@example.com")

		var resp api.ApiUserRecoverResponse
		rr := f.request("POST", "/api/user/"+targetAdminId+"/recover", nil, adminCookie, &resp)
		assert.Equal(t, http.StatusOK, rr.Code)
		assert.NotEmpty(t, resp.RecoveryUrl)
		assert.True(t, strings.HasPrefix(resp.RecoveryUrl, "https://test.example.com/signin/"))
	})

	t.Run("creates multiple recovery URLs for same user", func(t *testing.T) {
		f := CreateFixture(t)
		adminCookie, _ := f.CreateAdmin("admin@example.com")
		_, targetUserId := f.CreateUserGetId("user@example.com")

		// Create first recovery URL
		var resp1 api.ApiUserRecoverResponse
		rr1 := f.request("POST", "/api/user/"+targetUserId+"/recover", nil, adminCookie, &resp1)
		assert.Equal(t, http.StatusOK, rr1.Code)
		assert.NotEmpty(t, resp1.RecoveryUrl)

		// Create second recovery URL
		var resp2 api.ApiUserRecoverResponse
		rr2 := f.request("POST", "/api/user/"+targetUserId+"/recover", nil, adminCookie, &resp2)
		assert.Equal(t, http.StatusOK, rr2.Code)
		assert.NotEmpty(t, resp2.RecoveryUrl)

		// URLs should be different
		assert.NotEqual(t, resp1.RecoveryUrl, resp2.RecoveryUrl)

		// Verify both new tokens exist in the user's signin requests
		// Note: CreateUserGetId already creates 1 signin request, so we expect 3 total (1 original + 2 recovery)
		targetUser, err := f.Db.GetUserById(targetUserId)
		require.NoError(t, err)
		require.Len(t, targetUser.SigninRequests, 3, "User should have 3 signin requests (1 original + 2 recovery)")

		token1 := strings.Split(resp1.RecoveryUrl, "/signin/")[1]
		token2 := strings.Split(resp2.RecoveryUrl, "/signin/")[1]

		var foundToken1, foundToken2 bool
		for _, sr := range targetUser.SigninRequests {
			if sr.Id == token1 {
				foundToken1 = true
			}
			if sr.Id == token2 {
				foundToken2 = true
			}
		}
		assert.True(t, foundToken1, "Should find first token")
		assert.True(t, foundToken2, "Should find second token")
	})

	t.Run("recovery URL can be used to sign in", func(t *testing.T) {
		f := CreateFixture(t)
		adminCookie, _ := f.CreateAdmin("admin@example.com")
		_, targetUserId := f.CreateUserGetId("user@example.com")

		// Create recovery URL
		var resp api.ApiUserRecoverResponse
		rr := f.request("POST", "/api/user/"+targetUserId+"/recover", nil, adminCookie, &resp)
		assert.Equal(t, http.StatusOK, rr.Code)

		// Extract token from URL
		token := strings.Split(resp.RecoveryUrl, "/signin/")[1]

		// Use the recovery token to sign in (simulate the recovery flow)
		userCookie := f.SigninFromConfirmedPollId(token)
		assert.NotNil(t, userCookie, "Should be able to create session from recovery token")

		// Verify we can access user data with the new session
		user := f.getUser(userCookie, "me")
		assert.Equal(t, targetUserId, user.ID)
		assert.Equal(t, "user@example.com", user.Email)
	})
}
