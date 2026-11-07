package rest

import (
	"boivie/ubergang/server/api"
	"boivie/ubergang/server/models"
	"boivie/ubergang/server/session"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/descope/virtualwebauthn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	kioskHost   = "kiosk.example.com"
	noKioskHost = "nokiosk.example.com"
)

type kioskFixture struct {
	*Fixture
	adminCookie *http.Cookie
	// A user that can access kioskHost and noKioskHost, with a passkey.
	approverCookie *http.Cookie
	approverId     string
	enrollReq      *api.ApiEnrollRequest
	cred           virtualwebauthn.Credential
}

func setupKiosk(t *testing.T) *kioskFixture {
	t.Helper()
	f := CreateFixture(t)
	adminCookie, _ := f.CreateAdmin("admin@example.com")
	require.Equal(t, http.StatusOK, f.CreateBackend(adminCookie, &api.ApiBackend{
		Fqdn: kioskHost, UpstreamUrl: "http://localhost:1", AllowKioskSignin: true,
	}).Code)
	require.Equal(t, http.StatusOK, f.CreateBackend(adminCookie, &api.ApiBackend{
		Fqdn: noKioskHost, UpstreamUrl: "http://localhost:1",
	}).Code)

	user, _, err := f.Auth.CreateUser("approver@example.com", "Approver", false, []string{kioskHost, noKioskHost})
	require.NoError(t, err)
	sess, err := f.Auth.CreateSession(user.Id, "user-agent", "remote-addr")
	require.NoError(t, err)
	cookie := f.Session.CreateSessionCookie(sess)

	resp, err := f.StartEnroll(cookie)
	require.NoError(t, err)
	cred, res := f.GenerateCredential(resp.EnrollRequest)
	_, err = f.FinishEnroll(cookie, resp.EnrollRequest.Token, res)
	require.NoError(t, err)

	return &kioskFixture{f, adminCookie, cookie, user.Id, resp.EnrollRequest, cred}
}

func (f *kioskFixture) requestKiosk(t *testing.T, redirect string) *api.ApiRequestKioskSigninResponse {
	t.Helper()
	resp := &api.ApiRequestKioskSigninResponse{}
	rr := f.request("POST", "/api/kiosk/request", &api.ApiRequestKioskSigninRequest{Redirect: redirect}, nil, resp)
	require.Equal(t, http.StatusOK, rr.Code)
	return resp
}

func (f *kioskFixture) pollKiosk(t *testing.T, id string) *api.ApiPollKioskSigninResponse {
	t.Helper()
	resp := &api.ApiPollKioskSigninResponse{}
	rr := f.request("POST", "/api/kiosk/poll", &api.ApiPollKioskSigninRequest{ID: id}, nil, resp)
	require.Equal(t, http.StatusOK, rr.Code)
	return resp
}

func (f *kioskFixture) queryKiosk(t *testing.T, cookie *http.Cookie, code string) *api.ApiQueryKioskSigninResponse {
	t.Helper()
	resp := &api.ApiQueryKioskSigninResponse{}
	rr := f.request("POST", "/api/kiosk/query", &api.ApiQueryKioskSigninRequest{Code: code}, cookie, resp)
	require.Equal(t, http.StatusOK, rr.Code)
	return resp
}

func (f *kioskFixture) approveKiosk(t *testing.T, query *api.ApiQueryKioskSigninResponse, name string, duration time.Duration) *api.ApiApproveKioskSigninResponse {
	t.Helper()
	assertion := f.SignAssertionRequest(query.AssertionRequest, f.enrollReq.Options.User.ID, &f.cred)
	resp := &api.ApiApproveKioskSigninResponse{}
	rr := f.request("POST", "/api/kiosk/approve", &api.ApiApproveKioskSigninRequest{
		Token:           query.Token,
		Credential:      *assertion,
		Name:            name,
		DurationSeconds: int64(duration / time.Second),
	}, f.approverCookie, resp)
	require.Equal(t, http.StatusOK, rr.Code)
	return resp
}

// signinKiosk goes through the whole flow, and returns the kiosk's session.
func (f *kioskFixture) signinKiosk(t *testing.T, duration time.Duration) *models.Session {
	t.Helper()
	req := f.requestKiosk(t, "https://"+kioskHost+"/dashboard?x=1")
	require.NotNil(t, req.Success)
	query := f.queryKiosk(t, f.approverCookie, req.Success.Code)
	require.Nil(t, query.Error)
	require.Nil(t, f.approveKiosk(t, query, "Kitchen tablet", duration).Error)
	poll := f.pollKiosk(t, req.Success.ID)
	require.NotEmpty(t, poll.Redirect)

	u, err := url.Parse(poll.Redirect)
	require.NoError(t, err)
	_, sess, err := f.Session.ConsumeTrampolineToken(u.Query().Get(session.TrampolineParam), kioskHost)
	require.NoError(t, err)
	return sess
}

