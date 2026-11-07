package rest

import (
	"boivie/ubergang/server/api"
	"boivie/ubergang/server/models"
	"boivie/ubergang/server/mqtt"
	"net/http"
	"sort"
)

// toMqttClient converts `c` for the internal API. The password is only
// included if `includePassword` is set, which must only be done for admins.
func toMqttClient(c *models.MqttClient, connState *mqtt.ClientConnectionState, includePassword bool) api.ApiMqttClient {
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

	ret := api.ApiMqttClient{
		Id:           c.Id,
		ProfileId:    c.ProfileId,
		Values:       c.Values,
		Connected:    connected,
		Disconnected: disconnected,
	}
	if includePassword {
		ret.Password = c.Password
	}
	return ret
}

func (s *ApiModule) handleMqttClientList(w http.ResponseWriter, r *http.Request) {
	user, _, err := s.session.GetAndValidate(w, r)
	if err != nil {
		return
	}

	connectedClients := s.mqttProxy.GetActiveConnections()

	clients := make([]api.ApiMqttClient, 0)
	for _, c := range s.db.ListMqttClients() {
		clients = append(clients, toMqttClient(c, connectedClients[c.Id], user.IsAdmin))
	}
	sort.Slice(clients, func(i, j int) bool {
		return clients[i].Id < clients[j].Id
	})

	jsonify(w, api.ApiListMqttClientsResponse{
		MqttClients: clients,
	})
}
