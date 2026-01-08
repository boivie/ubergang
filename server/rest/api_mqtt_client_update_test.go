package rest

import (
	"boivie/ubergang/server/api"
	"net/http"
	"testing"
)

func setupMqttClientTest(t *testing.T) (*Fixture, *http.Cookie) {
	t.Helper()
	f := CreateFixture(t)
	cookie, _ := f.CreateAdmin("test")
	return f, cookie
}

func TestUpdateMqttClient(t *testing.T) {
	t.Run("create client", func(t *testing.T) {
		f, cookie := setupMqttClientTest(t)

		f.CreateMqttProfile(cookie, &api.ApiMqttProfile{Id: "test-profile"})
		rr := f.CreateMqttClient(cookie, &api.ApiMqttClient{
			Id:        "test-client",
			ProfileId: "test-profile",
		})

		if rr.Code != http.StatusOK {
			t.Fatalf("request failed with status %d: %s", rr.Code, rr.Body.String())
		}

		clients := f.ListMqttClients(cookie)
		if len(clients) != 1 {
			t.Fatalf("Expected 1 client, got %d", len(clients))
		}
		if clients[0].Id != "test-client" {
			t.Errorf("Expected id to be 'test-client', got %q", clients[0].Id)
		}
	})

	t.Run("update client", func(t *testing.T) {
		f, cookie := setupMqttClientTest(t)
		f.CreateMqttProfile(cookie, &api.ApiMqttProfile{Id: "test-profile"})
		f.CreateMqttProfile(cookie, &api.ApiMqttProfile{Id: "new-profile"})

		f.CreateMqttClient(cookie, &api.ApiMqttClient{
			Id: "test-client",
		})

		profileId := "new-profile"
		password := "new-password"
		values := map[string]string{"a": "b"}
		req := &api.ApiUpdateMqttClientRequest{
			ProfileId: &profileId,
			Password:  &password,
			Values:    &values,
		}
		resp := &api.ApiUpdateMqttClientResponse{}
		rr := f.request("POST", "/api/mqtt-client/test-client", req, cookie, resp)

		if rr.Code != http.StatusOK {
			t.Fatalf("request failed with status %d: %s", rr.Code, rr.Body.String())
		}

		client := f.GetMqttClient(cookie, "test-client")
		if client.ProfileId != "new-profile" {
			t.Errorf("Expected ProfileId to be updated, got %v", client.ProfileId)
		}
		if client.Password != "new-password" {
			t.Errorf("Expected Password to be updated, got %v", client.Password)
		}
		if client.Values["a"] != "b" {
			t.Errorf("Expected Values to be updated, got %v", client.Values)
		}
	})
}
