package proxy

import (
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

// Sets up a proxy for a backend with the provided rules, routed the same way as
// in production. Returns the router and a pointer to the paths that reached
// the upstream.
func setupProxy(t *testing.T, accessLevel models.AccessLevel, rules ...*models.AccessRule) (*mux.Router, *[]string) {
	t.Helper()
	logger := log.NewLogger(log.Fields{})
	database, err := db.New(logger, path.Join(t.TempDir(), "test.db"))
	require.NoError(t, err)

	reached := []string{}
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reached = append(reached, r.URL.Path)
		_, _ = w.Write([]byte("upstream"))
	}))
	t.Cleanup(upstream.Close)

	err = database.UpdateBackend(fqdn, func(_ *models.Backend) (*models.Backend, error) {
		return &models.Backend{
			Fqdn:        fqdn,
			UpstreamUrl: upstream.URL,
			AccessLevel: accessLevel,
			AccessRules: rules,
		}, nil
	})
	require.NoError(t, err)

	config := &models.Configuration{AdminFqdn: "admin.example.com"}
	p := New(config, logger, session.NewSessionStore(logger, database),
		make(chan *models.Session, 10), backends.New(database, logger), nil)

	r := mux.NewRouter()
	r.PathPrefix("/").HandlerFunc(p.ProxyHandler)
	return r, &reached
}

func do(r *mux.Router, method, target string) *httptest.ResponseRecorder {
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, httptest.NewRequest(method, "https://"+fqdn+target, nil))
	return rr
}

func rule(action models.AccessRuleAction, predicate string) *models.AccessRule {
	return &models.AccessRule{Action: action, Predicate: predicate}
}

func TestAccessRules(t *testing.T) {
	r, reached := setupProxy(t, models.AccessLevel_NORMAL,
		rule(models.AccessRuleAction_ACCESS_RULE_PUBLIC, `request.path.startsWith("/public/")`),
		rule(models.AccessRuleAction_ACCESS_RULE_DENY, `request.path.startsWith("/blocked")`),
	)

	t.Run("public rule proxies without session", func(t *testing.T) {
		rr := do(r, "GET", "/public/file.txt")
		assert.Equal(t, http.StatusOK, rr.Code)
		assert.Equal(t, "upstream", rr.Body.String())
	})

	t.Run("deny rule returns forbidden", func(t *testing.T) {
		rr := do(r, "GET", "/blocked/x")
		assert.Equal(t, http.StatusForbidden, rr.Code)
	})

	t.Run("fallback requires authentication", func(t *testing.T) {
		rr := do(r, "GET", "/private")
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
			*reached = nil
			rr := do(r, "GET", target)
			assert.NotEqual(t, http.StatusOK, rr.Code)
			assert.Empty(t, *reached)
		})
	}
}

func TestAccessRuleFallbackPublic(t *testing.T) {
	r, _ := setupProxy(t, models.AccessLevel_PUBLIC,
		rule(models.AccessRuleAction_ACCESS_RULE_AUTHENTICATED, `request.path.startsWith("/admin")`),
	)

	assert.Equal(t, http.StatusOK, do(r, "GET", "/anything").Code)
	assert.Equal(t, http.StatusFound, do(r, "GET", "/admin").Code)
}

func TestAccessRuleEvaluationErrorFailsClosed(t *testing.T) {
	r, reached := setupProxy(t, models.AccessLevel_PUBLIC,
		rule(models.AccessRuleAction_ACCESS_RULE_DENY, `request.headers["x-missing"] == "1"`),
	)

	rr := do(r, "GET", "/")
	assert.Equal(t, http.StatusInternalServerError, rr.Code)
	assert.Empty(t, *reached)
}

func TestInvalidStoredRuleFailsClosed(t *testing.T) {
	// Rules are validated when saved, but a broken rule in the database must
	// never make the backend reachable.
	r, reached := setupProxy(t, models.AccessLevel_PUBLIC,
		rule(models.AccessRuleAction_ACCESS_RULE_DENY, `request.pth == "/"`),
	)

	rr := do(r, "GET", "/")
	assert.NotEqual(t, http.StatusOK, rr.Code)
	assert.Empty(t, *reached)
}
