package rest

import (
	"boivie/ubergang/server/api"
	"net/http"
	"testing"
)

// setupBackendTest is a helper to reduce boilerplate. It creates a
// fixture and a user, returning the fixture and the user's session cookie.
func setupBackendTest(t *testing.T) (*Fixture, *http.Cookie) {
	t.Helper()
	f := CreateFixture(t)
	cookie, _ := f.CreateAdmin("test")
	return f, cookie
}

func TestUpdateBackend(t *testing.T) {
	t.Run("create backend", func(t *testing.T) {
		f, cookie := setupBackendTest(t)

		rr := f.CreateBackend(cookie, &api.ApiBackend{
			Fqdn:        "test.example.com",
			UpstreamUrl: "http://localhost:8080",
		})

		if rr.Code != http.StatusOK {
			t.Fatalf("request failed with status %d: %s", rr.Code, rr.Body.String())
		}

		backends := f.ListBackends(cookie)
		if len(backends) != 1 {
			t.Fatalf("Expected 1 backend, got %d", len(backends))
		}
		if backends[0].Fqdn != "test.example.com" {
			t.Errorf("Expected fqdn to be 'test.example.com', got %q", backends[0].Fqdn)
		}
		if backends[0].UpstreamUrl != "http://localhost:8080" {
			t.Errorf("Expected upstreamUrl to be 'http://localhost:8080', got %q", backends[0].UpstreamUrl)
		}
	})

	t.Run("update upstream url", func(t *testing.T) {
		f, cookie := setupBackendTest(t)
		rr := f.CreateBackend(cookie, &api.ApiBackend{
			Fqdn:        "test.example.com",
			UpstreamUrl: "http://localhost:8080",
		})
		if rr.Code != http.StatusOK {
			t.Fatalf("pre-condition: create backend failed with status %d: %s", rr.Code, rr.Body.String())
		}

		updatedURL := "http://new-upstream:9090"
		req := &api.ApiUpdateBackendRequest{
			UpstreamUrl: &updatedURL,
		}
		resp := &api.ApiUpdateBackendResponse{}
		rr = f.request("POST", "/api/backend/test.example.com", req, cookie, resp)

		if rr.Code != http.StatusOK {
			t.Fatalf("request failed with status %d: %s", rr.Code, rr.Body.String())
		}

		backends := f.ListBackends(cookie)
		if len(backends) != 1 {
			t.Fatalf("Expected 1 backend, got %d", len(backends))
		}
		if backends[0].UpstreamUrl != updatedURL {
			t.Errorf("Expected upstreamUrl to be updated to %q, got %q", updatedURL, backends[0].UpstreamUrl)
		}
	})

	t.Run("update access level", func(t *testing.T) {
		f, cookie := setupBackendTest(t)
		rr := f.CreateBackend(cookie, &api.ApiBackend{
			Fqdn:        "test.example.com",
			UpstreamUrl: "http://localhost:8080",
		})
		if rr.Code != http.StatusOK {
			t.Fatalf("pre-condition: create backend failed with status %d: %s", rr.Code, rr.Body.String())
		}

		// Update to PUBLIC
		updatedAccessLevel := "PUBLIC"
		req := &api.ApiUpdateBackendRequest{
			AccessLevel: &updatedAccessLevel,
		}
		resp := &api.ApiUpdateBackendResponse{}
		rr = f.request("POST", "/api/backend/test.example.com", req, cookie, resp)

		if rr.Code != http.StatusOK {
			t.Fatalf("request failed with status %d: %s", rr.Code, rr.Body.String())
		}

		backends := f.ListBackends(cookie)
		if len(backends) != 1 {
			t.Fatalf("Expected 1 backend, got %d", len(backends))
		}
		if backends[0].AccessLevel != "PUBLIC" {
			t.Errorf("Expected accessLevel to be 'PUBLIC', got %q", backends[0].AccessLevel)
		}

		// Update back to NORMAL
		updatedAccessLevel = "NORMAL"
		req = &api.ApiUpdateBackendRequest{
			AccessLevel: &updatedAccessLevel,
		}
		rr = f.request("POST", "/api/backend/test.example.com", req, cookie, resp)

		if rr.Code != http.StatusOK {
			t.Fatalf("request failed with status %d: %s", rr.Code, rr.Body.String())
		}

		backends = f.ListBackends(cookie)
		if backends[0].AccessLevel != "NORMAL" {
			t.Errorf("Expected accessLevel to be 'NORMAL', got %q", backends[0].AccessLevel)
		}
	})

	t.Run("create backend with access level", func(t *testing.T) {
		f, cookie := setupBackendTest(t)

		rr := f.CreateBackend(cookie, &api.ApiBackend{
			Fqdn:        "public.example.com",
			UpstreamUrl: "http://localhost:8080",
			AccessLevel: "PUBLIC",
		})

		if rr.Code != http.StatusOK {
			t.Fatalf("request failed with status %d: %s", rr.Code, rr.Body.String())
		}

		backends := f.ListBackends(cookie)
		if len(backends) != 1 {
			t.Fatalf("Expected 1 backend, got %d", len(backends))
		}
		if backends[0].AccessLevel != "PUBLIC" {
			t.Errorf("Expected accessLevel to be 'PUBLIC', got %q", backends[0].AccessLevel)
		}
	})

	t.Run("create backend with js script", func(t *testing.T) {
		f, cookie := setupBackendTest(t)

		jsScript := "console.log('test');"
		rr := f.CreateBackend(cookie, &api.ApiBackend{
			Fqdn:        "script.example.com",
			UpstreamUrl: "http://localhost:8080",
			JsScript:    jsScript,
		})

		if rr.Code != http.StatusOK {
			t.Fatalf("request failed with status %d: %s", rr.Code, rr.Body.String())
		}

		backends := f.ListBackends(cookie)
		if len(backends) != 1 {
			t.Fatalf("Expected 1 backend, got %d", len(backends))
		}
		if backends[0].JsScript != jsScript {
			t.Errorf("Expected jsScript to be %q, got %q", jsScript, backends[0].JsScript)
		}
	})

	t.Run("update js script", func(t *testing.T) {
		f, cookie := setupBackendTest(t)
		rr := f.CreateBackend(cookie, &api.ApiBackend{
			Fqdn:        "test.example.com",
			UpstreamUrl: "http://localhost:8080",
		})
		if rr.Code != http.StatusOK {
			t.Fatalf("pre-condition: create backend failed with status %d: %s", rr.Code, rr.Body.String())
		}

		// Verify initially no script
		backends := f.ListBackends(cookie)
		if backends[0].JsScript != "" {
			t.Errorf("Expected jsScript to be empty initially, got %q", backends[0].JsScript)
		}

		// Update with a script
		updatedScript := "function handler(req) { return req; }"
		req := &api.ApiUpdateBackendRequest{
			JsScript: updatedScript,
		}
		resp := &api.ApiUpdateBackendResponse{}
		rr = f.request("POST", "/api/backend/test.example.com", req, cookie, resp)

		if rr.Code != http.StatusOK {
			t.Fatalf("request failed with status %d: %s", rr.Code, rr.Body.String())
		}

		backends = f.ListBackends(cookie)
		if backends[0].JsScript != updatedScript {
			t.Errorf("Expected jsScript to be %q, got %q", updatedScript, backends[0].JsScript)
		}
	})

	t.Run("clear js script", func(t *testing.T) {
		f, cookie := setupBackendTest(t)

		// Create backend with script
		jsScript := "console.log('test');"
		rr := f.CreateBackend(cookie, &api.ApiBackend{
			Fqdn:        "test.example.com",
			UpstreamUrl: "http://localhost:8080",
			JsScript:    jsScript,
		})
		if rr.Code != http.StatusOK {
			t.Fatalf("pre-condition: create backend failed with status %d: %s", rr.Code, rr.Body.String())
		}

		// Verify script was set
		backends := f.ListBackends(cookie)
		if backends[0].JsScript != jsScript {
			t.Errorf("Expected jsScript to be %q, got %q", jsScript, backends[0].JsScript)
		}

		// Clear the script by setting it to empty string
		emptyScript := ""
		req := &api.ApiUpdateBackendRequest{
			JsScript: emptyScript,
		}
		resp := &api.ApiUpdateBackendResponse{}
		rr = f.request("POST", "/api/backend/test.example.com", req, cookie, resp)

		if rr.Code != http.StatusOK {
			t.Fatalf("request failed with status %d: %s", rr.Code, rr.Body.String())
		}

		backends = f.ListBackends(cookie)
		if backends[0].JsScript != "" {
			t.Errorf("Expected jsScript to be empty after clearing, got %q", backends[0].JsScript)
		}
	})
}