func TestKioskSignin(t *testing.T) {
	f := setupKiosk(t)

	req := f.requestKiosk(t, "https://"+kioskHost+"/dashboard?x=1")
	require.Nil(t, req.Error)
	require.NotNil(t, req.Success)
	assert.Len(t, req.Success.Code, 6)
	assert.Equal(t, "https://test.example.com/kiosk/", req.Success.ConfirmUrl)
	assert.NotEmpty(t, req.Success.QrCodeUrl)

	assert.True(t, f.pollKiosk(t, req.Success.ID).Pending)

	// The code can be entered with separators.
	query := f.queryKiosk(t, f.approverCookie, req.Success.Code[:3]+" "+req.Success.Code[3:])
	require.Nil(t, query.Error)
	assert.Equal(t, kioskHost, query.Host)
	assert.NotEmpty(t, query.Token)
	require.NotNil(t, query.AssertionRequest)

	// Still pending until it's approved with the passkey.
	assert.True(t, f.pollKiosk(t, req.Success.ID).Pending)
	require.Nil(t, f.approveKiosk(t, query, " Kitchen tablet ", 24*time.Hour).Error)

	poll := f.pollKiosk(t, req.Success.ID)
	u, err := url.Parse(poll.Redirect)
	require.NoError(t, err)
	assert.Equal(t, kioskHost, u.Host)
	assert.Equal(t, "/dashboard", u.Path)
	assert.Equal(t, "1", u.Query().Get("x"))
	token := u.Query().Get(session.TrampolineParam)
	require.NotEmpty(t, token)

	// The session can only be taken once.
	assert.True(t, f.pollKiosk(t, req.Success.ID).Expired)

	// The token is only valid on the kiosk's backend.
	user, sess, err := f.Session.ConsumeTrampolineToken(token, kioskHost)
	require.NoError(t, err)
	assert.Equal(t, f.approverId, user.Id)
	assert.Equal(t, kioskHost, sess.RestrictedHost)
	assert.Equal(t, "Kitchen tablet", sess.Name)
	require.NotNil(t, sess.ExpiresAt)
	assert.WithinDuration(t, time.Now().Add(24*time.Hour), sess.ExpiresAt.AsTime(), time.Minute)

	// It's listed among the approver's sessions.
	me := f.getUser(f.approverCookie, "me")
	var listed *api.ApiSession
	for i := range me.Sessions {
		if me.Sessions[i].ID == sess.Id {
			listed = &me.Sessions[i]
		}
	}
	require.NotNil(t, listed)
	assert.Equal(t, kioskHost, listed.RestrictedHost)
	assert.Equal(t, "Kitchen tablet", listed.Name)
	assert.NotEmpty(t, listed.ExpiresAt)
}

func TestKioskSessionCantUseAdminApi(t *testing.T) {
	f := setupKiosk(t)
	kioskCookie := f.Session.CreateSessionCookie(f.signinKiosk(t, 0))

	for _, endpoint := range []struct{ method, path string }{
		{"GET", "/api/user/me"},
		{"POST", "/api/enroll/start"},
		{"POST", "/api/kiosk/query"},
		{"GET", "/api/backend"},
	} {
		rr := f.request(endpoint.method, endpoint.path, struct{}{}, kioskCookie, nil)
		assert.Equal(t, http.StatusForbidden, rr.Code, "%s %s", endpoint.method, endpoint.path)
	}
}

func TestKioskSessionWithoutDurationDoesntExpire(t *testing.T) {
	f := setupKiosk(t)
	sess := f.signinKiosk(t, 0)
	assert.Nil(t, sess.ExpiresAt)
	assert.Equal(t, kioskHost, sess.RestrictedHost)
}

func TestKioskRequestRejectsInvalidRedirects(t *testing.T) {
	f := setupKiosk(t)
	for _, rd := range []string{
		"",
		"https://" + noKioskHost + "/",
		"https://unknown.example.com/",
		"https://test.example.com/",
		"http://" + kioskHost + "/",
		"https://user@" + kioskHost + "/",
		"//" + kioskHost + "/",
		"javascript:alert(1)",
	} {
		t.Run(rd, func(t *testing.T) {
			resp := f.requestKiosk(t, rd)
			assert.Nil(t, resp.Success)
			require.NotNil(t, resp.Error)
			assert.True(t, resp.Error.NotAllowed)
		})
	}
}

