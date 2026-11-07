package proxy

import (
	"boivie/ubergang/server/models"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const otherFqdn = "other.example.com"

// setupKioskProxy sets up the proxy for `fqdn`, with kiosk sign-in allowed,
// and for `otherFqdn`, with the same upstream and kiosk sign-in allowed too.
func setupKioskProxy(t *testing.T) *fixture {
	t.Helper()
	f := setupProxy(t, models.AccessLevel_NORMAL)
	f.setAllowKioskSignin(t, fqdn, true)
	be, err := f.db.GetBackend(fqdn)
	require.NoError(t, err)
	err = f.db.UpdateBackend(otherFqdn, func(_ *models.Backend) (*models.Backend, error) {
		return &models.Backend{
			Fqdn:             otherFqdn,
			UpstreamUrl:      be.UpstreamUrl,
			AccessLevel:      models.AccessLevel_NORMAL,
			AllowKioskSignin: true,
		}, nil
	})
	require.NoError(t, err)
	return f
}

func (f *fixture) setAllowKioskSignin(t *testing.T, host string, allow bool) {
	t.Helper()
	err := f.db.UpdateBackend(host, func(old *models.Backend) (*models.Backend, error) {
		old.AllowKioskSignin = allow
		return old, nil
	})
	require.NoError(t, err)
}

// kioskSession creates a user allowed to access both backends, and a kiosk
// session for it that is restricted to `fqdn`.
func (f *fixture) kioskSession(t *testing.T, expiresAt time.Time) *models.Session {
	t.Helper()
	user, _, err := f.auth.CreateUser("user@example.com", "User", false, []string{fqdn, otherFqdn})
	require.NoError(t, err)
	session, err := f.auth.CreateKioskSession(user.Id, "kiosk", "192.0.2.1", fqdn, "Kiosk", expiresAt)
	require.NoError(t, err)
	return session
}

func doOnHost(f *fixture, host string, cookie *http.Cookie) *httptest.ResponseRecorder {
	req := httptest.NewRequest("GET", "https://"+host+"/private", nil)
	req.AddCookie(cookie)
	rr := httptest.NewRecorder()
	f.router.ServeHTTP(rr, req)
	return rr
}

func isRedirectToAuthorize(rr *httptest.ResponseRecorder) bool {
	return rr.Code == http.StatusFound &&
		strings.HasPrefix(rr.Header().Get("Location"), "https://admin.example.com/authorize?")
}

func TestKioskSessionIsValidOnItsBackend(t *testing.T) {
	f := setupKioskProxy(t)
	cookie := f.session.CreateSessionCookie(f.kioskSession(t, time.Time{}))

	rr := doOnHost(f, fqdn, cookie)
	assert.Equal(t, http.StatusOK, rr.Code)
	require.Len(t, f.upstream.headers, 1)
	assert.Equal(t, "user@example.com", f.upstream.headers[0].Get("X-Forwarded-Email"))
}

func TestKioskSessionIsNotValidOnOtherBackends(t *testing.T) {
	f := setupKioskProxy(t)
	cookie := f.session.CreateSessionCookie(f.kioskSession(t, time.Time{}))

	assert.True(t, isRedirectToAuthorize(doOnHost(f, otherFqdn, cookie)))
	assert.Empty(t, f.upstream.paths)
}

func TestKioskSessionIsNotValidIfKioskSigninIsTurnedOff(t *testing.T) {
	f := setupKioskProxy(t)
	cookie := f.session.CreateSessionCookie(f.kioskSession(t, time.Time{}))
	f.setAllowKioskSignin(t, fqdn, false)

	assert.True(t, isRedirectToAuthorize(doOnHost(f, fqdn, cookie)))
	assert.Empty(t, f.upstream.paths)
}

func TestKioskSessionExpires(t *testing.T) {
	f := setupKioskProxy(t)
	session := f.kioskSession(t, time.Now().Add(time.Hour))
	cookie := f.session.CreateSessionCookie(session)
	assert.WithinDuration(t, session.ExpiresAt.AsTime(), cookie.Expires, time.Second)
	require.Equal(t, http.StatusOK, doOnHost(f, fqdn, cookie).Code)

	err := f.db.UpdateSession(session.Id, func(old *models.Session) (*models.Session, error) {
		old.ExpiresAt = timestamppb.New(time.Now().Add(-time.Second))
		return old, nil
	})
	require.NoError(t, err)
	assert.True(t, isRedirectToAuthorize(doOnHost(f, fqdn, cookie)))
}

func TestKioskSessionCantAuthorizeOtherBackends(t *testing.T) {
	f := setupKioskProxy(t)
	cookie := f.session.CreateSessionCookie(f.kioskSession(t, time.Time{}))

	// On the admin host, the kiosk session isn't valid, so /authorize sends
	// it to sign in instead of handing out a trampoline token.
	rr := authorize(f, "https://"+otherFqdn+"/", cookie)
	require.Equal(t, http.StatusFound, rr.Code)
	assert.True(t, strings.HasPrefix(rr.Header().Get("Location"), "https://admin.example.com/signin?"))
}

func TestKioskSessionTrampolineIsBoundToItsBackend(t *testing.T) {
	f := setupKioskProxy(t)
	session := f.kioskSession(t, time.Time{})

	// Even if a token were issued for another host, the session can't be
	// used there.
	token, err := f.session.CreateTrampolineToken(session, otherFqdn)
	require.NoError(t, err)
	rr := trampoline(f, otherFqdn, token)
	require.Equal(t, http.StatusFound, rr.Code)
	assert.Empty(t, rr.Result().Cookies())
}
