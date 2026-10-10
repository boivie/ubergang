package proxy

import (
	"boivie/ubergang/server/auth"
	"boivie/ubergang/server/backends"
	"boivie/ubergang/server/db"
	"boivie/ubergang/server/log"
	"boivie/ubergang/server/models"
	"boivie/ubergang/server/session"
	"net/http"
	"net/http/httptest"
	"path"
	"strings"
	"testing"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const fqdn = "app.example.com"

// upstreamRecorder records what the upstream server received.
type upstreamRecorder struct {
	paths   []string
	headers []http.Header
}

// The proxy under test, routed the same way as in production, with the
// components needed to sign in a session for it.
type fixture struct {
	router   *mux.Router
	upstream *upstreamRecorder
	session  *session.SessionStore
	auth     *auth.Auth
}

// Sets up a proxy for a backend with the provided rules.
func setupProxy(t *testing.T, accessLevel models.AccessLevel, rules ...*models.AccessRule) *fixture {
	t.Helper()
	logger := log.NewLogger(log.Fields{})
	database, err := db.New(logger, path.Join(t.TempDir(), "test.db"))
	require.NoError(t, err)

	upstream := &upstreamRecorder{}
	upstreamServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upstream.paths = append(upstream.paths, r.URL.Path)
		upstream.headers = append(upstream.headers, r.Header.Clone())
		_, _ = w.Write([]byte("upstream"))
	}))
	t.Cleanup(upstreamServer.Close)

	err = database.UpdateBackend(fqdn, func(_ *models.Backend) (*models.Backend, error) {
		return &models.Backend{
			Fqdn:        fqdn,
			UpstreamUrl: upstreamServer.URL,
			AccessLevel: accessLevel,
			AccessRules: rules,
		}, nil
	})
	require.NoError(t, err)

	config := &models.Configuration{AdminFqdn: "admin.example.com"}
	store := session.NewSessionStore(logger, database)
	p := New(config, logger, store,
		make(chan *models.Session, 10), backends.New(database, logger), nil)

	r := mux.NewRouter()
	r.PathPrefix("/").HandlerFunc(p.ProxyHandler)
	return &fixture{router: r, upstream: upstream, session: store, auth: auth.New(logger, database)}
}

// signin creates a user allowed to access the backend and returns its session
// cookie.
func (f *fixture) signin(t *testing.T) *http.Cookie {
	t.Helper()
	user, _, err := f.auth.CreateUser("user@example.com", "User", false, []string{fqdn})
	require.NoError(t, err)
	session, err := f.auth.CreateSession(user.Id, "user-agent", "remote-addr")
	require.NoError(t, err)
	return f.session.CreateSessionCookie(session)
}

func do(f *fixture, method, target string) *httptest.ResponseRecorder {
	return doRequest(f, method, target, nil, nil)
}

func doRequest(f *fixture, method, target string, headers map[string]string, cookie *http.Cookie) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, "https://"+fqdn+target, nil)
	for key, value := range headers {
		req.Header.Set(key, value)
	}
	if cookie != nil {
		req.AddCookie(cookie)
	}
	rr := httptest.NewRecorder()
	f.router.ServeHTTP(rr, req)
	return rr
}

func rule(action models.AccessRuleAction, predicate string) *models.AccessRule {
	return &models.AccessRule{Action: action, Predicate: predicate}
}

func TestAccessRules(t *testing.T) {
	f := setupProxy(t, models.AccessLevel_NORMAL,
		rule(models.AccessRuleAction_ACCESS_RULE_PUBLIC, `request.path.startsWith("/public/")`),
		rule(models.AccessRuleAction_ACCESS_RULE_DENY, `request.path.startsWith("/blocked")`),
	)

	t.Run("public rule proxies without session", func(t *testing.T) {
		rr := do(f, "GET", "/public/file.txt")
		assert.Equal(t, http.StatusOK, rr.Code)
		assert.Equal(t, "upstream", rr.Body.String())
	})

	t.Run("deny rule returns forbidden", func(t *testing.T) {
		rr := do(f, "GET", "/blocked/x")
		assert.Equal(t, http.StatusForbidden, rr.Code)
	})

	t.Run("fallback requires authentication", func(t *testing.T) {
		rr := do(f, "GET", "/private")
		assert.Equal(t, http.StatusFound, rr.Code)
		assert.True(t, strings.HasPrefix(rr.Header().Get("Location"), "https://admin.example.com/authorize"))
	})

	// Path traversal must not use a public rule to reach a protected path.
	for _, target := range []string{
		"/public/../private",
		"/public/%2e%2e/private",
		"/public%2F..%2Fprivate",
		"/public/./../blocked/x",
	} {
		t.Run("traversal "+target, func(t *testing.T) {
			f.upstream.paths = nil
			rr := do(f, "GET", target)
			assert.NotEqual(t, http.StatusOK, rr.Code)
			assert.Empty(t, f.upstream.paths)
		})
	}
}

