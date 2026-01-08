package rest

import (
	"boivie/ubergang/server/api"
	"testing"
)

func TestListMqttProfile(t *testing.T) {
	f, cookie := setupMqttProfileTest(t)

	f.CreateMqttProfile(cookie, &api.ApiMqttProfile{Id: "profile1"})
	f.CreateMqttProfile(cookie, &api.ApiMqttProfile{Id: "profile2"})

	profiles := f.ListMqttProfiles(cookie)
	if len(profiles) != 2 {
		t.Fatalf("Expected 2 profiles, got %d", len(profiles))
	}
	if profiles[0].Id != "profile1" {
		t.Errorf("Expected profile1, got %s", profiles[0].Id)
	}
	if profiles[1].Id != "profile2" {
		t.Errorf("Expected profile2, got %s", profiles[1].Id)
	}
}
