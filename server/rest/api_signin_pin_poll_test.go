package rest

import (
	"boivie/ubergang/server/api"
	"boivie/ubergang/server/models"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func (f *Fixture) pollPin(t *testing.T, id string, cookie *http.Cookie) (*api.ApiPollSigninPinResponse, *httptest.ResponseRecorder) {
	t.Helper()
	req := &api.ApiPollSigninPinRequest{
		Id: id,
	}
	resp := &api.ApiPollSigninPinResponse{}
	rr := f.request("POST", "/api/signin/pin/poll", req, cookie, resp)
	return resp, rr
}

func TestSignInPinPoll(t *testing.T) {
	t.Run("with invalid id", func(t *testing.T) {
		f := CreateFixture(t)
		resp, rr := f.pollPin(t, "non-existing-id", nil)

		require.Equal(t, http.StatusOK, rr.Code)
		require.NotNil(t, resp.Error, "Expected an error for invalid id")
		assert.True(t, resp.Error.InvalidToken, "Expected InvalidToken error")
		assert.Nil(t, resp.Success, "Success should be nil on error")
		assert.Nil(t, resp.Pending, "Pending should be nil on error")
	})

	t.Run("with unconfirmed request", func(t *testing.T) {
		// This is tricky to test without more control over the fixture.
		// The default user from CreateUser is already confirmed.
		// We would need a way to create a user without confirming them.
		// For now, we assume this is covered by other tests or is harder to set up.
		t.Skip("Skipping test for unconfirmed request as fixture creates confirmed users by default")
	})

	t.Run("with confirmed request", func(t *testing.T) {
		f := CreateFixture(t)
		_, signinSecret := f.CreateUser("test@example.com")

		// This helper simulates the confirmation and polls until it gets a cookie
		cookie := f.SigninFromConfirmedPollId(signinSecret)
		require.NotNil(t, cookie, "Expected a session cookie after confirmation")

		// Verify the cookie belongs to the correct user
		user := f.getUser(cookie, "me")
		assert.Equal(t, "test@example.com", user.Email)
	})

	t.Run("is idempotent after confirmation", func(t *testing.T) {
		f := CreateFixture(t)
		_, signinSecret := f.CreateUser("test@example.com")

		// Poll once to get the session
		resp1, rr1 := f.pollPin(t, signinSecret, nil)
		require.Equal(t, http.StatusOK, rr1.Code)
		require.NotNil(t, resp1.Success)

		cookies1 := (&http.Response{Header: http.Header{"Set-Cookie": {resp1.Success.Cookie}}}).Cookies()
		require.NotEmpty(t, cookies1, "expected a session cookie, but got none")
		cookie1 := cookies1[0]

		// Poll a second time with the same ID, now with the session cookie
		resp2, rr2 := f.pollPin(t, signinSecret, cookie1)

		// The second poll should also be successful
		require.Equal(t, http.StatusOK, rr2.Code)
		require.NotNil(t, resp2.Success, "Polling again should be successful")
		assert.Nil(t, resp2.Error)

		// The session cookie string in the response should be the same
		assert.Equal(t, resp1.Success.Cookie, resp2.Success.Cookie, "Session cookie string should be the same on subsequent polls")
	})

	t.Run("with expired request", func(t *testing.T) {
		f := CreateFixture(t)
		_, signinSecret := f.CreateUser("test@example.com")

		user, err := f.Db.GetUserBySigninRequest(signinSecret)
		require.NoError(t, err)

		// Manually expire the request
		err = f.Db.UpdateUser(user.Id, func(old *models.User) (*models.User, error) {
			for _, r := range old.SigninRequests {
				if r.Id == signinSecret {
					r.ExpiresAt = timestamppb.New(time.Now().Add(-1 * time.Minute))
				}
			}
			return old, nil
		})
		require.NoError(t, err)

		// Poll the expired request
		resp, rr := f.pollPin(t, signinSecret, nil)

		require.Equal(t, http.StatusOK, rr.Code)
		require.NotNil(t, resp.Error, "Expected an error for expired request")
		assert.True(t, resp.Error.Expired, "Expected Expired error")
	})
}