func TestAccessRuleFallbackPublic(t *testing.T) {
	f := setupProxy(t, models.AccessLevel_PUBLIC,
		rule(models.AccessRuleAction_ACCESS_RULE_AUTHENTICATED, `request.path.startsWith("/admin")`),
	)

	assert.Equal(t, http.StatusOK, do(f, "GET", "/anything").Code)
	assert.Equal(t, http.StatusFound, do(f, "GET", "/admin").Code)
}

func TestAccessRuleEvaluationErrorFailsClosed(t *testing.T) {
	f := setupProxy(t, models.AccessLevel_PUBLIC,
		rule(models.AccessRuleAction_ACCESS_RULE_DENY, `request.headers["x-missing"] == "1"`),
	)

	rr := do(f, "GET", "/")
	assert.Equal(t, http.StatusInternalServerError, rr.Code)
	assert.Empty(t, f.upstream.paths)
}

func TestInvalidStoredRuleFailsClosed(t *testing.T) {
	// Rules are validated when saved, but a broken rule in the database must
	// never make the backend reachable.
	f := setupProxy(t, models.AccessLevel_PUBLIC,
		rule(models.AccessRuleAction_ACCESS_RULE_DENY, `request.pth == "/"`),
	)

	rr := do(f, "GET", "/")
	assert.NotEqual(t, http.StatusOK, rr.Code)
	assert.Empty(t, f.upstream.paths)
}

// The headers that identify the client and the authenticated user must not be
// spoofable: the backend should only see what the proxy sets itself.
func TestForwardingHeadersAreNotSpoofable(t *testing.T) {
	spoofed := map[string]string{
		"X-Forwarded-Email": "ceo@example.com",
		"X-Forwarded-Name":  "The CEO",
		"X-Forwarded-For":   "203.0.113.1",
		"X-Forwarded-Host":  "spoofed.example.com",
		"X-Forwarded-Proto": "http",
		"Forwarded":         "for=203.0.113.1;proto=http",
		"X-Real-IP":         "203.0.113.1",
		"CF-Connecting-IP":  "203.0.113.1",
		"True-Client-IP":    "203.0.113.1",
	}

	t.Run("public request", func(t *testing.T) {
		// A public path is proxied without a user, so nothing overwrites the
		// client's own values - they must be dropped rather than passed on.
		f := setupProxy(t, models.AccessLevel_PUBLIC)
		rr := doRequest(f, "GET", "/public/file.txt", spoofed, nil)
		require.Equal(t, http.StatusOK, rr.Code)
		require.Len(t, f.upstream.headers, 1)
		got := f.upstream.headers[0]

		assert.Empty(t, got.Get("X-Forwarded-Email"))
		assert.Empty(t, got.Get("X-Forwarded-Name"))
		assert.Empty(t, got.Get("Forwarded"))
		assert.Empty(t, got.Get("X-Real-IP"))
		assert.Empty(t, got.Get("CF-Connecting-IP"))
		assert.Empty(t, got.Get("True-Client-IP"))
		// The proxy sets these itself, from the request it received.
		assert.Equal(t, fqdn, got.Get("X-Forwarded-Host"))
		assert.Equal(t, "https", got.Get("X-Forwarded-Proto"))
		// ReverseProxy appends the connection's address; the client-supplied
		// chain is gone.
		assert.Equal(t, "192.0.2.1", got.Get("X-Forwarded-For"))
	})

	t.Run("authenticated request", func(t *testing.T) {
		f := setupProxy(t, models.AccessLevel_NORMAL)
		cookie := f.signin(t)
		rr := doRequest(f, "GET", "/private", spoofed, cookie)
		require.Equal(t, http.StatusOK, rr.Code)
		require.Len(t, f.upstream.headers, 1)
		got := f.upstream.headers[0]

		// The signed-in user's identity replaces what the client sent.
		assert.Equal(t, "user@example.com", got.Get("X-Forwarded-Email"))
		assert.Equal(t, "User", got.Get("X-Forwarded-Name"))
		assert.Empty(t, got.Get("Forwarded"))
		assert.Empty(t, got.Get("X-Real-IP"))
		assert.Empty(t, got.Get("CF-Connecting-IP"))
		assert.Empty(t, got.Get("True-Client-IP"))
		assert.Equal(t, "192.0.2.1", got.Get("X-Forwarded-For"))
	})
}

func TestForwardedNameIsDroppedWithoutDisplayName(t *testing.T) {
	f := setupProxy(t, models.AccessLevel_NORMAL)
	user, _, err := f.auth.CreateUser("noname@example.com", "", false, []string{fqdn})
	require.NoError(t, err)
	session, err := f.auth.CreateSession(user.Id, "user-agent", "remote-addr")
	require.NoError(t, err)

	rr := doRequest(f, "GET", "/private",
		map[string]string{"X-Forwarded-Name": "The CEO"}, f.session.CreateSessionCookie(session))
	require.Equal(t, http.StatusOK, rr.Code)
	require.Len(t, f.upstream.headers, 1)
	got := f.upstream.headers[0]

	assert.Equal(t, "noname@example.com", got.Get("X-Forwarded-Email"))
	assert.Empty(t, got.Get("X-Forwarded-Name"))
}