func TestKioskPollUnknownRequest(t *testing.T) {
	f := setupKiosk(t)
	assert.True(t, f.pollKiosk(t, "unknown").Expired)
}

func TestKioskQueryRequiresSignin(t *testing.T) {
	f := setupKiosk(t)
	req := f.requestKiosk(t, "https://"+kioskHost+"/")
	rr := f.request("POST", "/api/kiosk/query", &api.ApiQueryKioskSigninRequest{Code: req.Success.Code}, nil, nil)
	assert.Equal(t, http.StatusForbidden, rr.Code)
}

func TestKioskQueryInvalidCode(t *testing.T) {
	f := setupKiosk(t)
	req := f.requestKiosk(t, "https://"+kioskHost+"/")
	wrong := "000000"
	if req.Success.Code == wrong {
		wrong = "000001"
	}
	query := f.queryKiosk(t, f.approverCookie, wrong)
	require.NotNil(t, query.Error)
	assert.True(t, query.Error.InvalidCode)
	assert.Empty(t, query.Token)
}

func TestKioskQueryRequiresAccessToBackend(t *testing.T) {
	f := setupKiosk(t)
	otherCookie, _ := f.CreateUser("other@example.com")
	req := f.requestKiosk(t, "https://"+kioskHost+"/")

	query := f.queryKiosk(t, otherCookie, req.Success.Code)
	require.NotNil(t, query.Error)
	assert.True(t, query.Error.NotAllowed)
	assert.Empty(t, query.Host)
}

func TestKioskApprovalCantBeReplayed(t *testing.T) {
	f := setupKiosk(t)
	req := f.requestKiosk(t, "https://"+kioskHost+"/")
	query := f.queryKiosk(t, f.approverCookie, req.Success.Code)
	require.Nil(t, f.approveKiosk(t, query, "", 0).Error)

	resp := f.approveKiosk(t, query, "", 0)
	require.NotNil(t, resp.Error)
	assert.True(t, resp.Error.InvalidRequest)
}

func TestKioskApprovalRejectsInvalidDuration(t *testing.T) {
	f := setupKiosk(t)
	req := f.requestKiosk(t, "https://"+kioskHost+"/")
	query := f.queryKiosk(t, f.approverCookie, req.Success.Code)

	resp := f.approveKiosk(t, query, "", -time.Second)
	require.NotNil(t, resp.Error)
	assert.True(t, resp.Error.InvalidRequest)
	assert.True(t, f.pollKiosk(t, req.Success.ID).Pending)
}

func TestKioskApprovalMustBeFromSameSession(t *testing.T) {
	f := setupKiosk(t)
	req := f.requestKiosk(t, "https://"+kioskHost+"/")
	query := f.queryKiosk(t, f.approverCookie, req.Success.Code)

	// Another session of the same user.
	otherSession, err := f.Auth.CreateSession(f.approverId, "user-agent", "remote-addr")
	require.NoError(t, err)
	assertion := f.SignAssertionRequest(query.AssertionRequest, f.enrollReq.Options.User.ID, &f.cred)
	rr := f.request("POST", "/api/kiosk/approve", &api.ApiApproveKioskSigninRequest{
		Token:      query.Token,
		Credential: *assertion,
	}, f.Session.CreateSessionCookie(otherSession), nil)
	assert.Equal(t, http.StatusBadRequest, rr.Code)
	assert.True(t, f.pollKiosk(t, req.Success.ID).Pending)
}

func TestKioskSigninFailsIfKioskDisabledAfterApproval(t *testing.T) {
	f := setupKiosk(t)
	req := f.requestKiosk(t, "https://"+kioskHost+"/")
	query := f.queryKiosk(t, f.approverCookie, req.Success.Code)
	require.Nil(t, f.approveKiosk(t, query, "", 0).Error)

	disabled := false
	rr := f.request("POST", "/api/backend/"+kioskHost, &api.ApiUpdateBackendRequest{AllowKioskSignin: &disabled}, f.adminCookie, nil)
	require.Equal(t, http.StatusOK, rr.Code)

	assert.True(t, f.pollKiosk(t, req.Success.ID).Expired)
}
