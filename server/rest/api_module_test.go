package rest

import (
	"boivie/ubergang/server/models"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestApiModule(t *testing.T) {
	t.Run("creates new API module successfully", func(t *testing.T) {
		f := CreateFixture(t)

		assert.NotNil(t, f.Db)
		// The API module is created inside CreateFixture via New()
		// We can't directly access it, but we can test that endpoints are registered

		// Test that a known endpoint responds (even if with authentication error)
		req, err := http.NewRequest("GET", "/api/user/me", nil)
		require.NoError(t, err)
		req.Host = "test.example.com"

		rr := httptest.NewRecorder()
		f.router.ServeHTTP(rr, req)

		// Should get some response (not 404), indicating the endpoint is registered
		assert.NotEqual(t, http.StatusNotFound, rr.Code)
	})

	t.Run("registers key GET endpoints", func(t *testing.T) {
		f := CreateFixture(t)

		// Test only GET endpoints that should work without specific setup
		endpoints := []struct {
			method string
			path   string
		}{
			{"GET", "/api/signin/start"},
			{"GET", "/api/backend"},
			{"GET", "/api/user/me"},
		}

		for _, endpoint := range endpoints {
			req, err := http.NewRequest(endpoint.method, endpoint.path, nil)
			require.NoError(t, err, "Failed to create request for %s %s", endpoint.method, endpoint.path)
			req.Host = "test.example.com"

			rr := httptest.NewRecorder()
			f.router.ServeHTTP(rr, req)

			// Should not return 404 (not found), indicating the endpoint is registered
			assert.NotEqual(t, http.StatusNotFound, rr.Code,
				"Endpoint %s %s should be registered but returned 404", endpoint.method, endpoint.path)
		}
	})

	t.Run("requires correct host for all endpoints", func(t *testing.T) {
		f := CreateFixture(t)

		// Test that endpoints only respond to the correct host
		req, err := http.NewRequest("GET", "/api/user/me", nil)
		require.NoError(t, err)
		req.Host = "wrong.example.com" // Wrong host

		rr := httptest.NewRecorder()
		f.router.ServeHTTP(rr, req)

		// Should return 404 because host doesn't match
		assert.Equal(t, http.StatusNotFound, rr.Code)
	})

	t.Run("handles different HTTP methods correctly", func(t *testing.T) {
		f := CreateFixture(t)

		// Test that GET /api/backend works but POST /api/backend doesn't
		getReq, err := http.NewRequest("GET", "/api/backend", nil)
		require.NoError(t, err)
		getReq.Host = "test.example.com"

		getRr := httptest.NewRecorder()
		f.router.ServeHTTP(getRr, getReq)

		// GET should work (even if it requires auth)
		assert.NotEqual(t, http.StatusNotFound, getRr.Code)
		assert.NotEqual(t, http.StatusMethodNotAllowed, getRr.Code)

		// POST to same path should not work (method not allowed or not found)
		postReq, err := http.NewRequest("POST", "/api/backend", nil)
		require.NoError(t, err)
		postReq.Host = "test.example.com"

		postRr := httptest.NewRecorder()
		f.router.ServeHTTP(postRr, postReq)

		// POST should fail (404 or 405)
		assert.True(t, postRr.Code == http.StatusNotFound || postRr.Code == http.StatusMethodNotAllowed)
	})

	t.Run("preserves configuration in module", func(t *testing.T) {
		// This test verifies that the configuration is properly passed through
		// We can't directly access the ApiModule instance, but we can verify
		// that the admin FQDN from the configuration is used correctly

		f := CreateFixture(t)

		// The fixture uses "test.example.com" as AdminFQDN
		req, err := http.NewRequest("GET", "/api/user/me", nil)
		require.NoError(t, err)
		req.Host = "test.example.com"

		rr := httptest.NewRecorder()
		f.router.ServeHTTP(rr, req)

		// Should respond (not 404), confirming the config FQDN is used
		assert.NotEqual(t, http.StatusNotFound, rr.Code)
	})

	t.Run("module components are properly initialized", func(t *testing.T) {
		// Test that we can create a module with all required dependencies
		f := CreateFixture(t)

		// Verify the fixture components are initialized
		assert.NotNil(t, f.Session)
		assert.NotNil(t, f.Auth)
		assert.NotNil(t, f.Db)

		// Test that we can perform basic operations that require all components
		cookie, _ := f.CreateUser("test@example.com")
		assert.NotNil(t, cookie)

		user := f.getUser(cookie, "me")
		assert.Equal(t, "test@example.com", user.Email)
	})

	t.Run("webauthn integration is initialized", func(t *testing.T) {
		f := CreateFixture(t)
		cookie, _ := f.CreateUser("test@example.com")

		// Test that WebAuthn functionality works (this requires the WA module)
		enrollResp, err := f.StartEnroll(cookie)
		require.NoError(t, err)

		assert.NotNil(t, enrollResp.EnrollRequest)
		assert.NotEmpty(t, enrollResp.EnrollRequest.Token)
		assert.NotEmpty(t, enrollResp.EnrollRequest.Options.Challenge)
	})
}

func TestApiModuleNew(t *testing.T) {
	t.Run("creates module with valid dependencies", func(t *testing.T) {
		// This test verifies the New function works correctly
		// We use the same setup as CreateFixture to test module creation

		config := &models.Configuration{
			AdminFqdn: "test.example.com",
		}

		f := CreateFixture(t)

		// Create a new API module using the same components
		apiModule := New(config, f.Db, nil, f.Session, f.Auth, &FakeMqttConnectionTracker{})

		assert.NotNil(t, apiModule)

		// Test that we can register endpoints on a new router
		router := mux.NewRouter()
		apiModule.RegisterEndpoints(router)

		// Test that endpoints are registered
		req, err := http.NewRequest("GET", "/api/user/me", nil)
		require.NoError(t, err)
		req.Host = "test.example.com"

		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		// Should not return 404, indicating endpoints are registered
		assert.NotEqual(t, http.StatusNotFound, rr.Code)
	})
}
