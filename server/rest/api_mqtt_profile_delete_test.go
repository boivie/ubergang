package rest

import (
	"boivie/ubergang/server/api"
	"net/http"
	"testing"
)

func TestDeleteMqttProfile(t *testing.T) {
	f, cookie := setupMqttProfileTest(t)

	f.CreateMqttProfile(cookie, &api.ApiMqttProfile{Id: "test-profile"})

	rr := f.DeleteMqttProfile(cookie, "test-profile")
	if rr.Code != http.StatusNoContent {
		t.Fatalf("request failed with status %d: %s", rr.Code, rr.Body.String())
	}

	profiles := f.ListMqttProfiles(cookie)
	if len(profiles) != 0 {
		t.Fatalf("Expected 0 profiles, got %d", len(profiles))
	}
}
