package rest

import (
	"boivie/ubergang/server/api"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

// setupAndCreateCredential is a helper to reduce boilerplate. It creates a
// fixture, a user, and a credential, returning the fixture, the user's session
// cookie, and the created credential info. This ensures each test runs with a
// fresh, isolated state.
func setupAndCreateCredential(t *testing.T) (*Fixture, *http.Cookie, api.ApiCredential) {
	t.Helper()
	f := CreateFixture(t)
	cookie, _ := f.CreateUser("test")

	request, err := f.StartEnroll(cookie)
	require.NoError(t, err)
	_, res := f.GenerateCredential(request.EnrollRequest)
	f.FinishEnroll(cookie, request.EnrollRequest.Token, res)

	user := f.getUser(cookie, "me")
	if len(user.Credentials) != 1 {
		t.Fatalf("Expected 1 credential to be created, but found %d", len(user.Credentials))
	}
	cred := user.Credentials[0]
	if cred.Name != "Unnamed passkey" {
		t.Fatalf("Expected initial credential name to be 'Unnamed passkey', but got %q", cred.Name)
	}
	return f, cookie, cred
}

func TestUpdateCredential(t *testing.T) {
	t.Run("update name", func(t *testing.T) {
		f, cookie, cred := setupAndCreateCredential(t)

		updatedName := "Updated"
		jsonReq := &api.ApiUpdateCredentialRequest{
			Name: &updatedName,
		}
		resp := &api.ApiUpdateCredentialResponse{}
		rr := f.request("POST", "/api/credential/"+cred.ID, jsonReq, cookie, resp)
		if rr.Code != http.StatusOK {
			t.Fatalf("request failed with status %d: %s", rr.Code, rr.Body.String())
		}

		user := f.getUser(cookie, "me")
		if len(user.Credentials) != 1 {
			t.Fatalf("Expected 1 credential after update, got %d", len(user.Credentials))
		}
		cred = user.Credentials[0]
		if cred.Name != updatedName {
			t.Errorf("Expected name to be updated to %q, but got %q", updatedName, cred.Name)
		}
	})

	t.Run("update non-existent credential", func(t *testing.T) {
		f := CreateFixture(t)
		cookie, _ := f.CreateUser("test")

		updatedName := "Updated"
		jsonReq := &api.ApiUpdateCredentialRequest{
			Name: &updatedName,
		}
		resp := &api.ApiUpdateCredentialResponse{}
		rr := f.request("POST", "/api/credential/non-existent-id", jsonReq, cookie, resp)

		if rr.Code != http.StatusBadRequest {
			t.Errorf("Expected status %d, but got %d: %s", http.StatusBadRequest, rr.Code, rr.Body.String())
		}
	})

	t.Run("update credential of another user", func(t *testing.T) {
		f, cookieA, credA := setupAndCreateCredential(t)
		cookieB, _ := f.CreateUser("another-user")

		updatedName := "Updated by another user"
		jsonReq := &api.ApiUpdateCredentialRequest{
			Name: &updatedName,
		}
		resp := &api.ApiUpdateCredentialResponse{}
		rr := f.request("POST", "/api/credential/"+credA.ID, jsonReq, cookieB, resp)

		if rr.Code != http.StatusBadRequest {
			t.Errorf("Expected status %d for authorization failure, but got %d: %s", http.StatusBadRequest, rr.Code, rr.Body.String())
		}

		// Verify that the credential was not changed
		userA := f.getUser(cookieA, "me")
		if len(userA.Credentials) != 1 {
			t.Fatal("User A should still have 1 credential")
		}
		if userA.Credentials[0].Name == updatedName {
			t.Error("Credential name was updated by another user, which should not happen")
		}
	})
}
