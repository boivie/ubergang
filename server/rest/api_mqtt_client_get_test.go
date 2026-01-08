package rest

import (
	"boivie/ubergang/server/api"
	"testing"
)

func TestGetMqttClient(t *testing.T) {
	f, cookie := setupMqttClientTest(t)

	f.CreateMqttProfile(cookie, &api.ApiMqttProfile{Id: "test-profile"})
	f.CreateMqttClient(cookie, &api.ApiMqttClient{
		Id:        "test-client",
		ProfileId: "test-profile",
	})

	client := f.GetMqttClient(cookie, "test-client")
	if client.Id != "test-client" {
		t.Errorf("Expected client id to be 'test-client', got %s", client.Id)
	}
	if client.ProfileId != "test-profile" {
		t.Errorf("Unexpected ProfileId: %v", client.ProfileId)
	}
}
