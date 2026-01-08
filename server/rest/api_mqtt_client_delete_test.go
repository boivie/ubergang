package rest

import (
	"boivie/ubergang/server/api"
	"net/http"
	"testing"
)

func TestDeleteMqttClient(t *testing.T) {
	f, cookie := setupMqttClientTest(t)

	f.CreateMqttProfile(cookie, &api.ApiMqttProfile{Id: "test-profile"})
	f.CreateMqttClient(cookie, &api.ApiMqttClient{Id: "test-client", ProfileId: "test-profile"})

	rr := f.DeleteMqttClient(cookie, "test-client")
	if rr.Code != http.StatusNoContent {
		t.Fatalf("request failed with status %d: %s", rr.Code, rr.Body.String())
	}

	clients := f.ListMqttClients(cookie)
	if len(clients) != 0 {
		t.Fatalf("Expected 0 clients, got %d", len(clients))
	}
}
