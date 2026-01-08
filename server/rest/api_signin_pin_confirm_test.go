package rest

import (
	"boivie/ubergang/server/api"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/descope/virtualwebauthn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupPinConfirm performs the initial steps of a PIN-based sign-in flow:
// 1. Creates a user and enrolls a credential.
// 2. Initiates a PIN sign-in request.
// 3. Polls for the request to be available.
// 4. Queries with the correct PIN.
// It returns the fixture, user cookie, the virtual credential, the original enroll request,
// and the response from the PIN query which contains the assertion request and token.
func setupPinConfirm(t *testing.T) (*Fixture, *http.Cookie, virtualwebauthn.Credential, *api.ApiEnrollRequest, *api.ApiQuerySigninPinResponse) {
	t.Helper()
	f := CreateFixture(t)
	cookie, _ := f.CreateUser("test@example.com")
	enrollResp, err := f.StartEnroll(cookie)
	require.NoError(t, err)
	enrollReq := enrollResp.EnrollRequest
	cred, res := f.GenerateCredential(enrollReq)
	_, err = f.FinishEnroll(cookie, enrollReq.Token, res)
	require.NoError(t, err, "FinishEnroll failed")

	// 1. Request PIN sign-in
	req1 := &api.ApiRequestSigninPinRequest{
		Email:     "test@example.com",
		UserAgent: "test useragent",
	}
	resp1 := &api.ApiRequestSigninPinResponse{}
	f.request("POST", "/api/signin/pin/request", req1, nil, resp1)
	require.Nil(t, resp1.Error, "PIN request failed")
	require.NotEmpty(t, resp1.ID, "PIN request ID is empty")

	// 2. Poll for the request
	req2 := &api.ApiPollSigninPinRequest{Id: resp1.ID}
	resp2 := &api.ApiPollSigninPinResponse{}
	f.request("POST", "/api/signin/pin/poll", req2, nil, resp2)
	require.Nil(t, resp2.Error, "PIN poll failed")
	require.NotNil(t, resp2.Pending, "Expected pending sign-in")

	// 3. Query with PIN
	req3 := &api.ApiQuerySigninPinRequest{Pin: resp2.Pending.Pin}
	resp3 := &api.ApiQuerySigninPinResponse{}
	f.request("POST", "/api/signin/pin/query", req3, cookie, resp3)
	require.Nil(t, resp3.Error, "PIN query failed")
	require.NotEmpty(t, resp3.Token, "PIN query token is empty")

	return f, cookie, cred, enrollReq, resp3
}

func (f *Fixture) confirmPinSignin(t *testing.T, cookie *http.Cookie, req *api.ApiConfirmSigninPinRequest) (*api.ApiConfirmSigninPinResponse, *httptest.ResponseRecorder) {
	t.Helper()
	resp := &api.ApiConfirmSigninPinResponse{}
	rr := f.request("POST", "/api/signin/pin/confirm", req, cookie, resp)
	return resp, rr
}

func TestSigninPinConfirm(t *testing.T) {
	t.Run("successful confirmation", func(t *testing.T) {
		f, cookie, cred, enrollReq, queryResp := setupPinConfirm(t)

		assertionResponse := f.SignAssertionRequest(queryResp.AssertionRequest, enrollReq.Options.User.ID, &cred)

		req := &api.ApiConfirmSigninPinRequest{
			Token:      queryResp.Token,
			Credential: *assertionResponse,
		}
		resp, rr := f.confirmPinSignin(t, cookie, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		assert.Nil(t, resp.Error, "Expected no error on confirmation")

		// Verify the sign-in request was marked as confirmed
		userIDBytes, err := base64.RawURLEncoding.DecodeString(enrollReq.Options.User.ID)
		require.NoError(t, err)
		user, err := f.Db.GetUserById(string(userIDBytes))
		require.NoError(t, err)
		found := false
		for _, r := range user.SigninRequests {
			if r.Pin == queryResp.Pin && r.Confirmed {
				found = true
				break
			}
		}
		assert.True(t, found, "Signin request was not marked as confirmed")
	})

	t.Run("fails with invalid token", func(t *testing.T) {
		f, cookie, cred, enrollReq, queryResp := setupPinConfirm(t)
		assertionResponse := f.SignAssertionRequest(queryResp.AssertionRequest, enrollReq.Options.User.ID, &cred)

		req := &api.ApiConfirmSigninPinRequest{
			Token:      "invalid-token",
			Credential: *assertionResponse,
		}
		resp, rr := f.confirmPinSignin(t, cookie, req)

		assert.Equal(t, http.StatusOK, rr.Code) // Handler returns 200 with error in body
		require.NotNil(t, resp.Error, "Expected an error")
		assert.True(t, resp.Error.InvalidEnrollment, "Expected InvalidEnrollment error")
	})

	t.Run("fails with invalid assertion", func(t *testing.T) {
		f, cookie, _, enrollReq, queryResp := setupPinConfirm(t)

		// Generate a new random credential to make the signature invalid
		otherCred := virtualwebauthn.NewCredential(virtualwebauthn.KeyTypeEC2)
		assertionResponse := f.SignAssertionRequest(queryResp.AssertionRequest, enrollReq.Options.User.ID, &otherCred)

		req := &api.ApiConfirmSigninPinRequest{
			Token:      queryResp.Token,
			Credential: *assertionResponse,
		}
		resp, rr := f.confirmPinSignin(t, cookie, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		require.NotNil(t, resp.Error, "Expected an error")
		assert.True(t, resp.Error.InvalidEnrollment, "Expected InvalidEnrollment error for bad assertion")
	})

	t.Run("fails with no session", func(t *testing.T) {
		f, _, cred, enrollReq, queryResp := setupPinConfirm(t)
		assertionResponse := f.SignAssertionRequest(queryResp.AssertionRequest, enrollReq.Options.User.ID, &cred)

		req := &api.ApiConfirmSigninPinRequest{
			Token:      queryResp.Token,
			Credential: *assertionResponse,
		}
		// Make request without a cookie
		_, rr := f.confirmPinSignin(t, nil, req)

		assert.Equal(t, http.StatusForbidden, rr.Code)
	})

	t.Run("fails with mismatched user", func(t *testing.T) {
		f, _, cred, enrollReq, queryResp := setupPinConfirm(t)

		// Create a second user and try to use their session
		otherUserCookie, _ := f.CreateUser("other@example.com")

		assertionResponse := f.SignAssertionRequest(queryResp.AssertionRequest, enrollReq.Options.User.ID, &cred)

		req := &api.ApiConfirmSigninPinRequest{
			Token:      queryResp.Token,
			Credential: *assertionResponse,
		}
		_, rr := f.confirmPinSignin(t, otherUserCookie, req)

		// This should fail because the token's user ID won't match the session user ID
		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})
}
