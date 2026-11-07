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

func TestMqttClientPasswordOnlyShownToAdmins(t *testing.T) {
	f, admin := setupMqttClientTest(t)
	user, _ := f.CreateUser("user@example.com")

	f.CreateMqttProfile(admin, &api.ApiMqttProfile{Id: "test-profile"})
	f.CreateMqttClient(admin, &api.ApiMqttClient{
		Id:        "test-client",
		ProfileId: "test-profile",
		Password:  "hunter2",
	})

	if c := f.GetMqttClient(admin, "test-client"); c.Password != "hunter2" {
		t.Errorf("Admin: expected password, got %q", c.Password)
	}
	if c := f.ListMqttClients(admin); len(c) != 1 || c[0].Password != "hunter2" {
		t.Errorf("Admin: expected password in list, got %+v", c)
	}

	c := f.GetMqttClient(user, "test-client")
	if c.Id != "test-client" || c.Password != "" {
		t.Errorf("User: unexpected client %+v", c)
	}
	list := f.ListMqttClients(user)
	if len(list) != 1 || list[0].Password != "" {
		t.Errorf("User: unexpected list %+v", list)
	}
}
