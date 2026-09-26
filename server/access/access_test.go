package access

import (
	"boivie/ubergang/server/models"
	"context"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func rules(pairs ...any) []*models.AccessRule {
	var out []*models.AccessRule
	for i := 0; i < len(pairs); i += 2 {
		out = append(out, &models.AccessRule{
			Action:    pairs[i].(models.AccessRuleAction),
			Predicate: pairs[i+1].(string),
		})
	}
	return out
}

const (
	public = models.AccessRuleAction_ACCESS_RULE_PUBLIC
	authn  = models.AccessRuleAction_ACCESS_RULE_AUTHENTICATED
	deny   = models.AccessRuleAction_ACCESS_RULE_DENY
)

func TestFallbackWithoutRules(t *testing.T) {
	r := httptest.NewRequest("GET", "https://app.example.com/", nil)

	p, err := Compile(nil, models.AccessLevel_NORMAL)
	require.NoError(t, err)
	d, err := p.Evaluate(context.Background(), r)
	require.NoError(t, err)
	assert.Equal(t, Authenticated, d)

	p, err = Compile(nil, models.AccessLevel_ACCESS_LEVEL_UNSPECIFIED)
	require.NoError(t, err)
	d, err = p.Evaluate(context.Background(), r)
	require.NoError(t, err)
	assert.Equal(t, Authenticated, d)

	p, err = Compile(nil, models.AccessLevel_PUBLIC)
	require.NoError(t, err)
	d, err = p.Evaluate(context.Background(), r)
	require.NoError(t, err)
	assert.Equal(t, Public, d)
}

func TestFirstMatchingRuleWins(t *testing.T) {
	p, err := Compile(rules(
		public, `request.path == "/health" || request.path.startsWith("/static/")`,
		deny, `request.path.startsWith("/admin") && request.method != "GET"`,
		authn, `request.path.startsWith("/admin")`,
		public, `request.path.startsWith("/")`,
	), models.AccessLevel_NORMAL)
	require.NoError(t, err)

	tests := []struct {
		method string
		path   string
		want   Decision
	}{
		{"GET", "/health", Public},
		{"GET", "/static/app.js", Public},
		{"POST", "/admin/users", Deny},
		{"GET", "/admin/users", Authenticated},
		{"GET", "/anything", Public},
	}
	for _, tt := range tests {
		t.Run(tt.method+" "+tt.path, func(t *testing.T) {
			r := httptest.NewRequest(tt.method, "https://app.example.com"+tt.path, nil)
			d, err := p.Evaluate(context.Background(), r)
			require.NoError(t, err)
			assert.Equal(t, tt.want, d)
		})
	}
}

func TestFallsThroughToAccessLevel(t *testing.T) {
	rs := rules(deny, `request.path.startsWith("/admin")`)
	r := httptest.NewRequest("GET", "https://app.example.com/home", nil)

	p, err := Compile(rs, models.AccessLevel_PUBLIC)
	require.NoError(t, err)
	d, err := p.Evaluate(context.Background(), r)
	require.NoError(t, err)
	assert.Equal(t, Public, d)

	p, err = Compile(rs, models.AccessLevel_NORMAL)
	require.NoError(t, err)
	d, err = p.Evaluate(context.Background(), r)
	require.NoError(t, err)
	assert.Equal(t, Authenticated, d)
}

func TestRequestVariables(t *testing.T) {
	r := httptest.NewRequest("PUT", "https://App.Example.com:8443/a/b?token=abc&x=1&x=2", nil)
	r.Header.Set("X-Api-Key", "secret")
	r.RemoteAddr = "192.0.2.10:54321"

	tests := []string{
		`request.method == "PUT"`,
		`request.path == "/a/b"`,
		`request.host == "app.example.com"`,
		`request.headers["x-api-key"] == "secret"`,
		`"x-api-key" in request.headers`,
		`request.query.token == "abc"`,
		`request.query.x == "1"`,
		`request.remote_ip == "192.0.2.10"`,
		`request.path.matches("^/a/[a-z]+$")`,
		`request.path.lowerAscii() == "/a/b"`,
	}
	for _, predicate := range tests {
		t.Run(predicate, func(t *testing.T) {
			p, err := Compile(rules(public, predicate), models.AccessLevel_NORMAL)
			require.NoError(t, err)
			d, err := p.Evaluate(context.Background(), r)
			require.NoError(t, err)
			assert.Equal(t, Public, d)
		})
	}
}

func TestCompileErrors(t *testing.T) {
	tests := []struct {
		name  string
		rules []*models.AccessRule
	}{
		{"syntax error", rules(public, `request.path ==`)},
		{"unknown field", rules(public, `request.pth == "/"`)},
		{"unknown variable", rules(public, `user.email == "a@b"`)},
		{"not a bool", rules(public, `request.path`)},
		{"empty", rules(public, ``)},
		{"unspecified action", rules(models.AccessRuleAction_ACCESS_RULE_ACTION_UNSPECIFIED, `true`)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Compile(tt.rules, models.AccessLevel_NORMAL)
			assert.Error(t, err)
		})
	}
}

func TestErrorFailsClosed(t *testing.T) {
	// A missing map key is a runtime error. Evaluation must stop rather than
	// fall through to the public fallback.
	p, err := Compile(rules(
		deny, `request.headers["x-missing"] == "1"`,
	), models.AccessLevel_PUBLIC)
	require.NoError(t, err)

	r := httptest.NewRequest("GET", "https://app.example.com/", nil)
	d, err := p.Evaluate(context.Background(), r)
	assert.Error(t, err)
	assert.Equal(t, Deny, d)
}

func TestFixed(t *testing.T) {
	r := httptest.NewRequest("GET", "https://app.example.com/", nil)
	d, err := Fixed(Public).Evaluate(context.Background(), r)
	require.NoError(t, err)
	assert.Equal(t, Public, d)
}
