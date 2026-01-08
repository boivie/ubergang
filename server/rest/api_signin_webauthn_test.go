package rest

import (
	"boivie/ubergang/server/api"
	"net/http"
	"testing"

	"github.com/descope/virtualwebauthn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupUserWithCredential creates a user and enrolls a WebAuthn credential.
// It returns the fixture, user cookie, the original enroll request, and the
// virtual credential object needed for signing assertions.
func setupUserWithCredential(t *testing.T) (*Fixture, *http.Cookie, *api.ApiEnrollRequest, virtualwebauthn.Credential) {
	t.Helper()
	f := CreateFixture(t)
	cookie, _ := f.CreateUser("test")
	resp, err := f.StartEnroll(cookie)
	require.NoError(t, err)
	request := resp.EnrollRequest
	cred, res := f.GenerateCredential(request)
	_, err = f.FinishEnroll(cookie, request.Token, res)
	require.NoError(t, err, "FinishEnroll failed")
	return f, cookie, request, cred
}

func (f *Fixture) signinWebauthn(t *testing.T, cookie *http.Cookie, token string, credential *api.ApiAssertionCredential) *http.Cookie {
	t.Helper()
	req := &api.ApiSignInWebauthnRequest{
		Token:      token,
		Credential: *credential,
	}
	resp := &api.ApiSignInWebauthResponse{}
	rr := f.request("POST", "/api/signin/webauthn", req, cookie, resp)
	require.Equal(t, http.StatusOK, rr.Code, "request to /api/signin/webauthn failed")
	require.Nil(t, resp.Error, "expected signin to be successful, but got error: %+v", resp.Error)
	require.NotNil(t, resp.Success, "expected signin to be successful, but it was not. response: %s", rr.Body.String())

	cs := resp.Success.Cookie
	cookies := (&http.Response{Header: http.Header{"Set-Cookie": {cs}}}).Cookies()
	require.NotEmpty(t, cookies, "expected a session cookie, but got none")

	return cookies[0]
}

func TestSigninWebauthn(t *testing.T) {
	t.Run("with email creates new session", func(t *testing.T) {
		f, cookie, enrollReq, cred := setupUserWithCredential(t)

		signin := f.signinEmail(t, "test")
		assertionResponse := f.SignAssertionRequest(&signin.Success.AssertionRequest, enrollReq.Options.User.ID, &cred)

		// Sign in without providing an existing session cookie
		cookie2 := f.signinWebauthn(t, nil, signin.Success.Token, assertionResponse)

		oldSessionId := f.getUser(cookie, "me").CurrentSession.ID
		newSessionId := f.getUser(cookie2, "me").CurrentSession.ID

		assert.NotEqual(t, oldSessionId, newSessionId, "Expected a new session to be created, but session was reused")
	})

	t.Run("with email reuses existing session", func(t *testing.T) {
		f, cookie, enrollReq, cred := setupUserWithCredential(t)

		signin := f.signinEmail(t, "test")
		assertionResponse := f.SignAssertionRequest(&signin.Success.AssertionRequest, enrollReq.Options.User.ID, &cred)

		// Sign in providing an existing session cookie
		cookie2 := f.signinWebauthn(t, cookie, signin.Success.Token, assertionResponse)

		oldSessionId := f.getUser(cookie, "me").CurrentSession.ID
		newSessionId := f.getUser(cookie2, "me").CurrentSession.ID

		assert.Equal(t, oldSessionId, newSessionId, "Expected session to be reused, but a new one was created.")
	})

	t.Run("passwordless signin creates new session", func(t *testing.T) {
		f, cookie, enrollReq, cred := setupUserWithCredential(t)

		resp := &api.ApiStartSigninResponse{}
		rr := f.request("GET", "/api/signin/start", nil, nil, resp)
		require.Equal(t, http.StatusOK, rr.Code, "request to /api/signin/start failed")

		assertionResponse := f.SignAssertionRequest(&resp.AssertionRequest, enrollReq.Options.User.ID, &cred)

		cookie2 := f.signinWebauthn(t, nil, resp.Token, assertionResponse)
		oldSessionId := f.getUser(cookie, "me").CurrentSession.ID
		newSessionId := f.getUser(cookie2, "me").CurrentSession.ID

		assert.NotEqual(t, oldSessionId, newSessionId, "Expected a new session to be created, but session was reused")
	})
}
