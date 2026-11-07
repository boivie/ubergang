package rest

import (
	"boivie/ubergang/server/api"
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"
)

type publicApiFixture struct {
	*Fixture
	admin      *http.Cookie
	readToken  string
	writeToken string
}

func setupPublicApiTest(t *testing.T) *publicApiFixture {
	f := CreateFixture(t)
	admin, _ := f.CreateAdmin("admin@example.com")
	read, _ := f.CreateApiKey(admin, "reader", "read")
	write, _ := f.CreateApiKey(admin, "writer", "write")
	return &publicApiFixture{f, admin, read.Token, write.Token}
}

func expectPublicError(t *testing.T, rr *httptest.ResponseRecorder, status int, code string) {
	t.Helper()
	if rr.Code != status {
		t.Fatalf("Expected status %d, got %d: %s", status, rr.Code, rr.Body.String())
	}
	if ct := rr.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("Expected JSON error, got Content-Type %q", ct)
	}
	var e api.PublicError
	if err := json.NewDecoder(rr.Body).Decode(&e); err != nil {
		t.Fatalf("Failed to decode error: %v", err)
	}
	if e.Error.Code != code {
		t.Errorf("Expected error code %q, got %q (%s)", code, e.Error.Code, e.Error.Message)
	}
}

func (f *publicApiFixture) createProfile(t *testing.T, id string) {
	t.Helper()
	rr := f.publicRequest("POST", "/api/v1/mqtt/profiles", f.writeToken,
		&api.PublicCreateMqttProfileRequest{Id: id, AllowPublish: []string{"a/#"}}, nil)
	if rr.Code != http.StatusCreated {
		t.Fatalf("Failed to create profile: %d %s", rr.Code, rr.Body.String())
	}
}

func (f *publicApiFixture) createClient(t *testing.T, id, profileId string) {
	t.Helper()
	rr := f.publicRequest("POST", "/api/v1/mqtt/clients", f.writeToken,
		&api.PublicCreateMqttClientRequest{Id: id, ProfileId: profileId, Password: "secret"}, nil)
	if rr.Code != http.StatusCreated {
		t.Fatalf("Failed to create client: %d %s", rr.Code, rr.Body.String())
	}
}

// Authentication

func TestPublicApiRequiresApiKey(t *testing.T) {
	f := setupPublicApiTest(t)

	rr := f.publicRequest("GET", "/api/v1/mqtt/profiles", "", nil, nil)
	expectPublicError(t, rr, http.StatusUnauthorized, "unauthorized")
	if !strings.HasPrefix(rr.Header().Get("WWW-Authenticate"), "Bearer") {
		t.Errorf("Expected WWW-Authenticate header, got %q", rr.Header().Get("WWW-Authenticate"))
	}

	for _, token := range []string{
		"garbage",
		"ug_",
		"ug_nosecret",
		"ug_unknownid_secret",
		f.readToken + "x",
		f.readToken[:strings.LastIndex(f.readToken, "_")+1] + "wrongsecret",
	} {
		rr := f.publicRequest("GET", "/api/v1/mqtt/profiles", token, nil, nil)
		expectPublicError(t, rr, http.StatusUnauthorized, "unauthorized")
	}
}

func TestPublicApiAcceptsOtherBearerCasing(t *testing.T) {
	f := setupPublicApiTest(t)
	req, _ := http.NewRequest("GET", "/api/v1/mqtt/profiles", nil)
	req.Host = "test.example.com"
	req.Header.Set("Authorization", "bearer "+f.readToken)
	rr := httptest.NewRecorder()
	f.router.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("Expected 200, got %d", rr.Code)
	}
}

func TestPublicApiDoesNotAcceptSessionCookie(t *testing.T) {
	f := setupPublicApiTest(t)
	rr := f.request("GET", "/api/v1/mqtt/profiles", nil, f.admin, nil)
	expectPublicError(t, rr, http.StatusUnauthorized, "unauthorized")
}

