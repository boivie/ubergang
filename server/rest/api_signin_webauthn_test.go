package rest

import (
	"boivie/ubergang/server/api"
	"boivie/ubergang/server/models"
	"encoding/base64"
	"net/http"
	"testing"
	"time"

	"github.com/descope/virtualwebauthn"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"
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

func TestLocalRedirect(t *testing.T) {
	for redirect, want := range map[string]string{
		"":              "",
		"/":             "/",
		"/backends?a=b": "/backends?a=b",
		"/authorize?rd=https%3A%2F%2Fbe.example.com%2Fx": "/authorize?rd=https%3A%2F%2Fbe.example.com%2Fx",
		"//evil.example.com":                             "",
		"/\\evil.example.com":                            "",
		"/\t/evil.example.com":                           "",
		"https://evil.example.com":                       "",
		"javascript:alert(1)":                            "",
	} {
		assert.Equal(t, want, localRedirect(redirect), "redirect %q", redirect)
	}
}

func (f *Fixture) trySigninWebauthn(token string, credential *api.ApiAssertionCredential) *api.ApiSignInWebauthResponse {
	req := &api.ApiSignInWebauthnRequest{Token: token, Credential: *credential}
	resp := &api.ApiSignInWebauthResponse{}
	f.request("POST", "/api/signin/webauthn", req, nil, resp)
	return resp
}

func TestPasswordlessSigninCantBeReplayed(t *testing.T) {
	f, _, enrollReq, cred := setupUserWithCredential(t)
	start := &api.ApiStartSigninResponse{}
	f.request("GET", "/api/signin/start", nil, nil, start)
	assertion := f.SignAssertionRequest(&start.AssertionRequest, enrollReq.Options.User.ID, &cred)

	require.NotNil(t, f.trySigninWebauthn(start.Token, assertion).Success)

	resp := f.trySigninWebauthn(start.Token, assertion)
	assert.Nil(t, resp.Success)
	assert.NotNil(t, resp.Error)
}

func TestPasswordlessSigninRequiresStoredChallenge(t *testing.T) {
	// A validly signed assertion over a challenge the server didn't issue, with
	// a token it doesn't know about.
	f, _, enrollReq, cred := setupUserWithCredential(t)
	assertion := f.SignAssertionRequest(&api.ApiAssertionRequest{
		Challenge:        base64.RawURLEncoding.EncodeToString([]byte("an-attacker-chosen-challenge-123")),
		RPID:             "test.example.com",
		UserVerification: "required",
	}, enrollReq.Options.User.ID, &cred)

	token, err := uuid.NewV7()
	require.NoError(t, err)
	resp := f.trySigninWebauthn(token.String(), assertion)
	assert.Nil(t, resp.Success)
	assert.NotNil(t, resp.Error)
}

func TestPasswordlessSigninExpires(t *testing.T) {
	for name, tc := range map[string]struct {
		expiresAt time.Time
		succeeds  bool
	}{
		"valid":   {time.Now().Add(time.Minute), true},
		"expired": {time.Now().Add(-time.Second), false},
	} {
		t.Run(name, func(t *testing.T) {
			f, _, enrollReq, cred := setupUserWithCredential(t)
			challenge := base64.RawURLEncoding.EncodeToString([]byte("a-challenge-issued-by-the-server"))
			token, err := uuid.NewV7()
			require.NoError(t, err)
			err = f.Db.StoreAuthenticationState(&token, &models.AuthenticationState{
				UserVerification: "required",
				Challenge:        challenge,
				ExpiresAt:        timestamppb.New(tc.expiresAt),
				Type:             &models.AuthenticationState_SignIn{SignIn: &models.AthenticationStateSignIn{}},
			})
			require.NoError(t, err)

			assertion := f.SignAssertionRequest(&api.ApiAssertionRequest{
				Challenge:        challenge,
				RPID:             "test.example.com",
				UserVerification: "required",
			}, enrollReq.Options.User.ID, &cred)
			resp := f.trySigninWebauthn(token.String(), assertion)
			assert.Equal(t, tc.succeeds, resp.Success != nil, "error: %+v", resp.Error)
		})
	}
}

func TestSigninStartStoresExpiringChallenge(t *testing.T) {
	f := CreateFixture(t)
	start := &api.ApiStartSigninResponse{}
	f.request("GET", "/api/signin/start", nil, nil, start)

	state, err := f.Db.ConsumeAuthenticationState(start.Token)
	require.NoError(t, err)
	assert.Empty(t, state.UserId)
	assert.Equal(t, start.AssertionRequest.Challenge, state.Challenge)
	assert.Equal(t, "required", state.UserVerification)
	assert.WithinDuration(t, time.Now().Add(5*time.Minute), state.ExpiresAt.AsTime(), time.Minute)
}
