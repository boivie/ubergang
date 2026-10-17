package proxy

import (
	"boivie/ubergang/server/models"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func authorize(f *fixture, rd string, cookie *http.Cookie) *httptest.ResponseRecorder {
	req := httptest.NewRequest("GET", "https://admin.example.com/authorize?rd="+url.QueryEscape(rd), nil)
	if cookie != nil {
		req.AddCookie(cookie)
	}
	rr := httptest.NewRecorder()
	f.proxy.HandleAuthorize(rr, req)
	return rr
}

func trampoline(f *fixture, host, token string) *httptest.ResponseRecorder {
	req := httptest.NewRequest("GET", "https://"+host+"/some/path?a=b&"+trampolineParam+"="+token, nil)
	rr := httptest.NewRecorder()
	f.router.ServeHTTP(rr, req)
	return rr
}

// requestURI returns the path and query of the response's redirect.
func requestURI(t *testing.T, rr *httptest.ResponseRecorder) string {
	t.Helper()
	u, err := url.Parse(rr.Header().Get("Location"))
	require.NoError(t, err)
	return u.RequestURI()
}

func TestAuthorizeRedirectsToBackendWithToken(t *testing.T) {
	f := setupProxy(t, models.AccessLevel_NORMAL)
	session := f.signinSession(t)

	rr := authorize(f, "https://"+fqdn+"/some/path?a=b", f.session.CreateSessionCookie(session))
	require.Equal(t, http.StatusFound, rr.Code)
	location := rr.Header().Get("Location")
	u, err := url.Parse(location)
	require.NoError(t, err)

	assert.Equal(t, fqdn, u.Host)
	assert.Equal(t, "/some/path", u.Path)
	assert.Equal(t, "b", u.Query().Get("a"))
	assert.NotEmpty(t, u.Query().Get(trampolineParam))
	// The session itself is never put in the URL.
	assert.NotContains(t, location, session.Secret)
}

func TestAuthorizeRejectsInvalidRedirects(t *testing.T) {
	for _, rd := range []string{
		"https://evil.example.com/",
		"https://" + fqdn + ".evil.example.com/",
		"http://" + fqdn + "/",
		"https://user@" + fqdn + "/",
		"//" + fqdn + "/",
		"javascript:alert(1)",
	} {
		t.Run(rd, func(t *testing.T) {
			f := setupProxy(t, models.AccessLevel_NORMAL)
			rr := authorize(f, rd, f.signin(t))
			assert.Equal(t, http.StatusBadRequest, rr.Code)
			assert.Empty(t, rr.Header().Get("Location"))
		})
	}
}

func TestTrampolineSetsCookieOnce(t *testing.T) {
	f := setupProxy(t, models.AccessLevel_NORMAL)
	session := f.signinSession(t)
	token, err := f.session.CreateTrampolineToken(session, fqdn)
	require.NoError(t, err)

	rr := trampoline(f, fqdn, token)
	require.Equal(t, http.StatusFound, rr.Code)
	assert.Equal(t, "/some/path?a=b", requestURI(t, rr))
	cookies := rr.Result().Cookies()
	require.Len(t, cookies, 1)
	assert.Equal(t, f.session.EncodeSessionCookie(session), cookies[0].Value)
	assert.True(t, cookies[0].HttpOnly)
	assert.True(t, cookies[0].Secure)

	// The token can't be used again.
	rr = trampoline(f, fqdn, token)
	require.Equal(t, http.StatusFound, rr.Code)
	assert.Equal(t, "/some/path?a=b", requestURI(t, rr))
	assert.Empty(t, rr.Result().Cookies())
}

func TestTrampolineIgnoresPortInHost(t *testing.T) {
	f := setupProxy(t, models.AccessLevel_NORMAL)
	token, err := f.session.CreateTrampolineToken(f.signinSession(t), fqdn)
	require.NoError(t, err)

	rr := trampoline(f, fqdn+":10443", token)
	require.Equal(t, http.StatusFound, rr.Code)
	assert.Len(t, rr.Result().Cookies(), 1)
}

func TestTrampolineTokenIsBoundToHost(t *testing.T) {
	f := setupProxy(t, models.AccessLevel_NORMAL)
	token, err := f.session.CreateTrampolineToken(f.signinSession(t), fqdn)
	require.NoError(t, err)

	rr := trampoline(f, "other.example.com", token)
	require.Equal(t, http.StatusFound, rr.Code)
	assert.Empty(t, rr.Result().Cookies())
}

func TestTrampolineTokenExpires(t *testing.T) {
	f := setupProxy(t, models.AccessLevel_NORMAL)
	session := f.signinSession(t)
	err := f.db.StoreTrampolineToken("expired", &models.TrampolineToken{
		SessionId: session.Id,
		Host:      fqdn,
		ExpiresAt: timestamppb.New(time.Now().Add(-time.Second)),
	})
	require.NoError(t, err)

	rr := trampoline(f, fqdn, "expired")
	require.Equal(t, http.StatusFound, rr.Code)
	assert.Empty(t, rr.Result().Cookies())
}

func TestSessionCookieIsNotForwarded(t *testing.T) {
	t.Run("among other cookies", func(t *testing.T) {
		f := setupProxy(t, models.AccessLevel_PUBLIC)
		rr := doRequest(f, "GET", "/", map[string]string{"Cookie": "a=1; __ug_sess=id:secret; b=2"}, nil)
		require.Equal(t, http.StatusOK, rr.Code)
		require.Len(t, f.upstream.headers, 1)
		assert.Equal(t, []string{"a=1; b=2"}, f.upstream.headers[0].Values("Cookie"))
	})

	t.Run("as the only cookie", func(t *testing.T) {
		f := setupProxy(t, models.AccessLevel_NORMAL)
		rr := doRequest(f, "GET", "/private", nil, f.signin(t))
		require.Equal(t, http.StatusOK, rr.Code)
		require.Len(t, f.upstream.headers, 1)
		assert.Empty(t, f.upstream.headers[0].Values("Cookie"))
	})
}