func TestApiKeyNotAcceptedOnInternalApi(t *testing.T) {
	f := setupPublicApiTest(t)
	rr := f.publicRequest("GET", "/api/mqtt-profile", f.writeToken, nil, nil)
	if rr.Code != http.StatusForbidden {
		t.Errorf("Expected 403, got %d", rr.Code)
	}
}

func TestPublicApiReadKeyCantWrite(t *testing.T) {
	f := setupPublicApiTest(t)
	f.createProfile(t, "p")
	f.createClient(t, "c", "p")

	for _, tc := range []struct {
		method, url string
		body        interface{}
	}{
		{"POST", "/api/v1/mqtt/profiles", &api.PublicCreateMqttProfileRequest{Id: "p2"}},
		{"PATCH", "/api/v1/mqtt/profiles/p", &api.PublicUpdateMqttProfileRequest{}},
		{"DELETE", "/api/v1/mqtt/profiles/p", nil},
		{"POST", "/api/v1/mqtt/clients", &api.PublicCreateMqttClientRequest{Id: "c2", ProfileId: "p", Password: "x"}},
		{"PATCH", "/api/v1/mqtt/clients/c", &api.PublicUpdateMqttClientRequest{}},
		{"DELETE", "/api/v1/mqtt/clients/c", nil},
	} {
		rr := f.publicRequest(tc.method, tc.url, f.readToken, tc.body, nil)
		expectPublicError(t, rr, http.StatusForbidden, "insufficient_scope")
	}

	if _, err := f.Db.GetMqttProfile("p2"); err == nil {
		t.Errorf("Profile was created by a read-only key")
	}
	if _, err := f.Db.GetMqttClient("c"); err != nil {
		t.Errorf("Client was deleted by a read-only key")
	}
}

func TestPublicApiUnknownEndpoint(t *testing.T) {
	f := setupPublicApiTest(t)
	rr := f.publicRequest("GET", "/api/v1/nope", f.readToken, nil, nil)
	expectPublicError(t, rr, http.StatusNotFound, "not_found")

	rr = f.publicRequest("PUT", "/api/v1/mqtt/profiles", f.writeToken, nil, nil)
	expectPublicError(t, rr, http.StatusNotFound, "not_found")
}

// Request parsing

