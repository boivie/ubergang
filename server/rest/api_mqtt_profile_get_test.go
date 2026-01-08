package rest

import (
	"boivie/ubergang/server/api"
	"testing"
)

func TestGetMqttProfile(t *testing.T) {
	f, cookie := setupMqttProfileTest(t)

	f.CreateMqttProfile(cookie, &api.ApiMqttProfile{
		Id:           "test-profile",
		AllowPublish: []string{"a/b"},
	})

	profile := f.GetMqttProfile(cookie, "test-profile")
	if profile.Id != "test-profile" {
		t.Errorf("Expected profile id to be 'test-profile', got %s", profile.Id)
	}
	if len(profile.AllowPublish) != 1 || profile.AllowPublish[0] != "a/b" {
		t.Errorf("Unexpected AllowPublish: %v", profile.AllowPublish)
	}
}
