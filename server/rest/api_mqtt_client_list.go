package rest

import (
	"boivie/ubergang/server/api"
	"boivie/ubergang/server/models"
	"boivie/ubergang/server/mqtt"
	"net/http"
	"sort"
)

func toMqttClient(c *models.MqttClient, connState *mqtt.ClientConnectionState) api.ApiMqttClient {
	var connected *api.ApiMqttConnected = nil
	var disconnected *api.ApiMqttDisconnected = nil

	if connState != nil && connState.Connected != nil {
		connected = &api.ApiMqttConnected{
			RemoteAddr:     connState.Connected.RemoteAddr,
			ConnectedAt:    connState.Connected.ConnectedAt.Format("2006-01-02 15:04:05"),
			ConnectionType: string(connState.Connected.ConnectionType),
		}
	} else if connState != nil && connState.Disconnected != nil {
		disconnected = &api.ApiMqttDisconnected{
			RemoteAddr:     connState.Disconnected.RemoteAddr,
			DisconnectedAt: connState.Disconnected.DisconnectedAt.Format("2006-01-02 15:04:05"),
		}
	}

	return api.ApiMqttClient{
		Id:           c.Id,
		ProfileId:    c.ProfileId,
		Password:     c.Password,
		Values:       c.Values,
		Connected:    connected,
		Disconnected: disconnected,
	}
}

func (s *ApiModule) handleMqttClientList(w http.ResponseWriter, r *http.Request) {
	_, _, err := s.session.GetAndValidate(w, r)
	if err != nil {
		return
	}

	connectedClients := s.mqttProxy.GetActiveConnections()

	clients := make([]api.ApiMqttClient, 0)
	for _, c := range s.db.ListMqttClients() {
		clients = append(clients, toMqttClient(c, connectedClients[c.Id]))
	}
	sort.Slice(clients, func(i, j int) bool {
		return clients[i].Id < clients[j].Id
	})

	jsonify(w, api.ApiListMqttClientsResponse{
		MqttClients: clients,
	})
}