func TestPublicApiRejectsBadBodies(t *testing.T) {
	f := setupPublicApiTest(t)

	send := func(contentType, body string) *httptest.ResponseRecorder {
		req, _ := http.NewRequest("POST", "/api/v1/mqtt/profiles", bytes.NewBufferString(body))
		req.Host = "test.example.com"
		req.Header.Set("Authorization", "Bearer "+f.writeToken)
		if contentType != "" {
			req.Header.Set("Content-Type", contentType)
		}
		rr := httptest.NewRecorder()
		f.router.ServeHTTP(rr, req)
		return rr
	}

	expectPublicError(t, send("", `{"id":"p"}`), http.StatusUnsupportedMediaType, "unsupported_media_type")
	expectPublicError(t, send("text/plain", `{"id":"p"}`), http.StatusUnsupportedMediaType, "unsupported_media_type")
	expectPublicError(t, send("application/json", `{"id":`), http.StatusBadRequest, "invalid_request")
	expectPublicError(t, send("application/json", `{"id":"p","allow_publsh":[]}`), http.StatusBadRequest, "invalid_request")
	expectPublicError(t, send("application/json", `{"id":"p"} {}`), http.StatusBadRequest, "invalid_request")

	if rr := send("application/json; charset=utf-8", `{"id":"p"}`); rr.Code != http.StatusCreated {
		t.Errorf("Expected 201 with charset, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestPublicApiValidatesIds(t *testing.T) {
	f := setupPublicApiTest(t)
	for _, id := range []string{"", "a/b", " a", "a\n", strings.Repeat("a", 257)} {
		rr := f.publicRequest("POST", "/api/v1/mqtt/profiles", f.writeToken,
			&api.PublicCreateMqttProfileRequest{Id: id}, nil)
		expectPublicError(t, rr, http.StatusBadRequest, "invalid_request")
	}
}

// MQTT profiles

func TestPublicMqttProfileCrud(t *testing.T) {
	f := setupPublicApiTest(t)

	var created api.PublicMqttProfile
	rr := f.publicRequest("POST", "/api/v1/mqtt/profiles", f.writeToken,
		&api.PublicCreateMqttProfileRequest{Id: "sensors", AllowPublish: []string{"sensors/#"}}, &created)
	if rr.Code != http.StatusCreated {
		t.Fatalf("Expected 201, got %d: %s", rr.Code, rr.Body.String())
	}
	if loc := rr.Header().Get("Location"); loc != "/api/v1/mqtt/profiles/sensors" {
		t.Errorf("Unexpected Location: %q", loc)
	}
	if !slices.Equal(created.AllowPublish, []string{"sensors/#"}) || created.AllowSubscribe == nil {
		t.Errorf("Unexpected profile: %+v", created)
	}

	rr = f.publicRequest("POST", "/api/v1/mqtt/profiles", f.writeToken,
		&api.PublicCreateMqttProfileRequest{Id: "sensors"}, nil)
	expectPublicError(t, rr, http.StatusConflict, "conflict")

	var got api.PublicMqttProfile
	if rr := f.publicRequest("GET", "/api/v1/mqtt/profiles/sensors", f.readToken, nil, &got); rr.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d", rr.Code)
	}
	if got.Id != "sensors" || !slices.Equal(got.AllowPublish, []string{"sensors/#"}) {
		t.Errorf("Unexpected profile: %+v", got)
	}

	// Only the given fields are changed.
	var updated api.PublicMqttProfile
	subs := []string{"commands/#"}
	rr = f.publicRequest("PATCH", "/api/v1/mqtt/profiles/sensors", f.writeToken,
		&api.PublicUpdateMqttProfileRequest{AllowSubscribe: &subs}, &updated)
	if rr.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	if !slices.Equal(updated.AllowPublish, []string{"sensors/#"}) || !slices.Equal(updated.AllowSubscribe, subs) {
		t.Errorf("Unexpected profile: %+v", updated)
	}

	var list api.PublicListMqttProfilesResponse
	f.publicRequest("GET", "/api/v1/mqtt/profiles", f.readToken, nil, &list)
	if len(list.Profiles) != 1 || list.Profiles[0].Id != "sensors" {
		t.Errorf("Unexpected list: %+v", list)
	}

	if rr := f.publicRequest("DELETE", "/api/v1/mqtt/profiles/sensors", f.writeToken, nil, nil); rr.Code != http.StatusNoContent {
		t.Fatalf("Expected 204, got %d", rr.Code)
	}
	rr = f.publicRequest("GET", "/api/v1/mqtt/profiles/sensors", f.readToken, nil, nil)
	expectPublicError(t, rr, http.StatusNotFound, "not_found")
}

func TestPublicMqttProfileNotFound(t *testing.T) {
	f := setupPublicApiTest(t)
	rr := f.publicRequest("PATCH", "/api/v1/mqtt/profiles/nope", f.writeToken, &api.PublicUpdateMqttProfileRequest{}, nil)
	expectPublicError(t, rr, http.StatusNotFound, "not_found")
	rr = f.publicRequest("DELETE", "/api/v1/mqtt/profiles/nope", f.writeToken, nil, nil)
	expectPublicError(t, rr, http.StatusNotFound, "not_found")
	if _, err := f.Db.GetMqttProfile("nope"); err == nil {
		t.Errorf("PATCH created a profile")
	}
}

func TestPublicMqttProfileDeleteInUse(t *testing.T) {
	f := setupPublicApiTest(t)
	f.createProfile(t, "p")
	f.createClient(t, "c", "p")

	rr := f.publicRequest("DELETE", "/api/v1/mqtt/profiles/p", f.writeToken, nil, nil)
	expectPublicError(t, rr, http.StatusConflict, "conflict")
}

func TestPublicMqttProfileUpdateDisconnectsClients(t *testing.T) {
	f := setupPublicApiTest(t)
	f.createProfile(t, "p")
	f.createProfile(t, "other")
	f.createClient(t, "c1", "p")
	f.createClient(t, "c2", "other")

	pubs := []string{}
	f.publicRequest("PATCH", "/api/v1/mqtt/profiles/p", f.writeToken,
		&api.PublicUpdateMqttProfileRequest{AllowPublish: &pubs}, nil)
	if !slices.Equal(f.Mqtt.Disconnected, []string{"c1"}) {
		t.Errorf("Unexpected disconnects: %v", f.Mqtt.Disconnected)
	}
}

// MQTT clients

func TestPublicMqttClientCrud(t *testing.T) {
	f := setupPublicApiTest(t)
	f.createProfile(t, "p1")
	f.createProfile(t, "p2")

	var created api.PublicMqttClient
	rr := f.publicRequest("POST", "/api/v1/mqtt/clients", f.writeToken, &api.PublicCreateMqttClientRequest{
		Id: "dev1", ProfileId: "p1", Password: "hunter2", Values: map[string]string{"room": "kitchen"},
	}, &created)
	if rr.Code != http.StatusCreated {
		t.Fatalf("Expected 201, got %d: %s", rr.Code, rr.Body.String())
	}
	if loc := rr.Header().Get("Location"); loc != "/api/v1/mqtt/clients/dev1" {
		t.Errorf("Unexpected Location: %q", loc)
	}
	if created.Password != "hunter2" || created.Values["room"] != "kitchen" {
		t.Errorf("Unexpected client: %+v", created)
	}

	rr = f.publicRequest("POST", "/api/v1/mqtt/clients", f.writeToken, &api.PublicCreateMqttClientRequest{
		Id: "dev1", ProfileId: "p1", Password: "x",
	}, nil)
	expectPublicError(t, rr, http.StatusConflict, "conflict")

	// Only the given fields are changed.
	var updated api.PublicMqttClient
	p2 := "p2"
	newPassword := "correct horse"
	rr = f.publicRequest("PATCH", "/api/v1/mqtt/clients/dev1", f.writeToken,
		&api.PublicUpdateMqttClientRequest{ProfileId: &p2, Password: &newPassword}, &updated)
	if rr.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	if updated.ProfileId != "p2" || updated.Password != newPassword || updated.Values["room"] != "kitchen" {
		t.Errorf("Unexpected client: %+v", updated)
	}
	stored, _ := f.Db.GetMqttClient("dev1")
	if stored.Password != newPassword {
		t.Errorf("Password not stored")
	}

	if rr := f.publicRequest("DELETE", "/api/v1/mqtt/clients/dev1", f.writeToken, nil, nil); rr.Code != http.StatusNoContent {
		t.Fatalf("Expected 204, got %d", rr.Code)
	}
	rr = f.publicRequest("GET", "/api/v1/mqtt/clients/dev1", f.readToken, nil, nil)
	expectPublicError(t, rr, http.StatusNotFound, "not_found")
	rr = f.publicRequest("DELETE", "/api/v1/mqtt/clients/dev1", f.writeToken, nil, nil)
	expectPublicError(t, rr, http.StatusNotFound, "not_found")
}

func TestPublicMqttClientPasswordRequiresWriteScope(t *testing.T) {
	f := setupPublicApiTest(t)
	f.createProfile(t, "p")
	f.createClient(t, "c", "p")

	var c api.PublicMqttClient
	f.publicRequest("GET", "/api/v1/mqtt/clients/c", f.readToken, nil, &c)
	if c.Id != "c" || c.Password != "" {
		t.Errorf("Read key: unexpected client %+v", c)
	}
	var list api.PublicListMqttClientsResponse
	rr := f.publicRequest("GET", "/api/v1/mqtt/clients", f.readToken, nil, &list)
	if len(list.Clients) != 1 || list.Clients[0].Password != "" || strings.Contains(rr.Body.String(), "secret") {
		t.Errorf("Read key: unexpected list %s", rr.Body.String())
	}

	f.publicRequest("GET", "/api/v1/mqtt/clients/c", f.writeToken, nil, &c)
	if c.Password != "secret" {
		t.Errorf("Write key: expected password, got %+v", c)
	}
	list = api.PublicListMqttClientsResponse{}
	f.publicRequest("GET", "/api/v1/mqtt/clients", f.writeToken, nil, &list)
	if len(list.Clients) != 1 || list.Clients[0].Password != "secret" {
		t.Errorf("Write key: unexpected list %+v", list)
	}
}

func TestPublicMqttClientValidation(t *testing.T) {
	f := setupPublicApiTest(t)
	f.createProfile(t, "p")

	for _, req := range []api.PublicCreateMqttClientRequest{
		{Id: "", ProfileId: "p", Password: "x"},
		{Id: "a/b", ProfileId: "p", Password: "x"},
		{Id: "c", ProfileId: "", Password: "x"},
		{Id: "c", ProfileId: "p", Password: ""},
	} {
		rr := f.publicRequest("POST", "/api/v1/mqtt/clients", f.writeToken, &req, nil)
		expectPublicError(t, rr, http.StatusBadRequest, "invalid_request")
	}

	rr := f.publicRequest("POST", "/api/v1/mqtt/clients", f.writeToken,
		&api.PublicCreateMqttClientRequest{Id: "c", ProfileId: "nope", Password: "x"}, nil)
	expectPublicError(t, rr, http.StatusUnprocessableEntity, "profile_not_found")

	f.createClient(t, "c", "p")
	empty := ""
	nope := "nope"
	rr = f.publicRequest("PATCH", "/api/v1/mqtt/clients/c", f.writeToken,
		&api.PublicUpdateMqttClientRequest{Password: &empty}, nil)
	expectPublicError(t, rr, http.StatusBadRequest, "invalid_request")
	rr = f.publicRequest("PATCH", "/api/v1/mqtt/clients/c", f.writeToken,
		&api.PublicUpdateMqttClientRequest{ProfileId: &nope}, nil)
	expectPublicError(t, rr, http.StatusUnprocessableEntity, "profile_not_found")
	// Renaming isn't possible.
	rr = f.publicRequest("PATCH", "/api/v1/mqtt/clients/c", f.writeToken, map[string]string{"id": "d"}, nil)
	expectPublicError(t, rr, http.StatusBadRequest, "invalid_request")
	rr = f.publicRequest("PATCH", "/api/v1/mqtt/clients/nope", f.writeToken, &api.PublicUpdateMqttClientRequest{}, nil)
	expectPublicError(t, rr, http.StatusNotFound, "not_found")

	stored, _ := f.Db.GetMqttClient("c")
	if stored.ProfileId != "p" || stored.Password != "secret" {
		t.Errorf("Client changed by failed updates: %+v", stored)
	}
}

func TestPublicMqttClientDisconnects(t *testing.T) {
	f := setupPublicApiTest(t)
	f.createProfile(t, "p")
	f.createClient(t, "c", "p")

	values := map[string]string{"a": "b"}
	f.publicRequest("PATCH", "/api/v1/mqtt/clients/c", f.writeToken,
		&api.PublicUpdateMqttClientRequest{Values: &values}, nil)
	if len(f.Mqtt.Disconnected) != 0 {
		t.Errorf("Changing values shouldn't disconnect, got %v", f.Mqtt.Disconnected)
	}

	password := "new"
	f.publicRequest("PATCH", "/api/v1/mqtt/clients/c", f.writeToken,
		&api.PublicUpdateMqttClientRequest{Password: &password}, nil)
	f.publicRequest("DELETE", "/api/v1/mqtt/clients/c", f.writeToken, nil, nil)
	if !slices.Equal(f.Mqtt.Disconnected, []string{"c", "c"}) {
		t.Errorf("Unexpected disconnects: %v", f.Mqtt.Disconnected)
	}
}
