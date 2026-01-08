package rest

import (
	"boivie/ubergang/server/api"
	"testing"
)

func TestListMqttClient(t *testing.T) {
	f, cookie := setupMqttClientTest(t)

	f.CreateMqttProfile(cookie, &api.ApiMqttProfile{Id: "test-profile"})
	f.CreateMqttClient(cookie, &api.ApiMqttClient{Id: "client1", ProfileId: "test-profile"})
	f.CreateMqttClient(cookie, &api.ApiMqttClient{Id: "client2", ProfileId: "test-profile"})

	clients := f.ListMqttClients(cookie)
	if len(clients) != 2 {
		t.Fatalf("Expected 2 clients, got %d", len(clients))
	}
	if clients[0].Id != "client1" {
		t.Errorf("Expected client1, got %s", clients[0].Id)
	}
	if clients[1].Id != "client2" {
		t.Errorf("Expected client2, got %s", clients[1].Id)
	}
}
