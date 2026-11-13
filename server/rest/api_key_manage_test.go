package rest

import (
	"net/http"
	"strings"
	"testing"
)

func TestCreateAndListApiKeys(t *testing.T) {
	f := CreateFixture(t)
	admin, _ := f.CreateAdmin("admin@example.com")

	created, rr := f.CreateApiKey(admin, " provisioning ", "write")
	if rr.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	if !strings.HasPrefix(created.Token, "ug_"+created.ApiKey.Id+"_") {
		t.Errorf("Unexpected token format: %q", created.Token)
	}
	if created.ApiKey.Name != "provisioning" || created.ApiKey.Scope != "write" {
		t.Errorf("Unexpected key: %+v", created.ApiKey)
	}
	if created.ApiKey.CreatedBy != "admin@example.com" {
		t.Errorf("Unexpected createdBy: %q", created.ApiKey.CreatedBy)
	}

	keys := f.ListApiKeys(admin)
	if len(keys) != 1 || keys[0].Id != created.ApiKey.Id || keys[0].Scope != "write" {
		t.Fatalf("Unexpected keys: %+v", keys)
	}
	if keys[0].LastUsedAt != "" {
		t.Errorf("Expected key to be unused, got %q", keys[0].LastUsedAt)
	}

	// The secret must not be stored or returned.
	rr = f.request("GET", "/api/api-key", nil, admin, nil)
	secret := created.Token[strings.LastIndex(created.Token, "_")+1:]
	if strings.Contains(rr.Body.String(), secret) {
		t.Errorf("Listing API keys returned the secret")
	}
	stored, _ := f.Db.GetApiKey(created.ApiKey.Id)
	if strings.Contains(string(stored.SecretSha256), secret) {
		t.Errorf("Secret stored in the clear")
	}
}

func TestApiKeyLastUsed(t *testing.T) {
	f := CreateFixture(t)
	admin, _ := f.CreateAdmin("admin@example.com")
	created, _ := f.CreateApiKey(admin, "test", "read")

	f.publicRequest("GET", "/api/v1/mqtt/profiles", created.Token, nil, nil)

	keys := f.ListApiKeys(admin)
	if keys[0].LastUsedAt == "" {
		t.Errorf("Expected lastUsedAt to be set")
	}
}

func TestCreateApiKeyValidation(t *testing.T) {
	f := CreateFixture(t)
	admin, _ := f.CreateAdmin("admin@example.com")

	for _, tc := range []struct{ name, scope string }{
		{"", "read"},
		{"   ", "read"},
		{strings.Repeat("x", 101), "read"},
		{"test", ""},
		{"test", "ADMIN"},
		{"test", "READ"},
	} {
		_, rr := f.CreateApiKey(admin, tc.name, tc.scope)
		if rr.Code != http.StatusBadRequest {
			t.Errorf("name=%q scope=%q: expected 400, got %d", tc.name, tc.scope, rr.Code)
		}
	}
	if keys := f.ListApiKeys(admin); len(keys) != 0 {
		t.Errorf("Expected no keys, got %+v", keys)
	}
}

func TestApiKeyManagementRequiresAdmin(t *testing.T) {
	f := CreateFixture(t)
	admin, _ := f.CreateAdmin("admin@example.com")
	user, _ := f.CreateUser("user@example.com")
	created, _ := f.CreateApiKey(admin, "test", "read")

	if _, rr := f.CreateApiKey(user, "test", "read"); rr.Code != http.StatusForbidden {
		t.Errorf("Create: expected 403, got %d", rr.Code)
	}
	if rr := f.request("GET", "/api/api-key", nil, user, nil); rr.Code != http.StatusForbidden {
		t.Errorf("List: expected 403, got %d", rr.Code)
	}
	if rr := f.DeleteApiKey(user, created.ApiKey.Id); rr.Code != http.StatusForbidden {
		t.Errorf("Delete: expected 403, got %d", rr.Code)
	}
	if rr := f.request("GET", "/api/api-key", nil, nil, nil); rr.Code != http.StatusForbidden {
		t.Errorf("List without session: expected 403, got %d", rr.Code)
	}
	if len(f.ListApiKeys(admin)) != 1 {
		t.Errorf("Expected the key to remain")
	}
}

func TestApiKeysCantManageApiKeys(t *testing.T) {
	f := CreateFixture(t)
	admin, _ := f.CreateAdmin("admin@example.com")
	created, _ := f.CreateApiKey(admin, "test", "write")

	rr := f.publicRequest("GET", "/api/api-key", created.Token, nil, nil)
	if rr.Code != http.StatusForbidden {
		t.Errorf("Expected 403, got %d", rr.Code)
	}
}

func TestDeleteApiKey(t *testing.T) {
	f := CreateFixture(t)
	admin, _ := f.CreateAdmin("admin@example.com")
	created, _ := f.CreateApiKey(admin, "test", "read")

	if rr := f.DeleteApiKey(admin, created.ApiKey.Id); rr.Code != http.StatusNoContent {
		t.Fatalf("Expected 204, got %d", rr.Code)
	}
	if rr := f.DeleteApiKey(admin, created.ApiKey.Id); rr.Code != http.StatusNotFound {
		t.Errorf("Expected 404 on second delete, got %d", rr.Code)
	}
	if keys := f.ListApiKeys(admin); len(keys) != 0 {
		t.Errorf("Expected no keys, got %+v", keys)
	}
	rr := f.publicRequest("GET", "/api/v1/mqtt/profiles", created.Token, nil, nil)
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("Deleted key: expected 401, got %d", rr.Code)
	}
}
