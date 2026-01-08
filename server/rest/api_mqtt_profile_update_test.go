package rest

import (
	"boivie/ubergang/server/api"
	"net/http"
	"testing"
)

func setupMqttProfileTest(t *testing.T) (*Fixture, *http.Cookie) {
	t.Helper()
	f := CreateFixture(t)
	cookie, _ := f.CreateAdmin("test")
	return f, cookie
}

func TestUpdateMqttProfile(t *testing.T) {
	t.Run("create profile", func(t *testing.T) {
		f, cookie := setupMqttProfileTest(t)

		rr := f.CreateMqttProfile(cookie, &api.ApiMqttProfile{
			Id:             "test-profile",
			AllowPublish:   []string{"a/b"},
			AllowSubscribe: []string{"c/d"},
		})

		if rr.Code != http.StatusOK {
			t.Fatalf("request failed with status %d: %s", rr.Code, rr.Body.String())
		}

		profiles := f.ListMqttProfiles(cookie)
		if len(profiles) != 1 {
			t.Fatalf("Expected 1 profile, got %d", len(profiles))
		}
		if profiles[0].Id != "test-profile" {
			t.Errorf("Expected id to be 'test-profile', got %q", profiles[0].Id)
		}
	})

	t.Run("update profile", func(t *testing.T) {
		f, cookie := setupMqttProfileTest(t)
		f.CreateMqttProfile(cookie, &api.ApiMqttProfile{
			Id: "test-profile",
		})

		allowPublish := []string{"e/f"}
		allowSubscribe := []string{"g/h"}
		req := &api.ApiUpdateMqttProfileRequest{
			AllowPublish:   &allowPublish,
			AllowSubscribe: &allowSubscribe,
		}
		resp := &api.ApiUpdateMqttProfileResponse{}
		rr := f.request("POST", "/api/mqtt-profile/test-profile", req, cookie, resp)

		if rr.Code != http.StatusOK {
			t.Fatalf("request failed with status %d: %s", rr.Code, rr.Body.String())
		}

		profile := f.GetMqttProfile(cookie, "test-profile")
		if len(profile.AllowPublish) != 1 || profile.AllowPublish[0] != "e/f" {
			t.Errorf("Expected AllowPublish to be updated, got %v", profile.AllowPublish)
		}
		if len(profile.AllowSubscribe) != 1 || profile.AllowSubscribe[0] != "g/h" {
			t.Errorf("Expected AllowSubscribe to be updated, got %v", profile.AllowSubscribe)
		}
	})
}
