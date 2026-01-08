package rest

import (
	"boivie/ubergang/server/api"
	"net/http"
	"testing"

	"github.com/descope/virtualwebauthn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupPinRequest creates a user and a PIN sign-in request, returning the test
// fixture, the user's session cookie, the generated PIN, the enrolled credential,
// and the original enroll request.
func setupPinRequest(t *testing.T) (*Fixture, *http.Cookie, string, virtualwebauthn.Credential, *api.ApiEnrollRequest) {
	t.Helper()
	f := CreateFixture(t)
	cookie, _ := f.CreateUser("test@example.com")

	// Enroll a credential for the user
	enrollResp, err := f.StartEnroll(cookie)
	require.NoError(t, err)
	enrollReq := enrollResp.EnrollRequest
	cred, res := f.GenerateCredential(enrollReq)
	_, err = f.FinishEnroll(cookie, enrollReq.Token, res)
	require.NoError(t, err, "FinishEnroll failed")

	// Create a PIN sign-in request
	req1 := &api.ApiRequestSigninPinRequest{Email: "test@example.com", UserAgent: "test-user-agent"}
	resp1 := &api.ApiRequestSigninPinResponse{}
	f.request("POST", "/api/signin/pin/request", req1, nil, resp1)
	require.Nil(t, resp1.Error, "PIN request failed")

	// Poll for the PIN
	req2 := &api.ApiPollSigninPinRequest{Id: resp1.ID}
	resp2 := &api.ApiPollSigninPinResponse{}
	f.request("POST", "/api/signin/pin/poll", req2, nil, resp2)
	require.NotNil(t, resp2.Pending, "PIN poll failed")

	return f, cookie, resp2.Pending.Pin, cred, enrollReq
}

func (f *Fixture) confirmSignin(t *testing.T, cookie *http.Cookie, token string, credential *api.ApiAssertionCredential) {
	t.Helper()
	req := &api.ApiConfirmSigninPinRequest{
		Token:      token,
		Credential: *credential,
	}
	resp := &api.ApiConfirmSigninPinResponse{}
	rr := f.request("POST", "/api/signin/pin/confirm", req, cookie, resp)
	require.Equal(t, http.StatusOK, rr.Code)
	require.Nil(t, resp.Error, "confirm signin failed: %+v", resp.Error)
}

func TestHandleSigninPinQuery(t *testing.T) {
	t.Run("should return error if no session", func(t *testing.T) {
		f, _, pin, _, _ := setupPinRequest(t)
		req := &api.ApiQuerySigninPinRequest{Pin: pin}
		resp := &api.ApiQuerySigninPinResponse{}
		// Note: No cookie is sent
		rr := f.request("POST", "/api/signin/pin/query", req, nil, resp)
		assert.Equal(t, http.StatusForbidden, rr.Code)
	})

	t.Run("should return error for invalid pin", func(t *testing.T) {
		f, cookie, _, _, _ := setupPinRequest(t)
		req := &api.ApiQuerySigninPinRequest{Pin: "123-456"}
		resp := &api.ApiQuerySigninPinResponse{}
		rr := f.request("POST", "/api/signin/pin/query", req, cookie, resp)
		assert.Equal(t, http.StatusOK, rr.Code)
		assert.NotNil(t, resp.Error, "expected an error")
		assert.True(t, resp.Error.InvalidPin, "expected InvalidPin error")
	})

	t.Run("should return error if user has no credentials", func(t *testing.T) {
		f := CreateFixture(t)
		cookie, _ := f.CreateUser("test@example.com")

		// Create a PIN sign-in request for a user without credentials
		req1 := &api.ApiRequestSigninPinRequest{Email: "test@example.com"}
		resp1 := &api.ApiRequestSigninPinResponse{}
		f.request("POST", "/api/signin/pin/request", req1, nil, resp1)
		require.Nil(t, resp1.Error)

		req2 := &api.ApiPollSigninPinRequest{Id: resp1.ID}
		resp2 := &api.ApiPollSigninPinResponse{}
		f.request("POST", "/api/signin/pin/poll", req2, nil, resp2)
		require.NotNil(t, resp2.Pending)

		// Query with the PIN
		req3 := &api.ApiQuerySigninPinRequest{Pin: resp2.Pending.Pin}
		resp3 := &api.ApiQuerySigninPinResponse{}
		rr := f.request("POST", "/api/signin/pin/query", req3, cookie, resp3)

		assert.Equal(t, http.StatusOK, rr.Code)
		require.NotNil(t, resp3.Error, "expected an error due to no credentials")
		assert.True(t, resp3.Error.InvalidCredentials, "expected InvalidCredentials error")
	})

	t.Run("should return assertion request for valid pin", func(t *testing.T) {
		f, cookie, pin, _, _ := setupPinRequest(t)
		req := &api.ApiQuerySigninPinRequest{Pin: pin}
		resp := &api.ApiQuerySigninPinResponse{}
		rr := f.request("POST", "/api/signin/pin/query", req, cookie, resp)

		assert.Equal(t, http.StatusOK, rr.Code)
		require.Nil(t, resp.Error, "expected no error, got %+v", resp.Error)
		assert.Equal(t, "test-user-agent", resp.RequestorUserAgent)
		assert.False(t, resp.Confirmed, "request should not be confirmed yet")
		assert.NotNil(t, resp.AssertionRequest, "expected an assertion request")
		assert.NotEmpty(t, resp.Token, "expected a token")
	})

	t.Run("should handle pin with spaces and dashes", func(t *testing.T) {
		f, cookie, pin, _, _ := setupPinRequest(t)
		// In the DB the pin is stored as "123456", but the user might enter "123-456"
		formattedPin := pin[:3] + "-" + pin[3:]
		req := &api.ApiQuerySigninPinRequest{Pin: formattedPin}
		resp := &api.ApiQuerySigninPinResponse{}
		rr := f.request("POST", "/api/signin/pin/query", req, cookie, resp)

		assert.Equal(t, http.StatusOK, rr.Code)
		require.Nil(t, resp.Error, "expected no error for formatted pin")
		assert.False(t, resp.Confirmed)
		assert.NotNil(t, resp.AssertionRequest)
	})

	t.Run("should not return assertion if already confirmed", func(t *testing.T) {
		f, cookie, pin, cred, enrollReq := setupPinRequest(t)

		// First, query and get the assertion
		req1 := &api.ApiQuerySigninPinRequest{Pin: pin}
		resp1 := &api.ApiQuerySigninPinResponse{}
		f.request("POST", "/api/signin/pin/query", req1, cookie, resp1)
		require.Nil(t, resp1.Error)
		require.NotNil(t, resp1.AssertionRequest)

		// Confirm the sign-in
		assertion := f.SignAssertionRequest(resp1.AssertionRequest, enrollReq.Options.User.ID, &cred)
		f.confirmSignin(t, cookie, resp1.Token, assertion)

		// Now, query again with the same PIN
		req2 := &api.ApiQuerySigninPinRequest{Pin: pin}
		resp2 := &api.ApiQuerySigninPinResponse{}
		rr := f.request("POST", "/api/signin/pin/query", req2, cookie, resp2)

		assert.Equal(t, http.StatusOK, rr.Code)
		require.Nil(t, resp2.Error, "expected no error on second query")
		assert.True(t, resp2.Confirmed, "expected request to be confirmed")
		assert.Nil(t, resp2.AssertionRequest, "should not get a new assertion request")
	})
}
